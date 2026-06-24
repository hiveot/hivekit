package certspkg

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/certs"
	"github.com/hiveot/hivekit/go/modules/certs/internal/service"
	factory "github.com/hiveot/hivekit/go/modules/factory"
)

// Create a new instance of the certs server module
// This module is reachable as the DefaultCertsServiceID ThingID
// certsDir is the storage directory to read or create keys and certificates.
func NewCertsService(certsDir string) certs.ICertsService {
	m := service.NewCertsService(certsDir)
	return m
}

// Create a new instance of the certs server module using the factory environment
// This module is reachable as the DefaultCertsServiceID ThingID
// certsDir is the storage directory to read or create keys and certificates.
func NewCertsServiceFactory(f factory.IModuleFactory, md *factory.ModuleDefinition) (modules.IHiveModule, error) {
	envDir := f.GetEnvironment()
	certsDir := envDir.CertsDir
	m := service.NewCertsService(certsDir)
	return m, nil
}
