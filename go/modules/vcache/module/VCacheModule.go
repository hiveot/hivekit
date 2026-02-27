package module

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/logging/config"
	"github.com/hiveot/hivekit/go/modules/vcache"
	"github.com/hiveot/hivekit/go/msg"
)

// VCacheModule is the value-cache module implementation
// this implements the IVCache and IHiveModule interface
type VCacheModule struct {
	modules.HiveModuleBase
}

// log notifications upstream and logs them if they pass the filter
func (m *VCacheModule) HandleNotification(notif *msg.NotificationMessage) {
	m.ForwardNotification(notif)
}

// HandleRequest forwards requests downstream and logs them if they pass the filter
func (m *VCacheModule) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	return m.ForwardRequest(req, replyTo)
}

// Start opens the logging destination.
func (m *VCacheModule) Start(configYaml string) (err error) {
	return nil
}

// Stop closes the logging destination.
func (m *VCacheModule) Stop() {
}

// Create a new instance of the value cache module.
func NewVCacheModule(config config.LoggingConfig) *VCacheModule {

	m := &VCacheModule{}

	var _ vcache.IVCacheModule = m // interface check
	return m
}
