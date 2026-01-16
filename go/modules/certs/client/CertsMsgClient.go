package certsclient

import (
	"crypto/x509"
	"fmt"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/certs"
)

// CertsMsgClient is a client for the Certificate service using RRN messages.
// This implements the ICertsService interface.
type CertsMsgClient struct {
	// CertsMsgClient is the RRN client for the directory service.

	// Certificate service ThingID to connect to.
	directoryID string
	// sink that forwards notifications submitted by this module
	sink modules.IHiveModule
}

// GetCACert returns the x509 CA certificate.
func (cl *CertsMsgClient) GetCACert() (*x509.Certificate, error) {
	return nil, fmt.Errorf("not yet implemented")
}

// NewCertsMsgClient creates a new CertsMsgClient instance.
// Use the sink to attach a transport module
//
//	thingID is the unique ID of the certificate service instance
//	sink is the handler that forwards requests to the module. Typically a messaging client.
func NewCertsMsgClient(thingID string, sink modules.IHiveModule) *CertsMsgClient {
	if thingID == "" {
		thingID = certs.DefaultCertsThingID
	}
	client := &CertsMsgClient{
		directoryID: thingID,
		sink:        sink,
	}
	return client
}
