package internal

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/modules"
	loggingapi "github.com/hiveot/hivekit/go/modules/logging/api"
	"github.com/hiveot/hivekit/go/modules/logging/config"
	"github.com/hiveot/hivekit/go/utils"
)

// LoggingService is a module for writing request, response and notification messages to a log output.
// The module is configured using yaml.
type LoggingService struct {
	modules.HiveModuleBase

	// configuration. Allow manual configuration
	Config config.LoggingConfig

	// log destination for notifications
	notificationLogger *slog.Logger
	// log destination for notifications
	requestLogger *slog.Logger

	// handler to release log resources
	releaseFn func()
}

// log notifications upstream and logs them if they pass the filter
func (m *LoggingService) HandleNotification(notif *msg.NotificationMessage) {
	go func() {
		if m.Config.NotificationFilter.AcceptNotification(notif) {
			m.LogNotification(notif)
		}
	}()
	m.ForwardNotification(notif)
}

// HandleRequest forwards requests downstream and logs them if they pass the filter
func (m *LoggingService) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	go func() {
		if m.Config.NotificationFilter.AcceptRequest(req) {
			m.LogRequest(req)
		}
	}()
	return m.ForwardRequest(req, replyTo)
}

// write notifications to the logging backend
func (m *LoggingService) LogNotification(notif *msg.NotificationMessage) {
	value := utils.DecodeAsString(notif.Data, 32)
	m.notificationLogger.Info("Notification",
		slog.String("type", string(notif.AffordanceType)),
		slog.String("thingID", notif.ThingID),
		slog.String("name", notif.Name),
		slog.String("value", value),
		slog.String("timestamp", notif.Timestamp),
	)

}

// write request to the logging backend
func (m *LoggingService) LogRequest(req *msg.RequestMessage) {
	value := utils.DecodeAsString(req.Input, 32)
	m.requestLogger.Info("Request",
		slog.String("op", string(req.Operation)),
		slog.String("thingID", req.ThingID),
		slog.String("name", req.Name),
		slog.String("value", value),
		slog.String("created", req.Timestamp),
		slog.String("sender", string(req.SenderID)),
	)

}

// NewLogger returns a new instance of a logger using the given backend along with
// a function to release resources.
func (m *LoggingService) NewLogger(cfg *config.LoggingConfig) (
	logger *slog.Logger, releaseFn func()) {

	var logFile *os.File
	var logWriter io.Writer
	var err error

	if cfg.Backend == loggingapi.LoggingBackendFile {
		// ensure the directory exists
		logDir := filepath.Dir(cfg.LogDestination)
		_ = os.MkdirAll(logDir, 0750)

		logFile, err = os.OpenFile(cfg.LogDestination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			// fallback to stdout
			logWriter = os.Stdout
			slog.Error("NewLogger: Unable to open logfile",
				"destination", cfg.LogDestination, "err", err.Error())
		} else if cfg.Log2Stdout {
			// log to both stdout and to file
			logWriter = io.MultiWriter(os.Stdout, logFile)
		} else {
			// log only to file
			logWriter = logFile
		}
		if logFile != nil {
			releaseFn = func() {
				logFile.Close()
			}
		}
	} else {
		// default to stdout
		logWriter = os.Stdout
	}
	// todo: config text vs json
	if cfg.LogAsJson {
		handler := slog.NewJSONHandler(logWriter, &slog.HandlerOptions{})
		logger = slog.New(handler)
	} else {
		handler := slog.NewTextHandler(logWriter, &slog.HandlerOptions{})
		logger = slog.New(handler)
	}

	return logger, releaseFn
}

// SetSource is a convenience function to set the source module of requests and destination of notifications
func (m *LoggingService) SetSource(source modules.IHiveModule) {
	source.SetRequestSink(m.HandleRequest)
	m.SetNotificationSink(source.HandleNotification)
}

// SetSink is a convenience function to set the downstream module of requests and source of notifications
func (m *LoggingService) SetSink(sink modules.IHiveModule) {
	m.SetRequestSink(sink.HandleRequest)
	sink.SetNotificationSink(m.HandleNotification)
}

// Start opens the logging destination.
func (m *LoggingService) Start() (err error) {
	slog.Info("Start: Starting logging module")
	// TBD: separate config for  notifications vs requests logs?
	m.requestLogger, m.releaseFn = m.NewLogger(&m.Config)
	m.notificationLogger = m.requestLogger
	return nil
}

// Stop closes the logging destination.
func (m *LoggingService) Stop() {
	slog.Info("Stop: Stopping logging module")
	if m.releaseFn != nil {
		m.releaseFn()
		m.releaseFn = nil
	}
}

// NewLoggingService creates a new instance of the logging module.
//
// config is the default module configuration.
func NewLoggingService(config config.LoggingConfig) *LoggingService {

	m := &LoggingService{}
	m.Config = config
	m.SetModuleID(loggingapi.LoggingModuleType)

	var _ loggingapi.ILoggingService = m // interface check
	return m
}
