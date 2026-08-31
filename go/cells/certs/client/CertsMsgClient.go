package certsclient

import (
	"crypto/x509"

	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells"
	"github.com/hiveot/hivekit/go/cells/certs"
	certsapi "github.com/hiveot/hivekit/go/cells/certs"
	"github.com/hiveot/hivekit/go/utils"
)

// CertsClient is a client for the Certificate service using RRN messages.
type CertsClient struct {
	cells.HiveCellBase // clients can be used as cells

	// CertsMsgClient is the RRN client for the directory service.

	// Certificate service ThingID to connect to.
	certServiceID string
}

// GetCACert returns the CA x509 certificate.
func (cl *CertsClient) GetCACert() (caCert *x509.Certificate, err error) {
	var certPem string

	err = cl.Rpc(td.OpInvokeAction, cl.certServiceID, certs.GetCACertAction, nil, &certPem)
	if err == nil {
		caCert, err = utils.X509CertFromPEM(certPem)
	}
	return caCert, err
}

// Verify if the given client certificate is still valid
func (cl *CertsClient) VerifyClientCert(clientID string, clientCert *x509.Certificate) error {

	certPem := utils.X509CertToPEM(clientCert)
	err := cl.Rpc(td.OpInvokeAction, cl.certServiceID, certs.VerifyClientCertAction, &certPem, nil)
	return err
}

// StartCertsClient creates a new CertsMsgClient instance.
// Use the sink to attach a transport client
//
//	certServiceID is the certificate service instance thingID, "" to select default.
func StartCertsClient(svcThingID string) *CertsClient {
	if svcThingID == "" {
		svcThingID = certsapi.DefaultCertsServiceThingID
	}
	cl := &CertsClient{
		certServiceID: svcThingID,
	}
	// not all service methods are available through this client
	// var _ certs.ICertsService = cl // API check

	return cl
}
