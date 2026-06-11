package certspkg

import (
	"crypto/x509"

	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/certs"
	certsapi "github.com/hiveot/hivekit/go/modules/certs"
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

	err = cl.Rpc(td.OpInvokeAction, cl.certServiceID, certs.ActionGetCACert, nil, &certPem)
	if err == nil {
		cert, err = certutils.X509CertFromPEM(certPem)
	}
	return cert, err
}

// NewCertsMsgClient creates a new CertsMsgClient instance.
// Use the sink to attach a transport client
//
//	certServiceID is the certificate service instance thingID, "" to select default.
//	sink is the handler that forwards requests to the service and receives notifications. nil to ignore.
func NewCertsMsgClient(sink modules.IHiveModule, svcThingID string) *CertsMsgClient {
	if svcThingID == "" {
		svcThingID = certsapi.DefaultCertsServiceThingID
	}
	cl := &CertsMsgClient{
		certServiceID: svcThingID,
	}
	if sink != nil {
		cl.SetRequestSink(sink)
		sink.SetNotificationSink(cl)
	}
	// not all service methods are available through this client
	// var _ certs.ICertsService = cl // API check

	return cl
}
