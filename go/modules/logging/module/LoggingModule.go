package module

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/logging"
	"github.com/hiveot/hivekit/go/modules/logging/config"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert/yaml"
)

// LoggingModule is a module for writing request, response and notification messages to a log output.

// The module is configured using yaml.
type LoggingModule struct {
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
func (m *LoggingModule) HandleNotification(notif *msg.NotificationMessage) {
	go func() {
		if m.Config.NotificationFilter.AcceptNotification(notif) {
			m.LogNotification(notif)
		}
	}()
	m.ForwardNotification(notif)
}

// HandleRequest forwards requests downstream and logs them if they pass the filter
func (m *LoggingModule) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	go func() {
		if m.Config.NotificationFilter.AcceptRequest(req) {
			m.LogRequest(req)
		}
	}()
	return m.ForwardRequest(req, replyTo)
}

// write notifications to the logging backend
func (m *LoggingModule) LogNotification(notif *msg.NotificationMessage) {
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
func (m *LoggingModule) LogRequest(req *msg.RequestMessage) {
	value := utils.DecodeAsString(req.Input, 32)
	m.requestLogger.Info("Request",
		slog.String("op", string(req.Operation)),
		slog.String("thingID", req.ThingID),
		slog.String("name", req.Name),
		slog.String("value", value),
		slog.String("created", req.Created),
		slog.String("sender", string(req.SenderID)),
	)

}

// NewLogger returns a new instance of a logger using the given backend along with
// a function to release resources.
func (m *LoggingModule) NewLogger(cfg *config.LoggingConfig) (
	logger *slog.Logger, releaseFn func()) {

	var logFile *os.File
	var logWriter io.Writer
	var err error

	if cfg.Backend == logging.LoggingBackendFile {
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
func (m *LoggingModule) SetSource(source modules.IHiveModule) {
	source.SetRequestSink(m.HandleRequest)
	m.SetNotificationSink(source.HandleNotification)
}

// SetSink is a convenience function to set the downstream module of requests and source of notifications
func (m *LoggingModule) SetSink(sink modules.IHiveModule) {
	m.SetRequestSink(sink.HandleRequest)
	sink.SetNotificationSink(m.HandleNotification)
}

// Start opens the logging destination.
func (m *LoggingModule) Start(configYaml string) (err error) {

	if configYaml != "" {
		err = yaml.Unmarshal([]byte(configYaml), &m.Config)
		if err != nil {
			slog.Error("Start: Failed to load logging module config", "error", err)
			return err
		}
	}

	m.SetModuleID(logging.DefaultLoggingModuleID)
	// TBD: separate config for  notifications vs requests logs?
	m.requestLogger, m.releaseFn = m.NewLogger(&m.Config)
	m.notificationLogger = m.requestLogger
	return nil
}

// Stop closes the logging destination.
func (m *LoggingModule) Stop() {
	if m.releaseFn != nil {
		m.releaseFn()
		m.releaseFn = nil
	}
}

// Create a new instance of the logging module.
//
// The storageRoot is the root directory for storing log files.
// It can be used to create a file-based log sink, or it can be ignored if the logging
// module uses a different log sink (e.g. console, remote server).
//
// config is the default module configuration.
func NewLoggingModule(config config.LoggingConfig) *LoggingModule {

	m := &LoggingModule{}
	m.Config = config

	var _ logging.ILoggingModule = m // interface check
	return m
}
