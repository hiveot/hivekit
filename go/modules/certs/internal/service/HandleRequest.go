package service

import (
	_ "embed"
	"fmt"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules/certs"
	"github.com/hiveot/hivekit/go/modules/certs/certutils"
)

// Embed the certs TM
//
//go:embed "certs-tm.json"
var CertsTMJson []byte

// HandleRequest for properties or actions
// If the request is not recognized nil is returned.
// If the request is missing the sender, an error is returned
func (svc *CertsService) HandleRequest(
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
		case certs.ActionGetCACert:
			resp, err = svc._handleGetCACert(req)
		case certs.ActionGetServerCert:
			resp, err = svc._handleGetServerCert(req)
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

// Invoke the GetCACert method
func (svc *CertsService) _handleGetCACert(req *msg.RequestMessage) (resp *msg.ResponseMessage, err error) {
	// no args
	cert, err := svc.GetCACert()
	if err != nil {
		return nil, err
	}
	// convert cert to PEM
	caPEM := certutils.X509CertToPEM(cert)
	resp = req.CreateResponse(caPEM, err)
	return resp, nil
}

// Decode the Get Server cert method
func (svc *CertsService) _handleGetServerCert(req *msg.RequestMessage) (resp *msg.ResponseMessage, err error) {
	// no args
	cert, err := svc.GetDefaultServerCert()
	if err != nil {
		return nil, err
	}
	// convert cert to PEM
	certPEM := certutils.X509CertToPEM(cert)
	resp = req.CreateResponse(certPEM, err)
	return resp, nil
}
