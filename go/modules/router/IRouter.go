package router

import (
	"crypto/tls"
	"time"

	"github.com/hiveot/hivekit/go/api"
)

const RouterModuleType = "router"

// The instance ThingID of the router. This must match its TD (if any)
const DefaultRouterThingID = "router"

type IRouterService interface {
	api.IHiveModule

	// Add the secret to access one or more Things on a device.
	//
	// If it already exists then it is replaced.
	// Used in combination with the Thing TD that describes how the secret is used to
	// authenticate with the device.
	//
	// deviceID is the thingID of the device connected to.
	// clientID is the ID the router service uses to identify itself as when connecting to the device.
	// secret is the auth token or cert tls PEM used to authenticate as the clientID.
	// secScheme indicates the type of credentials stored: SecSchemeBearer, ...
	//  See also SecSchemeXyz and https://www.w3.org/TR/wot-thing-description11/#securityscheme
	//  This also supports secSchemeCert although WoT removed it from the draft.
	//
	// When routing a request to a Thing device, this secret is used to authenticate
	// when creating a new connection. This is typically bearer token or client cert.
	AddDeviceCredential(deviceID string, clientID string, secret string, secScheme string)

	// Remove the secret to access a Thing
	DeleteThingCredential(thingID string)

	// Return a flag indicating whether the credentials are set for a Thing
	HasThingCredentials(thingID string) (credType string, found bool)

	// Set the default client certificate the router can use to authenticate new
	// client connections.
	// See also AddDeviceCredential() for a per-device authentication.
	SetClientCert(*tls.Certificate)

	// Enable/disable auto reconnect for new connections
	SetAutoReconnect(enable bool)

	// Set the communication timeout that is applied to new connections made by this module
	SetTimeout(time.Duration)
}
