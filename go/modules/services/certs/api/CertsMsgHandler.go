package api

import (
	"fmt"

	"github.com/hiveot/hivekit/go/modules/messaging"
	"github.com/hiveot/hivekit/go/modules/services/certs"
	"github.com/hiveot/hivekit/go/wot"
)

// Embed the directory TM
//
// //go:embed certs-tm.json
//var CertsTMJson []byte

// CertsMsgHandler maps SME messages to the native directory interface
type CertsMsgHandler struct {
	// the certificate manager instance ThingID that must match the requests
	thingID string
	service certs.ICertsService
}

// HandleRequest for properties or actions
// If the request is not recognized nil is returned.
// If the request is missing the sender, an error is returned
func (handler *CertsMsgHandler) HandleRequest(req *messaging.RequestMessage) (resp *messaging.ResponseMessage) {
	if req.ThingID != handler.thingID {
		return nil
	} else if req.SenderID == "" {
		err := fmt.Errorf("missing senderID in request")
		return req.CreateErrorResponse(err)
	}
	if req.Operation == wot.OpInvokeAction {
		// certificate specific operations
		switch req.Name {
		// case ActionGetCA:
		}
	}
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
