package msg

import (
	"github.com/hiveot/hivekit/go/utils"
	"github.com/teris-io/shortid"
)

// ForwardRequestWait is a helper function to forward a request to the given handler
// and waits for a response.
//
// This assigns a request correlationID if none is set.
//
// If the response contains an error, it is return as the error.
func ForwardRequestWait(req *RequestMessage, reqHandler RequestHandler) (resp *ResponseMessage, err error) {
	if req.CorrelationID == "" {
		req.CorrelationID = shortid.MustGenerate()
	}
	ar := utils.NewAsyncReceiver[*ResponseMessage]()

	err = reqHandler(req, func(r *ResponseMessage) error {
		ar.SetResponse(r)
		return nil
	})
	if err != nil {
		return nil, err
	}
	resp, err = ar.WaitForResponse(0)
	if err == nil {
		err = resp.AsError()
	}
	return resp, err
}
