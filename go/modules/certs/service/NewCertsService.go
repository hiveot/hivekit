package certs_service

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/certs"
	"github.com/hiveot/hivekit/go/modules/certs/internal/serviceimpl"
)

// Create a new instance of the certs server module
// This module is reachable as the DefaultCertsServiceID ThingID
// certsDir is the storage directory to read or create keys and certificates.
func NewCertsService(config certs.CertsConfig) certs.ICertsService {
	m := serviceimpl.NewCertsServiceImpl(config)
	return m
}

// Create a new instance of the certs server module using the factory environment
// This module is reachable as the DefaultCertsServiceID ThingID
// certsDir is the storage directory to read or create keys and certificates.
func NewCertsServiceFactory(f api.IModuleFactory, md *api.ModuleDefinition) (api.IHiveModule, error) {
	envDir := f.GetEnvironment()

	config, ok := md.Config.(*certs.CertsConfig)
	if !ok {
		config = &certs.CertsConfig{
			CertsDir: envDir.CertsDir,
		}
	}
	m := serviceimpl.NewCertsServiceImpl(*config)
	return m, nil
}
