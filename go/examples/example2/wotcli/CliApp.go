package wotcli

import (
	"crypto/x509"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/consumer"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
)

type CliAppConfig struct {
	// Do not start with discovery
	NoDisco bool
	// Subscribe to events or property
	Subscribe bool
	// Show more detailed output
	Verbose bool
}

// The CLI App has a module wrapper so it can be used as part of the module chain
type CliApp struct {
	// this is a consumer. Don't use it directly.
	*consumer.Consumer

	// the consumer this app is linked to
	// right now the choice is to make the app itself the consumer.
	// This allows the option to change that if needed.
	co *consumer.Consumer

	// The discovery client to use for discovering directories and devices
	discoClient discovery.IDiscoveryClient

	dirClient directory.IDirectoryClient

	// for contacting the directory using http
	caCert *x509.Certificate

	// app config
	config CliAppConfig
}

// Start the CLI application
// This expects the next module in the chain to be the consumer
func (app *CliApp) Start() error {
	app.co = app.Consumer
	return nil
}

// Create a new instance of the CLI app
func NewCliApp(config CliAppConfig, discoClient discovery.IDiscoveryClient, caCert *x509.Certificate) *CliApp {
	m := &CliApp{
		Consumer: consumer.NewConsumer(nil, nil),
		caCert:   caCert,
		config:   config,
	}
	return m
}

// Factory function for the cli app
func NewCliAppFactory(f api.IModuleFactory,
	modDef *api.ModuleDefinition) (api.IHiveModule, error) {

	config, ok := modDef.Config.(CliAppConfig)
	_ = ok
	discoMod := f.GetModule(discovery.DiscoveryClientModuleType)
	discoClient, ok := discoMod.(discovery.IDiscoveryClient)

	m := NewCliApp(config, discoClient, f.GetEnvironment().CaCert)
	return m, nil
}
