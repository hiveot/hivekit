package certs_client

import (
	"crypto"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/wot"
	"github.com/hiveot/hivekit/go/cells"
	"github.com/hiveot/hivekit/go/cells/certs"
	"github.com/hiveot/hivekit/go/utils"
)

// CertsClient is a client for the Certificate service using RRN messages.
type CertsClient struct {
	cells.HiveCellBase // clients can be used as cells

	// CertsMsgClient is the RRN client for the directory service.

	// Certificate service ThingID to connect to.
	certServiceID string
}

// Request the creation of a client certificate
func (cl *CertsClient) CreateClientCert(
	clientID string, ou string, validity time.Duration,
	pubKey crypto.PublicKey) (*x509.Certificate, error) {

	pubKeyPEM := utils.PublicKeyToPem(pubKey)
	if pubKeyPEM == "" || clientID == "" || ou == "" {
		return nil, fmt.Errorf("CreateClientCert: Invalid arg")
	}
	days := int(validity / (time.Hour * 24))
	if days <= 1 {
		days = 30
	}
	input := certs.CreateClientCertInput{
		ClientID:  clientID,
		OU:        ou,
		Days:      days,
		PubKeyPEM: pubKeyPEM,
	}
	clientCertPEM := ""
	err := cl.Rpc(wot.OpInvokeAction, cl.certServiceID, certs.CreateClientCertAction, input, &clientCertPEM)
	if err != nil {
		return nil, err
	}
	cert, err := utils.X509CertFromPEM(clientCertPEM)
	return cert, err
}

// GetCACert returns the CA x509 certificate.
func (cl *CertsClient) GetCACert() (caCert *x509.Certificate, err error) {
	var certPem string

	err = cl.Rpc(wot.OpInvokeAction, cl.certServiceID, certs.GetCACertAction, nil, &certPem)
	if err == nil {
		caCert, err = utils.X509CertFromPEM(certPem)
	}
	return caCert, err
}

// Verify if the given client certificate is still valid
func (cl *CertsClient) VerifyClientCert(clientID string, clientCert *x509.Certificate) error {

	certPem := utils.X509CertToPEM(clientCert)
	err := cl.Rpc(wot.OpInvokeAction, cl.certServiceID, certs.VerifyClientCertAction, &certPem, nil)
	return err
}

// NewCertsClient creates a ready-to-use CertsClient instance.
// Use the sink to attach a transport client
//
//	thingID is the certificate service instance thingID.
//	sink is the optional cell that passes request to the transport client.
func NewCertsClient(thingID string, sink api.IHiveCell) *CertsClient {
	cl := &CertsClient{
		certServiceID: thingID,
	}
	if sink != nil {
		cl.SetRequestSink(sink)
		sink.SetNotificationSink(cl)
	}

	// not all service methods are available through this client
	// var _ certs.ICertsService = cl // API check

	return cl
}
