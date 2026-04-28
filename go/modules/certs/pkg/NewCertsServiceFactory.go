package certspkg

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/certs/internal/service"
	factory "github.com/hiveot/hivekit/go/modules/factory"
)

// Create a new instance of the certs server module using the factory environment
// This module is reachable as the DefaultCertsServiceID ThingID
// certsDir is the storage directory to read or create keys and certificates.
func NewCertsServiceFactory(f factory.IModuleFactory) modules.IHiveModule {
	envDir := f.GetEnvironment()
	certsDir := envDir.CertsDir
	m := service.NewCertsService(certsDir)
	return m
}
