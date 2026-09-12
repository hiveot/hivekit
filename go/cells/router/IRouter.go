package router

import (
	"crypto/tls"
	"time"

	"github.com/hiveot/hivekit/go/api"
)

const RouterCellType = "router"

// The instance ThingID of the router. This must match its TD (if any)

type IRouterService interface {
	api.IHiveCell

	// Add the secret to access one or more Things.
	//
	// This stores the credentials using the connect URL, eg all Things that use
	// the same connection endpoint.
	//
	// If no thingID is provided then use these credentials as the default. Intended
	// for testing only as this exposes the token to all servers it tries connecting to.
	//
	// If it already exists then it is replaced.
	// Used in combination with the Thing TD that describes how the secret is used to
	// authenticate with the device.
	//
	// thingID is the thingID of the device connecting to.
	// clientID is the ID the router service uses to identify itself as when connecting to the device.
	// secret is the auth token or cert tls PEM used to authenticate as the clientID.
	// secScheme indicates the type of credentials stored: SecSchemeBearer, ...
	//  See also SecSchemeXyz and https://www.w3.org/TR/wot-thing-description11/#securityscheme
	//  This also supports secSchemeCert although WoT removed it from the draft.
	//
	// When routing a request to a Thing device, this secret is used to authenticate
	// when creating a new connection. This is typically bearer token or client cert.
	AddCredentials(thingID string, clientID string, secret string, secScheme string)

	// Remove the secret to access a Thing.
	// The TD of the thing has to be available.
	DeleteCredentials(thingID string)

	// Return a flag indicating whether the credentials are set for a Thing
	// The TD of the thing has to be available.
	HasCredentials(thingID string) (credType string, found bool)

	// Set the default client certificate the router can use to authenticate new
	// client connections.
	// See also AddDeviceCredential() for a per-device authentication.
	SetClientCert(*tls.Certificate)

	// Enable/disable auto reconnect for new connections
	SetAutoReconnect(enable bool)

	// Set the communication timeout that is applied to new connections made by this service
	SetTimeout(time.Duration)
}
