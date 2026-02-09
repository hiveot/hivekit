package module

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/logging"
	"github.com/hiveot/hivekit/go/msg"
)

// LoggingModule is a module for writing request, response and notification messages to a log sink.

// The module is configured using yaml.
type LoggingModule struct {
	modules.HiveModuleBase

	// Root for storing log files.
	storageRoot string
}

// log notifications and forward upstream
func (m *LoggingModule) HandleNotification(notif *msg.NotificationMessage) {
	// todo: filters and multiple destinations
	// m.LogNotification(req)
	m.ForwardNotification(notif)
}

// log requests and forward downstream
func (m *LoggingModule) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	// todo: filters and multiple destinations
	// m.LogRequest(req)
	return m.ForwardRequest(req, replyTo)
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
func (m *LoggingModule) Start(_ string) (err error) {
	return nil
}

// Stop closes the logging destination.
func (m *LoggingModule) Stop() {
}

// Create a new instance of the logging module.
//
// The storageRoot is the root directory for storing log files.
// It can be used to create a file-based log sink, or it can be ignored if the logging
// module uses a different log sink (e.g. console, remote server).
func NewLoggingModule(storageRoot string) *LoggingModule {

	m := &LoggingModule{
		storageRoot: storageRoot,
	}
	m.SetModuleID(logging.LoggingModuleID)

	var _ logging.ILoggingModule = m // interface check
	return m
}
