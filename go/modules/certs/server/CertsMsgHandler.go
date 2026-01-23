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
func (handler *CertsMsgHandler) HandleRequest(req *msg.RequestMessage) (resp *msg.ResponseMessage) {
	if req.ThingID != handler.thingID {
		return nil
	} else if req.SenderID == "" {
		err := fmt.Errorf("missing senderID in request")
		return req.CreateErrorResponse(err)
	}
	if req.Operation == wot.OpInvokeAction {
		// certificate specific operations
		switch req.Name {
		case certs.ActionGetCACert:
			resp = handler.GetCaCert(req)
		}
	}
	return resp
}

// Invoke the GetCACert method
func (handler *CertsMsgHandler) GetCaCert(req *msg.RequestMessage) (resp *msg.ResponseMessage) {
	// no args
	cert, err := handler.service.GetCACert()
	// convert cert to PEM
	caPEM := certutils.X509CertToPEM(cert)
	req.CreateActionResponse("", msg.StatusCompleted, caPEM, err)
	return resp
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
