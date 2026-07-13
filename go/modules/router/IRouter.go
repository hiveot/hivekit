package router

import (
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
	// Setting a unique login credentials for each device is only realistic if only a few devices
	// are used, or if a manual OOB (out of band) method of provisioning is used. Eg a tool that
	// can upload a table and create accounts.
	//
	// As a 'convenient' way to provision a few devices, all devices can be set to use the same
	// credentials. In this case use "" to set the default clientID and secret. Note that all this
	// is horribly unsafe. It is safer to have devices use reverse connection with a gateway.
	//
	// deviceID is the connection account ID that was used when writing the device TD.
	// clientID is the ID the router service uses to identify itself as when connecting to the device.
	// secret is the auth token used to authenticate as the clientID.
	// secScheme indicates the type of credentials stored: SecSchemeBearer, ...
	//  See also SecSchemeXyz and https://www.w3.org/TR/wot-thing-description11/#securityscheme
	//
	// When routing a request to a Thing device, this secret is used to authenticate
	// the connection needed to pass the request. The TD describes the securityDefinitions
	// available.
	AddDeviceCredential(deviceID string, clientID, secret string, secScheme string)

	// Remove the secret to access a Thing
	DeleteThingCredential(thingID string)

	// Return a flag indicating whether the credentials are set for a Thing
	HasThingCredentials(thingID string) bool

	// Determine if the thing is reachable by the router.
	//
	// This returns true if a device connection is established by the router, or if
	// a reverse connection exists by the thing's deviceID.
	//
	// This determines the deviceID that manages the thing and looks up connections made
	// to or from the deviceID.
	// IsReachable(thingID string) bool

	// Return the ISO timestamp when the Thing was last seen by the router.
	// This returns an empty string if no known record exists.
	// LastSeen(thingID string) string

	// Set the communication timeout that is applied to new connections made by this module
	SetTimeout(time.Duration)
}
