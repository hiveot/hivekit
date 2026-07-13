package cliex

import (
	"crypto/x509"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/consumer"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
)

type CliexConfig struct {
	// Do not start with discovery
	NoDisco bool
	// Subscribe to events or property
	Subscribe bool
	// Show more detailed output
	Verbose bool
}

// The CLI example has a module wrapper so it can be used as part of the module chain
type Cliex struct {
	// this is a consumer for chaining modules. Don't use it directly.
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
	config CliexConfig
}

// Create a new instance of the CLI app
func NewCliex(config CliexConfig,
	discoClient discovery.IDiscoveryClient,
	dirClient directory.IDirectoryClient,
	caCert *x509.Certificate) *Cliex {

	m := &Cliex{
		Consumer:    consumer.NewConsumer(nil, nil),
		caCert:      caCert,
		config:      config,
		discoClient: discoClient,
		dirClient:   dirClient,
	}
	m.co = m.Consumer
	return m
}

// Factory function for the cli app
func NewCliexFactory(f api.IModuleFactory, modDef *api.ModuleDefinition) (api.IHiveModule, error) {

	config, ok := modDef.Config.(CliexConfig)
	discoClient := api.GetFactoryModule[discovery.IDiscoveryClient](f, discovery.DiscoveryClientModuleType)
	dirClient := api.GetFactoryModule[directory.IDirectoryClient](f, directory.DirectoryClientModuleType)
	_ = ok

	m := NewCliex(config, discoClient, dirClient, f.GetEnvironment().CaCert)
	return m, nil
}
