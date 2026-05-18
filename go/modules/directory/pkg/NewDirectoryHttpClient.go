package directorypkg

import (
	"context"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/modules/transports"
	httptransportpkg "github.com/hiveot/hivekit/go/modules/transports/httptransport/pkg"
	"github.com/hiveot/hivekit/go/utils"
)

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
	modules.HiveModuleBase // clients can be used as modules

	// The TD of the directory itself containing the base URL
	dirTD *td.TD

	// the base path as published by the directory
	// directoryBasePath string

	// Connection to the directory for reading or update
	tlsClient transports.ITLSClient

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

// CreateThing creates a new TD document in the remote directory.
func (cl *DirectoryHttpClient) CreateThing(tdJson string) error {

	// validate the TD and determine the thingID needed in the path
	// tdoc, err := td.UnmarshalTD(tdJson)
	// if err != nil {
	// 	return err
	// }

	f, href, err := cl.dirTD.GetFormHRef(
		td.OpInvokeAction, directory.ActionCreateThing,
		[]string{transports.ProtocolSchemeWotHttpBasic}, nil)

	if err != nil {
		return fmt.Errorf("CreateThing: operation %s.%s has no matching form for http protocol in the td",
			td.OpInvokeAction, directory.ActionCreateThing)
	}
	ctx := context.Background()
	methodName, _ := f.GetMethodName()
	data, status, _, err := cl.tlsClient.Send(ctx, methodName, href, nil, []byte(tdJson), "")
	_ = status
	_ = data
	return err
}

// DeleteThing removes a TD document from the remote directory.
func (cl *DirectoryHttpClient) DeleteThing(thingID string) error {
	basePath := cl.dirTD.Base
	deletePath := fmt.Sprintf("%s/things/%s", basePath, thingID)
	_, err := cl.tlsClient.Delete(deletePath)
	return err
}

// RetrieveAllThings retrieves a list of things to update the local directory
// This follows: https://w3c.github.io/wot-discovery/#exploration-directory-api-things-listing
// which requires the http get at /things?limit=...
func (cl *DirectoryHttpClient) RetrieveAllThings(offset int, limit int) ([]string, error) {

	basePath := cl.dirTD.Base
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
// which requires the http get at /things/{thingID}
func (cl *DirectoryHttpClient) RetrieveThing(thingID string) (string, error) {

	basePath := cl.dirTD.Base
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
	basePath := cl.dirTD.Base
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
	tlsClient := httptransportpkg.NewHttpTransportClient(parts.Host, caCert, 0)

	cl := &DirectoryHttpClient{
		dirTD:     dirTD,
		timeout:   msg.DefaultRnRTimeout,
		tlsClient: tlsClient,
	}
	return cl
}
