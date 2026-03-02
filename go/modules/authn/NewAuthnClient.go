package authn

import (
	"crypto/x509"

	authnclient "github.com/hiveot/hivekit/go/modules/authn/client"
	"github.com/hiveot/hivekit/go/modules/clients"
)

// NewAuthnHttpClient creates an instance of the authentication client to login and obtain
// auth tokens using the http API.
//
//	serverURL is the host:port of the http server
//	caCert is the server CA
func NewAuthnHttpClient(serverURL string, caCert *x509.Certificate) *authnclient.AuthnHttpClient {
	cl := authnclient.NewAuthnHttpClient(serverURL, caCert)
	return cl
}

// Create a new instance of the authn messaging consumer client
// This clients uses the RRN messaging system to transport messages using the given
// consumer instance.
func NewAuthnUserMsgClient(co *clients.Consumer) *authnclient.AuthnUserMsgClient {
	cl := authnclient.NewAuthnUserMsgClient(co)
	return cl
}
