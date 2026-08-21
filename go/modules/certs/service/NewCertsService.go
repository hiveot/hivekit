package certs_service

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/certs"
	"github.com/hiveot/hivekit/go/modules/certs/internal"
)

// Create a new instance of the default certs service module.
// This uses the self-signed cert service implementation
func NewCertsService(config *certs.CertsConfig) certs.ICertsService {
	svc := internal.NewCertsServiceImpl(config)
	return svc
}

// Create a new instance of the certs server module using the factory environment
// This module is reachable as the DefaultCertsServiceID ThingID
// Configuration is optional and defaults to the self-signed cert provider.
//
// Configuration:
//
//	CertsDir is the storage directory to read or create keys and certificates.
func NewCertsServiceFactory(f api.IModuleFactory, md *api.ModuleDefinition) (
	svc api.IHiveModule, err error) {

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
