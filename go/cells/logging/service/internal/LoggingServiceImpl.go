package internal

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/cells"
	"github.com/hiveot/hivekit/go/cells/logging"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/lmittmann/tint"
)

// LoggingServiceImpl is a service for writing request, response and notification messages to a log output.
// The service is configured using yaml.
type LoggingServiceImpl struct {
	*cells.HiveCellBase

	// configuration. Allow manual configuration
	Config logging.LoggingConfig

	// log destination for notifications
	notificationLogger *slog.Logger
	// log destination for notifications
	requestLogger *slog.Logger

	// handler to release log resources
	releaseFn func()
}

// log notifications upstream and logs them if they pass the filter
func (svc *LoggingServiceImpl) HandleNotification(notif *msg.NotificationMessage) {
	go func() {
		if svc.Config.NotificationFilter.AcceptNotification(notif) {
			svc.LogNotification(notif)
		}
	}()
	svc.ForwardNotification(notif)
}

// HandleRequest forwards requests downstream and logs them if they pass the filter
func (svc *LoggingServiceImpl) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	go func() {
		if svc.Config.NotificationFilter.AcceptRequest(req) {
			svc.LogRequest(req)
		}
	}()
	if req.ThingID != svc.GetID() && req.ThingID != logging.LoggingServiceCellType {
		return svc.ForwardRequest(req, replyTo)
	}
	slog.Info("HandleRequest", slog.String("name", req.Name), slog.String("sender", req.SenderID))
	return fmt.Errorf("HandleRequest: unsupported request '%s'", req.Name)
}

// write notifications to the logging backend
func (svc *LoggingServiceImpl) LogNotification(notif *msg.NotificationMessage) {
	value := utils.DecodeAsString(notif.Data, 32)
	svc.notificationLogger.Info("Notification",
		slog.String("correlationID", notif.CorrelationID),
		slog.String("type", string(notif.AffordanceType)),
		slog.String("thingID", notif.ThingID),
		slog.String("name", notif.Name),
		// slog.String("timestamp", notif.Timestamp),
		slog.String("sender", string(notif.SenderID)),
		slog.String("value", value),
	)

}

// write request to the logging backend
func (svc *LoggingServiceImpl) LogRequest(req *msg.RequestMessage) {
	value := utils.DecodeAsString(req.Input, 32)
	svc.requestLogger.Info("Request",
		slog.String("correlationID", req.CorrelationID),
		slog.String("op", string(req.Operation)),
		slog.String("thingID", req.ThingID),
		slog.String("name", req.Name),
		// slog.String("timestamp", req.Timestamp),
		slog.String("sender", string(req.SenderID)),
		slog.String("value", value),
	)

}

// NewLogger returns a new instance of a logger using the given backend along with
// a function to release resources.
func (svc *LoggingServiceImpl) NewLogger(cfg *logging.LoggingConfig) (
	logger *slog.Logger, releaseFn func()) {

	var logFile *os.File
	var logWriter io.Writer
	var err error

	if cfg.Backend == logging.LoggingBackendFile {
		// ensure the directory exists
		logDir := filepath.Dir(cfg.LogDestination)
		_ = os.MkdirAll(logDir, 0750)

		slog.Warn("Logging to logfile:" + cfg.LogDestination)
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
		opts := &tint.Options{
			AddSource:  false, // no need for notifications and requests
			TimeFormat: "Jan _2 15:04:05.0000",
		}
		handler := tint.NewTextHandler(logWriter, opts)
		logger = slog.New(handler)
	}
	logger.Info("Logging service started")
	return logger, releaseFn
}

// SetSource is a convenience function to set the source of requests and destination of notifications
func (svc *LoggingServiceImpl) SetSource(source api.IHiveCell) {
	source.SetRequestSink(svc)
	svc.SetNotificationSink(source)
}

// SetSink is a convenience function to set the downstream cell of requests and source of notifications
func (svc *LoggingServiceImpl) SetSink(sink api.IHiveCell) {
	svc.SetRequestSink(sink)
	sink.SetNotificationSink(svc)
}

// Stop closes the logging destination.
func (svc *LoggingServiceImpl) Stop() {
	slog.Info("Stop: Stopping logging service")
	if svc.releaseFn != nil {
		svc.releaseFn()
		svc.releaseFn = nil
	}
}

// NewLoggingServiceImpl creates a ready-to-use instance of the logging service.
//
// config is the default service configuration.
func NewLoggingServiceImpl(
	config logging.LoggingConfig) (*LoggingServiceImpl, error) {

	slog.Info("StartLoggingServiceImpl: Starting logging service")
	thingID := config.ThingID
	svc := &LoggingServiceImpl{
		HiveCellBase: cells.NewHiveCellBase(thingID),
		Config:       config,
	}
	// TBD: separate config for  notifications vs requests logs?
	svc.requestLogger, svc.releaseFn = svc.NewLogger(&svc.Config)
	svc.notificationLogger = svc.requestLogger

	return svc, nil
}
