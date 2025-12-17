package wothttpbasic

// DefaultHttpBasicThingID is the default thingID of the http basic server module.
const DefaultHttpBasicThingID = "http-basic"

// Interface of the HttpBasic service
type IHttpBasicTransport interface {
	// Provide the URL to connect to the server
	GetConnectURL() string
}
