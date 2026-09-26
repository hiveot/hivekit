package internal

import (
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells"
	"github.com/hiveot/hivekit/go/cells/rcrouter"
	"github.com/teris-io/shortid"
)

// Implementation of the rc-router service.
//
// This routes requests to reverse-connected devices. Intended for use in gateways.
type RCRouterServiceImpl struct {
	*cells.HiveCellBase

	// handler that provides a TD for the given thingID.
	// Required for determining the connection clientID that serves a Thing.
	getTD func(thingID string) *td.TD

	// handler to get available transport servers for forwarding to RC clients.
	getSrv func() []api.ITransportServer
}

// Return the reverse-client connection to a device, if it exists.
// This returns nil if the clientID does not have an existing connection.
func (svc *RCRouterServiceImpl) GetRCConnection(clientID string) (c api.IConnection) {
	if svc.getSrv == nil {
		return nil
	}
	serverList := svc.getSrv()
	for _, tp := range serverList {
		c := tp.GetConnectionByClientID(clientID)
		if c != nil {
			return c
		}
	}
	return nil
}

// HandleRequest handles requests or routes the request to its destination
func (svc *RCRouterServiceImpl) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {

	if req.ThingID != svc.GetThingID() {
		return svc.RouteRequest(req, replyTo)
	}
	// This service doesn't define any requests.
	// TODO: maybe keep track of some counters?
	err := fmt.Errorf("RCRouterServiceImpl.HandleRequest: Unhandled request: thingID='%s', op='%s', name='%s", req.ThingID, req.Operation, req.Name)
	return err
}

// Route the request to remote devices.
// If no connection can be established then pass the request to the sink.
//
// Lookup the TD of the ThingID and determine its destination:
func (svc *RCRouterServiceImpl) RouteRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	tdoc := svc.getTD(req.ThingID)
	//  look for a reverse connection using the TD senderID.
	//
	// TODO: if an admin uploads a bunch of TDs without forms then requests for those
	// TDs will go to the admin user. This is obviously not intended.
	rcClientID := tdoc.GetSenderID()
	if rcClientID != "" {
		c := svc.GetRCConnection(rcClientID)
		if c == nil {
			err = fmt.Errorf("RouteRequest: device '%s' isnt connected", rcClientID)
		} else {
			err = c.SendRequest(req, replyTo)
		}
		return err
	}

	// unable to route request. Forward it to the next cell.
	err = svc.ForwardRequest(req, replyTo)

	return err
}

// NewRCRouterServiceImpl creates a new router service
//
// getSrv provides servers to route the request to.
//
//	getTD  handler to lookup a TD for a thingID from a directory. Required.
//	getSrv handler returning a list of transport servers that can contain RC devices.
func NewRCRouterServiceImpl(
	getTD func(thingID string) *td.TD,
	getSrv func() []api.ITransportServer,
) (*RCRouterServiceImpl, error) {

	if getTD == nil || getSrv == nil {
		return nil, fmt.Errorf("NewRCRouterServiceImpl: nil argument")
	}

	slog.Info("NewRCRouterServiceImpl: Starting RC-Router service")

	thingID := rcrouter.RCRouterCellType + "-" + shortid.MustGenerate()
	svc := &RCRouterServiceImpl{
		HiveCellBase: cells.NewHiveCellBase(thingID),
		getTD:        getTD,
		getSrv:       getSrv,
	}

	var _ rcrouter.IRCRouterService = svc // interface check

	return svc, nil
}
