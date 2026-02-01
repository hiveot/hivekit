package server

import (
	_ "embed"
	"fmt"

	"github.com/hiveot/hivekit/go/modules/certs"
	"github.com/hiveot/hivekit/go/modules/certs/certutils"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot"
)

// Embed the certs TM
//
//go:embed "certs-tm.json"
var CertsTMJson []byte

// CertsMsgHandler maps RRN messages to the native service interface
type CertsMsgHandler struct {
	// the certificate manager instance ThingID that must match the requests
	thingID string
	service certs.ICertsService
}

// HandleRequest for properties or actions
// If the request is not recognized nil is returned.
// If the request is missing the sender, an error is returned
func (handler *CertsMsgHandler) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	var resp *msg.ResponseMessage
	if req.ThingID != handler.thingID {
		return nil
	} else if req.SenderID == "" {
		// todo: is this really needed?
		err = fmt.Errorf("missing senderID in request")
	} else if req.Operation == wot.OpInvokeAction {
		// certificate specific operations
		switch req.Name {
		case certs.ActionGetCACert:
			resp, err = handler.GetCaCert(req)
		case certs.ActionGetServerCert:
			resp, err = handler.GetDefaultServerCert(req)
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
func (handler *CertsMsgHandler) GetCaCert(req *msg.RequestMessage) (resp *msg.ResponseMessage, err error) {
	// no args
	cert, err := handler.service.GetCACert()
	if err != nil {
		return nil, err
	}
	// convert cert to PEM
	caPEM := certutils.X509CertToPEM(cert)
	resp, _ = req.CreateActionResponse("", msg.StatusCompleted, caPEM, err)
	return resp, nil
}

// Decode the Get Server cert method
func (handler *CertsMsgHandler) GetDefaultServerCert(req *msg.RequestMessage) (resp *msg.ResponseMessage, err error) {
	// no args
	cert, err := handler.service.GetDefaultServerCert()
	if err != nil {
		return nil, err
	}
	// convert cert to PEM
	certPEM := certutils.X509CertToPEM(cert)
	resp, _ = req.CreateActionResponse("", msg.StatusCompleted, certPEM, err)
	return resp, nil
}

// Create a new directory message handler. On start this creates the server and store.
// bucketStore is the store to use for this module chain.
func NewCertsMsgHandler(thingID string, service certs.ICertsService) *CertsMsgHandler {

	handler := &CertsMsgHandler{
		thingID: thingID,
		service: service,
	}
	return handler
}
