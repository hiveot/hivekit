package directoryclient

import (
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/modules/transports"
	tlsclient "github.com/hiveot/hivekit/go/modules/transports/httpserver/client"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot/td"
)

// Default timeout for directory operations
const defaultTimeout = 3 * time.Second

// The DirectoryHttpClient is a client for the Directory service using the REST API.
// It can be used to connect to a directory service and read its content.
// This implements the IDirectory interface.
//
// Note, this client is based on the directory specification found here:
//
//	https://www.w3.org/TR/wot-discovery/#exploration-directory-api-things
//
// Note: This does not use the Directory TD forms to generate as it needlessly
// complicates the implementation. Instead, it uses fixed paths as per the
// specification along with the base URL from the directory TD.
//
// Intended for use by consumers of the directory to read TDs they have access to.
type DirectoryHttpClient struct {
	modules.HiveModuleBase // clients can be used as modules

	// The TD of the directory itself containing the base URL
	// directoryTD *td.TD

	// the base path as published by the directory
	directoryBasePath string

	// Connection to the directory for reading or update
	tlsClient transports.ITlsClient

	timeout time.Duration
}

// Close the connection with the directory and release resources.
func (cl *DirectoryHttpClient) Close() error {
	if cl.tlsClient != nil {
		cl.tlsClient.Close()
	}
	return nil
}

// ConnectWithToken creates a TLS client, connects to the directory, reads the directory's TD.
func (cl *DirectoryHttpClient) ConnectWithToken(clientID string, token string) error {

	// 1: connect to the directory and read its TD
	cl.tlsClient.ConnectWithToken(clientID, token)

	// 2: read its TD
	GetTDPath := directory.WellKnownWoTPath
	resp, status, err := cl.tlsClient.Get(GetTDPath)
	_ = status
	if err != nil {
		return err
	}
	tdi, err := td.UnmarshalTD(string(resp))
	if err != nil {
		cl.tlsClient.Close()
		return err
	}
	// don't need a base if its the same as the directory client path
	_ = tdi.Base
	// cl.directoryBasePath = tdi.Base
	return nil
}

// CreateThing creates a new TD document in the remote directory.
func (cl *DirectoryHttpClient) CreateThing(tdJson string) error {

	// validate the TD and determine the thingID needed in the path
	tdi, err := td.UnmarshalTD(tdJson)
	if err != nil {
		return err
	}

	// Must match the href and method in the directory TD.
	basePath := cl.directoryBasePath
	updatePath := fmt.Sprintf("%s/things/%s", basePath, tdi.ID)
	data, status, err := cl.tlsClient.Put(updatePath, []byte(tdJson))
	_ = status
	_ = data
	return err
}

// DeleteThing removes a TD document from the remote directory.
func (cl *DirectoryHttpClient) DeleteThing(thingID string) error {
	basePath := cl.directoryBasePath
	deletePath := fmt.Sprintf("%s/things/%s", basePath, thingID)
	_, err := cl.tlsClient.Delete(deletePath)
	return err
}

// RetrieveAllThings retrieves a list of things to update the local directory
// This follows: https://w3c.github.io/wot-discovery/#exploration-directory-api-things-listing
// which requires the http get at /things?limit=...
func (cl *DirectoryHttpClient) RetrieveAllThings(offset int, limit int) ([]string, error) {

	basePath := cl.directoryBasePath
	listPath := fmt.Sprintf("%s/things?offset=%d&limit=%d", basePath, offset, limit)
	raw, stat, err := cl.tlsClient.Get(listPath)
	_ = stat
	if err != nil {
		return nil, err
	}
	var tdJsonList []string
	err = utils.DecodeAsObject(raw, &tdJsonList)
	return tdJsonList, err
}

// RetrieveThing refreshes the cached TD from the directory
// This follows: https://w3c.github.io/wot-discovery/#exploration-directory-api
// which requires the http get at /things/{id}
func (cl *DirectoryHttpClient) RetrieveThing(thingID string) (string, error) {

	basePath := cl.directoryBasePath
	retrievePath := fmt.Sprintf("%s/things/%s", basePath, thingID)
	tdJson, stat, err := cl.tlsClient.Get(retrievePath)
	_ = stat
	if err != nil {
		return "", err
	}
	return string(tdJson), err
}

// UpdateThing updates the TD document in the remote directory.
// On success this updates the local directory.
// Intended for use by Thing agents to publish their TD. Regular consumers
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
	basePath := cl.directoryBasePath
	updatePath := fmt.Sprintf("%s/things/%s", basePath, tdi.ID)
	data, status, err := cl.tlsClient.Put(updatePath, []byte(tdJson))
	_ = status
	_ = data
	return err
}

// NewDirectoryHttpClient creates a new client for accessing a Thing Directory.
//
// Call Connect() to connect to the directory service and Close() to release resources.
//
//	thingID is the unique ID of the certificate service instance
//	sink is the handler that passes requests to the service and receives notifications.
func NewDirectoryHttpClient(serverURL string, caCert *x509.Certificate) *DirectoryHttpClient {

	parts, err := url.Parse(serverURL)
	if err != nil {
		slog.Error("NewAuthnClient: invalid server URL", "err", err.Error())
		return nil
	}

	tlsClient := tlsclient.NewTLSClient(parts.Host, nil, caCert, 0)

	cl := &DirectoryHttpClient{
		timeout:   transports.DefaultRpcTimeout,
		tlsClient: tlsClient,
	}

	var _ directory.IDirectoryModule = cl // API check

	return cl
}
