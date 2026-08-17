package internal

import (
	"crypto/x509"
	"fmt"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules/certs"
	"github.com/hiveot/hivekit/go/utils"
)

// Invoke the GetCACert method
func (svc *CertsServiceImpl) handleGetCACert(req *msg.RequestMessage) (resp *msg.ResponseMessage, err error) {
	// no args
	caCert := svc.GetCACert()
	if caCert == nil {
		return nil, fmt.Errorf("No CA cert")
	}
	// convert cert to PEM
	caPem := utils.X509CertToPEM(caCert)
	resp = req.CreateResponse(caPem, err)
	return resp, nil
}

// Decode the Get Server cert method
func (svc *CertsServiceImpl) handleGetServerCert(req *msg.RequestMessage) (resp *msg.ResponseMessage, err error) {
	// no args
	var serverName string
	req.DecodeInput(&serverName)
	certChain, err := svc.GetServerCert(serverName)
	if err != nil {
		return nil, err
	}
	// convert cert to PEM
	certPem := utils.X509ChainToPEM(certChain)
	resp = req.CreateResponse(certPem, err)
	return resp, nil
}

func (svc *CertsServiceImpl) handleVerifyClientCert(req *msg.RequestMessage) (resp *msg.ResponseMessage, err error) {
	// pem args
	var clientCert *x509.Certificate
	var clientCertPem string

	err = req.DecodeInput(&clientCertPem)
	if err == nil {
		clientCert, err = utils.X509CertFromPEM(clientCertPem)
	}
	if err == nil {
		err = svc.VerifyClientCert(req.SenderID, clientCert)
	}
	resp = req.CreateResponse(nil, err)
	return resp, nil
}

// HandleRequest for properties or actions
// If the request is not recognized nil is returned.
// If the request is missing the sender, an error is returned
func (svc *CertsServiceImpl) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	if req.ThingID != svc.GetThingID() {
		return svc.ForwardRequest(req, replyTo)
	}

	var resp *msg.ResponseMessage
	if req.SenderID == "" {
		// todo: is this really needed?
		err = fmt.Errorf("missing senderID in request")
	} else if req.Operation == td.OpInvokeAction {
		// certificate specific operations
		switch req.Name {
		case certs.GetCACertAction:
			resp, err = svc.handleGetCACert(req)
		case certs.GetServerCertAction:
			resp, err = svc.handleGetServerCert(req)
		case certs.VerifyClientCertAction:
			resp, err = svc.handleVerifyClientCert(req)
		default:
			err = fmt.Errorf("Unknown request name '%s' for thingID '%s'", req.Name, req.ThingID)
		}
	} else {
		err = fmt.Errorf("Unsupported operation '%s' for thingID '%s'", req.Operation, req.ThingID)
	}
	if resp != nil {
		err = replyTo(resp)
	}
	return err
}
