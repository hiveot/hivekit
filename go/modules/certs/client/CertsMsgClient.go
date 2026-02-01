package certsclient

import (
	"crypto/x509"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/certs"
	"github.com/hiveot/hivekit/go/modules/certs/certutils"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot"
)

// CertsMsgClient is a client for the Certificate service using RRN messages.
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
	req := msg.NewRequestMessage(
		wot.OpInvokeAction, certs.DefaultCertsThingID, certs.ActionGetCACert, nil, "")

	resp, err := cl.ForwardRequestWait(req)
	if err == nil {
		err = resp.Decode(&certPem)
	}
	if err == nil {
		cert, err = certutils.X509CertFromPEM(certPem)
	}
	return cert, err
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
	cl := &CertsMsgClient{
		certServiceID: thingID,
	}
	cl.SetModuleID(thingID + "-client")
	if sink != nil {
		cl.SetRequestSink(sink.HandleRequest)
		sink.SetNotificationSink(cl.HandleNotification)
	}
	return cl
}
