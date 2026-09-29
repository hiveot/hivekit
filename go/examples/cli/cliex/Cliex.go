package cliex

import (
	"crypto/x509"
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells/consumer"
	"github.com/hiveot/hivekit/go/cells/directory"
	"github.com/hiveot/hivekit/go/cells/transport/discovery"
)

type CliexConfig struct {
	// Do not start with discovery
	NoDisco bool
	// Subscribe to events or property
	Subscribe bool
	// Show more detailed output
	Verbose bool
}

// The CLI example consumer.
type Cliex struct {
	// this is a consumer for chaining cells and sending Thing operations.
	co *consumer.Consumer

	// The discovery client to use for discovering directories and devices
	discoClient discovery.IDiscoveryClient

	dirClient directory.IDirectoryClient

	// for contacting the directory using http
	caCert *x509.Certificate

	// app config
	config CliexConfig
}

// locate a TD through the directory or discovery.
// This searches in the following sequence:
//  1. checks if the TD is known to the directory client
//  2. search using the discovery client
//  3. if no remote directory is set then discover a directory,
//     and check the directory client again.
func (cliex *Cliex) FindTD(thingID string) (tdoc *td.TD) {
	// var err error
	var maxWaitTime = time.Second * 1
	// var tddURL string

	// 1. search using discovery
	tdoc = cliex.discoClient.DiscoverFirstTD(thingID, "", maxWaitTime)
	if tdoc != nil {
		cliex.dirClient.Cache().ImportTD(tdoc)
		return tdoc
	}

	// 2. Ask the directory client.
	tdoc, _ = cliex.dirClient.RetrieveThing(thingID)
	if tdoc == nil {
		slog.Warn("FindTD. No TD found", "thingID", thingID)
	}
	return tdoc
}

// Create a new instance of the CLI app
//
//	config is the CLI configuration
//	co is the consumer helper for publishing requests
//	discoClient is the discovery client
//	dirClient is the client for contacting a discovered directory
//	caCert is the application environment CA
func NewCliex(config CliexConfig,
	co *consumer.Consumer,
	discoClient discovery.IDiscoveryClient,
	dirClient directory.IDirectoryClient,
	caCert *x509.Certificate) *Cliex {

	m := &Cliex{
		co:          co,
		caCert:      caCert,
		config:      config,
		discoClient: discoClient,
		dirClient:   dirClient,
	}
	return m
}
