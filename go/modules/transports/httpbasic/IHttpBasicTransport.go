package httpbasic

// DefaultHttpBasicThingID is the default thingID of the http basic server module.
const DefaultHttpBasicThingID = "http-basic"

const (
	// ConnectionIDHeader is intended for linking return channels to requests.
	// intended for separated return channel like sse.
	ConnectionIDHeader = "cid"
	// CorrelationIDHeader is the header to be able to link requests to out of band responses
	// tentative as it isn't part of the wot spec
	CorrelationIDHeader = "correlationID"

	// HttpPostLoginPath is the fixed authentication endpoint of the hub
	HttpPostLoginPath   = "/authn/login"
	HttpPostLogoutPath  = "/authn/logout"
	HttpPostRefreshPath = "/authn/refresh"
	HttpGetPingPath     = "/ping"

	// The generic path for thing operations over http using URI variables
	HttpBaseFormOp                   = "/things"
	HttpBasicAffordanceOperationPath = "/things/{operation}/{thingID}/{name}"
	HttpBasicThingOperationPath      = "/things/{operation}/{thingID}"
	HttpBasicOperationURIVar         = "operation"
	HttpBasicThingIDURIVar           = "thingID"
	HttpBasicNameURIVar              = "name"
)

// Interface of the HttpBasic service
type IHttpBasicTransport interface {
	// Provide the URL to connect to the server
	GetConnectURL() string
}
