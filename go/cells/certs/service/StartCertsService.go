package certs_service

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/certs"
	"github.com/hiveot/hivekit/go/cells/certs/internal"
)

// Start a new instance of the default certs service.
// This uses the self-signed cert service implementation
func StartCertsService(config *certs.CertsConfig) (certs.ICertsService, error) {
	svc, err := internal.StartCertsServiceImpl(config)
	return svc, err
}

// Create a new instance of the certs service using the cell factory environment.
//
// Configuration is optional and defaults to the self-signed cert provider.
func StartCertsServiceFactory(
	f api.ICellFactory, md *api.CellDefinition) (svc api.IHiveCell, err error) {

	envDir := f.GetEnvironment()

	config, ok := md.Config.(*certs.CertsConfig)
	if !ok {
		config, _ = md.Config.(*certs.CertsConfig)
		if config == nil {
			config = &certs.CertsConfig{
				// Provider: certs.SelfSignedProvider,
				CertsDir: envDir.CertsDir,
			}
		}
	}
	svc, err = StartCertsService(config)
	// if config.Provider == certs.LetsEncryptProvider {
	// 	m = NewLetsEncryptCertService(config)
	// } else {
	// 	m = NewSelfSignedCertService(config)
	// }
	return svc, err
}
