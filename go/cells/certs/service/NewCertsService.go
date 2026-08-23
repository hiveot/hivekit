package certs_service

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/certs"
	"github.com/hiveot/hivekit/go/cells/certs/internal"
)

// Create a new instance of the default certs service.
// This uses the self-signed cert service implementation
func NewCertsService(config *certs.CertsConfig) certs.ICertsService {
	svc := internal.NewCertsServiceImpl(config)
	return svc
}

// Create a new instance of the certs service using the cell factory environment.
//
// Configuration is optional and defaults to the self-signed cert provider.
func NewCertsServiceFactory(f api.ICellFactory, md *api.CellDefinition) (
	svc api.IHiveCell, err error) {

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
	svc = NewCertsService(config)
	// if config.Provider == certs.LetsEncryptProvider {
	// 	m = NewLetsEncryptCertService(config)
	// } else {
	// 	m = NewSelfSignedCertService(config)
	// }
	return svc, nil
}
