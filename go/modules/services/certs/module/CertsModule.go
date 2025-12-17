package module

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/services/certs"
	"github.com/hiveot/hivekit/go/modules/services/certs/service"
)

// CertsModule is a module for managing certificates.
// This implements IHiveModule interface.
//
// The module's HandleRequest and HandleNotification methods can be used to
// send messages to connected clients.
// Handlers of received messages can be added as sinks to this module.
// Since http is a connectionless protocol, this does not have the onConnect hook
// that other transports have.
type CertsModule struct {
	modules.HiveModuleBase
	service *service.CertsService

	// directory where certificates are stored
	certsDir string
}

// GetService returns the certificate service API
func (m *CertsModule) GetService() (service certs.ICertsService) {
	return m.service
}

// Start readies the certificate management module for use.
func (m *CertsModule) Start() (err error) {
	m.service = service.NewCertsService(m.certsDir)
	return err
}

// Stop any running actions
func (m *CertsModule) Stop() {
	// m.service.Stop()
}

// Create a new certificate management module
// certsDir is the storage directory to read or create keys and certificates.
func NewCertsModule(certsDir string) *CertsModule {
	m := &CertsModule{
		certsDir: certsDir,
	}
	var _ modules.IHiveModule = m // interface check
	return m
}
