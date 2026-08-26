package transport

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/cells"
	"github.com/teris-io/shortid"
)

// Base for client connections.
// This implements some of the boiler plate used to establish connections.
// Use of this is entirely optional. Implement your own if a different authentication
// scheme is needed.
type TransportClientBase struct {
	cells.HiveCellBase

	// the authentication scheme to use in Connect
	// See td.SecScheme... for possible values
	authScheme string

	// authentication bearer token or some other secret as determined by authScheme
	authSecret string

	// client certificate set by AuthenticateWithCert
	clientCert *tls.Certificate

	// connection ID set during connect
	cid string

	// clientID set by AuthenticateWith...
	clientID string

	// current connection status
	connectStatus api.ConnectionStatus
	// callback when connection changes
	connectHandler func(newStatus api.ConnectionStatus, c api.ITransportClient)

	// variables access
	mux sync.RWMutex

	// the request & response channel handler
	// all responses are passed here to support response callbacks
	// rnrChan *msg.RnRChan

	// Root CA's for client cert validation before use. nil to ignore validation.
	rootCAs *x509.CertPool

	// the parent transport client to pass to connection status callback
	transportClient api.ITransportClient
}

// GetAuthToken returns the client's authentication token and scheme
func (cl *TransportClientBase) GetAuthToken() (token string, scheme string) {
	cl.mux.RLock()
	defer cl.mux.RUnlock()
	return cl.authSecret, cl.authScheme
}

// GetClientID returns the client's connection details
func (cl *TransportClientBase) GetClientID() string {
	cl.mux.RLock()
	defer cl.mux.RUnlock()
	return cl.clientID
}

// GetClientCert returns the client certificate
func (cl *TransportClientBase) GetClientCert() *tls.Certificate {
	cl.mux.RLock()
	defer cl.mux.RUnlock()
	return cl.clientCert
}

// // GetConnectionID returns the client's connection details
func (cl *TransportClientBase) GetConnectionID() string {
	cl.mux.RLock()
	defer cl.mux.RUnlock()
	return cl.cid
}

// // GetConnectionStatus returns the current connection status
func (cl *TransportClientBase) GetConnectionStatus() api.ConnectionStatus {
	cl.mux.RLock()
	defer cl.mux.RUnlock()
	stat := cl.connectStatus
	return stat
}

// SetAuthToken sets the token credentials to use in Connect.
// This returns an error if a connection is in progress
func (cl *TransportClientBase) SetAuthToken(clientID string, token string, authScheme string) error {

	status := cl.GetConnectionStatus()
	if status == api.StatusConnecting {
		return fmt.Errorf("SetAuthToken: Already connected or connection in progress.")
	}
	cl.mux.Lock()
	cl.clientID = clientID
	cl.authScheme = authScheme
	cl.authSecret = token
	cl.mux.Unlock()
	return nil
}

// SetClientCert sets the client certificate for mutual authentication and
// determine the client ID to connect as.
//
// This validates the certificate against the root CA pool, if set.
//
// This returns an error if a connection is in progress or the certificate doesn't
// validate using the root CA pool.
func (cl *TransportClientBase) SetClientCert(clientCert *tls.Certificate) (err error) {
	status := cl.GetConnectionStatus()
	if status == api.StatusConnecting {
		return fmt.Errorf("SetClientCert: Connection in progress.")
	}
	// tell the client to use the certificate
	cl.mux.Lock()
	cl.clientCert = clientCert
	cl.mux.Unlock()

	//--- verify the client certificate against the available CA and extract the clientID
	// if a client cert is given then test if it is valid for our CA.
	// this detects problems with certs that can be hard to track down
	x509Cert, err := x509.ParseCertificate(clientCert.Certificate[0])
	if err == nil {

		// cert subject is clientID
		cl.clientID = x509Cert.Subject.CommonName

		// verify the validity of this certificate against the CA
		// without this one can spend a long time figuring out why the connection fails.
		if cl.rootCAs != nil {
			opts := x509.VerifyOptions{
				Roots:     cl.rootCAs,
				KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			}
			_, err = x509Cert.Verify(opts)
		}
	}
	return err
}

// update the connection status and publish an notification if it differs from the last status
// a 'lost' status is ignored if the current status is set to closed as it was intentional.
func (cl *TransportClientBase) SetConnectionStatus(
	newStatus api.ConnectionStatus, err error) {

	cl.mux.RLock()
	oldStatus := cl.connectStatus
	cl.mux.RUnlock()

	if newStatus == oldStatus {
		return
	} else if oldStatus == api.StatusClosed && newStatus == api.StatusLost {
		// already closed, don't re-send status lost
		return
	} else if newStatus == api.StatusLost {
		slog.Info("SetConnectionStatus: client connection lost", "previous status", oldStatus)
		// fail all outstanding RnR requests
		// cl.rnrChan.CloseAll() // should be done by caller
	}
	cl.mux.Lock()
	cl.connectStatus = newStatus
	ch := cl.connectHandler
	cl.mux.Unlock()

	// notify upstream of status change
	cellID := cl.GetThingID()
	// cid := cl.GetConnectionID()
	evName := api.ClientConnectionStatusEvent
	notif := msg.NewNotificationMessage(
		cellID, msg.AffordanceTypeEvent, cellID, evName, newStatus)
	cl.EmitNotification(notif)

	// invoke the callback after the notification so that the proper sequence is maintained
	// if the callback tries to reconnect.
	if ch != nil {
		ch(newStatus, cl.transportClient)
	}
}

// SetConnectHandler sets the callback to invoke when the connection status changes
func (cl *TransportClientBase) SetConnectHandler(
	h func(newStatus api.ConnectionStatus, c api.ITransportClient)) {
	cl.mux.Lock()
	defer cl.mux.Unlock()
	cl.connectHandler = h
}

// SetTransportClient sets the transport implementation to pass with connection callbacks
// For use by the transport implementation during creation.
func (cl *TransportClientBase) SetTransportClient(c api.ITransportClient) {
	cl.mux.Lock()
	defer cl.mux.Unlock()
	cl.transportClient = c
}

// NewTransportClientBase creates a new instance of the base for client connection
//
//	thingID is the instance thingID of this cell. "" to auto generate.
//	rootCAs root CA pool used to verify client auth certificate. It can be nil to ignore client cert validation.
//	rpcTimeout to use for messaging
func NewTransportClientBase(thingID string, rootCAs *x509.CertPool, rpcTimeout time.Duration) *TransportClientBase {
	m := &TransportClientBase{
		HiveCellBase: *cells.NewHiveCellBase(thingID, rpcTimeout),
		cid:          shortid.MustGenerate(),
		rootCAs:      rootCAs,
	}
	return m
}
