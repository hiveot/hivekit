package directory_client

import (
	"context"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/directory"
	clientimpl "github.com/hiveot/hivekit/go/modules/directory/internal/clientimpl"
	"github.com/hiveot/hivekit/go/modules/transport/tlsclient"
	tls_client "github.com/hiveot/hivekit/go/modules/transport/tlsclient/client"
	"github.com/hiveot/hivekit/go/utils"
)

// Deprecated: this client should not be needed. TD Forms should contain all the
// information needed to map requests from the regular directory client to http
// requests.

// The DirectoryHttpClient is a client module for the Directory service using the REST API.
// It can be used to connect to a directory service and read its content.
//
// Note, this client is based on the directory specification found here:
//
//	https://www.w3.org/TR/wot-discovery/#exploration-directory-api-things
//
// Note: This does not use the Directory TD forms to determine endpoints.
// Instead, it uses fixed paths as per the specification along with the base
// URL from the directory TD.
//
// Intended for use by consumers of the directory to read TDs they have access to.
type DirectoryHttpClient struct {
	*modules.HiveModuleBase // clients can be used as modules
	cache                   *clientimpl.DirectoryCacheImpl

	// The TD of the directory itself containing the base URL
	dirTD *td.TD

	// the base path as published by the directory
	// directoryBasePath string

	// Connection to the directory for reading or update
	tlsClient tlsclient.ITLSClient

	timeout time.Duration
}

// A little helper to return the href and method for the requested action.
// This first tries using the directory forms and falls back to the default path
// and method provided.
func (cl *DirectoryHttpClient) _send(
	actionName string, defaultPath string, defaultMethod string, thingID string, payload []byte) (
	reply []byte, err error) {

	uriVars := map[string]string{
		td.UriVarName:      actionName,
		td.UriVarThingID:   thingID,
		td.UriVarOperation: td.OpInvokeAction,
	}

	// Use the form from the directory TD if available
	var method string
	f, href, err := cl.dirTD.GetFormHRef(td.OpInvokeAction, actionName, "https", uriVars)
	if err == nil {
		// use the form href and method
		method, _ = f.GetMethodName()
	} else {
		// no matching form, fallback to the hard-coded path from the spec using base-url
		href = fmt.Sprintf("%s%s", cl.dirTD.Base, defaultPath)
		method = defaultMethod
	}
	ctx := context.Background()
	reply, _, _, err = cl.tlsClient.Send(ctx, method, href, nil, payload, "")
	return reply, err
}

// AuthenticateWithToken creates a TLS client, connects to the directory, reads the directory's TD.
func (cl *DirectoryHttpClient) AuthenticateWithToken(clientID string, token string) error {

	// 1: connect to the directory and read its TD

	cl.tlsClient.AuthenticateWithToken(clientID, token)

	// 2: read its TD
	// GetTDPath := directory.WellKnownWoTPath
	// resp, status, err := cl.tlsClient.Get(GetTDPath)
	// _ = status
	// if err != nil {
	// 	return err
	// }
	// tdi, err := td.UnmarshalTD(string(resp))
	// if err != nil {
	// 	cl.tlsClient.Close()
	// 	return err
	// }
	// don't need a base if its the same as the directory client path
	// _ = tdi.Base
	// cl.directoryBasePath = tdi.Base
	return nil
}

// Return the local cache of Things
func (cl *DirectoryHttpClient) Cache() directory.IDirectoryCache {
	return cl.cache
}

// Close the connection with the directory and release resources.
func (cl *DirectoryHttpClient) Close() error {
	if cl.tlsClient != nil {
		cl.tlsClient.Close()
	}
	return nil
}

// CreateThing creates a new TD document in the remote directory.
func (cl *DirectoryHttpClient) CreateThing(tdJson string) error {

	cl.cache.ImportTDJson(tdJson)

	// validate the TD and determine the thingID needed in the path
	tdi, err := td.UnmarshalTD(tdJson)
	if err != nil {
		return err
	}

	defaultPath := fmt.Sprintf("/things/%s", tdi.ID)
	_, err = cl._send(
		directory.CreateThingAction, defaultPath, http.MethodPost, tdi.ID, []byte(tdJson))

	return err
}

// DeleteThing removes a TD document from the remote directory.
func (cl *DirectoryHttpClient) DeleteThing(thingID string) error {
	cl.cache.RemoveTD(thingID)

	defaultPath := fmt.Sprintf("/things/%s", thingID)
	_, err := cl._send(
		directory.DeleteThingAction, defaultPath, http.MethodDelete, thingID, nil)
	return err
}

// RetrieveAllThings retrieves a list of things to update the local directory
// This follows: https://w3c.github.io/wot-discovery/#exploration-directory-api-things-listing
// which requires the http get at /things?limit=...
func (cl *DirectoryHttpClient) RetrieveAllThings(offset int, limit int) ([]*td.TD, error) {

	var tdList []*td.TD

	defaultPath := fmt.Sprintf("/things?offset=%d&limit=%d", offset, limit)
	raw, err := cl._send(
		directory.RetrieveAllThingsAction, defaultPath, http.MethodGet, "", nil)

	if err != nil {
		return nil, err
	}
	var tdJsonList []string
	err = utils.DecodeAsObject(raw, &tdJsonList)

	tdList = make([]*td.TD, 0, len(tdJsonList))
	for _, tdJson := range tdJsonList {
		tdoc, err := cl.cache.ImportTDJson(tdJson)
		if err == nil {
			tdList = append(tdList, tdoc)
		}
	}

	return tdList, err
}

// RetrieveThing loads the TD from the directory.
// If the TD exists in the local chace it is returned instead.
// This follows: https://w3c.github.io/wot-discovery/#exploration-directory-api
// which requires the http get at /things/{id}
func (cl *DirectoryHttpClient) RetrieveThing(thingID string) (tdoc *td.TD, err error) {

	// first try the cache
	tdoc = cl.cache.GetThing(thingID)
	if tdoc != nil {
		return tdoc, nil
	}

	defaultPath := fmt.Sprintf("/things/%s", thingID)
	raw, err := cl._send(
		directory.RetrieveThingAction, defaultPath, http.MethodGet, thingID, nil)
	if err != nil {
		return nil, err
	}

	tdoc, err = cl.cache.ImportTDJson(string(raw))
	return tdoc, err
}

// set the TDD of the directory server
func (cl *DirectoryHttpClient) SetTDD(tdd *td.TD) {
	cl.dirTD = tdd
}

// UpdateThing updates the TD document in the remote directory.
// On success this updates the local directory.
// Intended for use by devices to publish their TD. Regular consumers
// are not allowed to do this.
//
//	tdjson is the TD document in JSON format
func (cl *DirectoryHttpClient) UpdateThing(tdJson string) error {

	// validate the TD and determine the thingID needed in the path
	tdi, err := td.UnmarshalTD(tdJson)
	if err != nil {
		return err
	}

	// Must match the href and method in the directory TD.

	defaultPath := fmt.Sprintf("/things/%s", tdi.ID)
	_, err = cl._send(
		directory.CreateThingAction, defaultPath, http.MethodPut, tdi.ID, []byte(tdJson))
	return err
}

// Deprecated: this client should not be needed as forms should be able to describe
// all request messages using the http-basic client.
//
// NewDirectoryHttpClient creates a new client for accessing a Thing Directory
// using the provided directory TDD.
//
// Call Connect() to connect to the directory service and Close() to release resources.
//
//	dirTD is the discovered TD of the directory; This must contain a base URL
//	caCert is the directory CA to match or nil to ignore this safety check for testing
func NewDirectoryHttpClient(dirTD *td.TD, caCert *x509.Certificate) *DirectoryHttpClient {

	if dirTD == nil {
		slog.Error("NewDirectoryHttpClient: no TD provided")
		return nil
	} else if dirTD.Base == "" {
		slog.Error("NewDirectoryHttpClient: TD has no Base URL")
		return nil
	}

	parts, err := url.Parse(dirTD.Base)
	if err != nil {
		slog.Error("NewDirectoryHttpClient: TD has no invalid Base URL: " + err.Error())
		return nil
	}
	tlsClient := tls_client.NewTLSClient(parts.Host, caCert, 0)

	cl := &DirectoryHttpClient{
		HiveModuleBase: modules.NewHiveModuleBase("", 0),
		cache:          clientimpl.NewDirectoryCacheImpl(),

		dirTD:     dirTD,
		timeout:   msg.DefaultRnRTimeout,
		tlsClient: tlsClient,
	}
	var _ directory.IDirectoryClient = cl // interface check
	return cl
}
