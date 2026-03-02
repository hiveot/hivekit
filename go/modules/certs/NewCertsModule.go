package certs

import (
	certsapi "github.com/hiveot/hivekit/go/modules/certs/api"
	certsserver "github.com/hiveot/hivekit/go/modules/certs/internal/server"
)

// Create a new instance of the certs server module
// This module is reachable as the DefaultCertsServiceID ThingID
// certsDir is the storage directory to read or create keys and certificates.
func NewCertsServer(certsDir string) certsapi.ICertsServer {
	m := certsserver.NewCertsServer(certsDir)
	return m
}
