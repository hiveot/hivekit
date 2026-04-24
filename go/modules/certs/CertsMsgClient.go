package certs

import (
	"crypto/x509"

	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	certsapi "github.com/hiveot/hivekit/go/modules/certs/api"
	"github.com/hiveot/hivekit/go/modules/certs/certutils"
)

// CertsMsgClient is a client for the Certificate module using RRN messages.
// This implements the ICertsService interface.
type CertsMsgClient struct {
	modules.HiveModuleBase // clients can be used as modules

	// CertsMsgClient is the RRN client for the directory service.

	// Certificate service ThingID to connect to.
	certServiceID string
}

// GetCACert returns the x509 CA certificate.
func (cl *CertsMsgClient) GetCACert() (cert *x509.Certificate, err error) {
	var certPem string

	err = cl.Rpc("", td.OpInvokeAction, cl.certServiceID, certsapi.ActionGetCACert, nil, &certPem)
	if err == nil {
		cert, err = certutils.X509CertFromPEM(certPem)
	}
	return cert, err
}

// NewCertsMsgClient creates a new CertsMsgClient instance.
// Use the sink to attach a transport client
//
//	thingID is the unique ID of the certificate service instance
//	sink is the handler that passes requests to the service and receives notifications.
func NewCertsMsgClient(thingID string, sink modules.IHiveModule) *CertsMsgClient {
	if thingID == "" {
		thingID = certsapi.DefaultCertsServiceThingID
	}
	cl := &CertsMsgClient{
		certServiceID: thingID,
	}
	if sink != nil {
		cl.SetRequestSink(sink.HandleRequest)
		sink.SetNotificationSink(cl.HandleNotification)
	}
	// not all service methods are available through this client
	// var _ certs.ICertsService = cl // API check

	return cl
}
