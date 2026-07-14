package cliex

import (
	"crypto/x509"
	"time"

	"github.com/hiveot/hivekit/go/api/td"
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

// The CLI example consumer module.
type Cliex struct {
	// this is a consumer for chaining modules and sending Thing operations.
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

// locate a TD through the directory or discovery
// This takes the following steps:
// 1. checks if the TD is known to the directory client
// 2. check if the Thing is found by Thing discovery
// 3. locate a directory service
// 4. ask the directory service
func (cliex *Cliex) FindTD(thingID string) (tdoc *td.TD) {
	var maxWaitTime = time.Second * 1
	var tdd *td.TD

	// 1. ask the directory client
	tdoc, _ = cliex.dirClient.RetrieveThing(thingID)
	if tdoc != nil {
		return tdoc
	}
	// 2. attempt thing discovery
	cliex.discoClient.DiscoverThingTDs("", maxWaitTime, func(discoTD *td.TD) bool {
		if discoTD.ID == thingID {
			tdoc = discoTD
			return true
		}
		// while we're looking, capture a directory for the next step
		if discoTD.IsDirectory() {
			tdd = discoTD
		}
		return false
	})
	if tdoc != nil {
		return tdoc
	}

	// 3. Locate a directory
	if tdd == nil {
		return nil
	}
	// 4. Read this directory
	// this will pass a request down the recipe chain to the router
	cliex.dirClient.SetTDD(tdd)
	tdoc, _ = cliex.dirClient.RetrieveThing(thingID)
	return tdoc
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

// // Factory function for the cli app
// func NewCliexFactory(f api.IModuleFactory, modDef *api.ModuleDefinition) (api.IHiveModule, error) {

// 	config, ok := modDef.Config.(CliexConfig)
// 	discoClient := api.GetFactoryModule[discovery.IDiscoveryClient](f, discovery.DiscoveryClientModuleType)
// 	dirClient := api.GetFactoryModule[directory.IDirectoryClient](f, directory.DirectoryClientModuleType)
// 	_ = ok

// 	m := NewCliex(config, discoClient, dirClient, f.GetEnvironment().CaCert)
// 	return m, nil
// }
