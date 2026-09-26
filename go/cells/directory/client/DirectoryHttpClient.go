package directory_client

import (
	"crypto/x509"
	"fmt"
	"strconv"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells/directory"
	clientimpl "github.com/hiveot/hivekit/go/cells/directory/internal/clientimpl"
	httpbasic_client "github.com/hiveot/hivekit/go/cells/transport/httpbasic/client"
)

// The DirectoryHttpClient is a client for the Directory service using the REST API.
//
// This simply takes the RRN directory client and attaches it to the http-basic
// transport using the directory TDD.
//
// Intended for use by consumers of the directory to read TDs they have access to.
type DirectoryHttpClient struct {
	// the httpbasic converts RRN requests to http using the directory TDD
	api.ITransportClient

	cache *clientimpl.DirectoryCacheImpl

	// The TD of the directory itself containing the base URL
	dirTD *td.TD
}

// Return the local cache of Things
func (cl *DirectoryHttpClient) Cache() directory.IDirectoryCache {
	return cl.cache
}

// Create a Thing in the directory
func (cl *DirectoryHttpClient) CreateThing(tdoc *td.TD) (err error) {

	// do not marshal the TD twice, so send it as an object.
	// the thingID is the ID to be created.
	err = cl.Rpc(td.OpInvokeAction, tdoc.ID, directory.CreateThingAction, tdoc, nil)
	return err
}

// Create a Thing in the directory
func (cl *DirectoryHttpClient) DeleteThing(thingID string) (err error) {
	err = cl.Rpc(
		td.OpInvokeAction, thingID, directory.DeleteThingAction, nil, nil)
	cl.cache.RemoveTD(thingID)
	return err
}

// Get the TD for the given thing ID
func (cl *DirectoryHttpClient) GetTD(thingID string) *td.TD {
	tdoc, err := cl.RetrieveThing(thingID)
	_ = err
	return tdoc
}

// RetrieveAllThings retrieves a list of things to update the local directory
// This follows: https://w3c.github.io/wot-discovery/#exploration-directory-api-things-listing
// which requires the http get at /things?limit=...
func (cl *DirectoryHttpClient) RetrieveAllThings(offset int, limit int) ([]*td.TD, error) {

	var tdList []*td.TD

	// NOTE: this is a dependency on the URI variables in the path
	args := map[string]string{}
	args["offset"] = strconv.Itoa(offset)
	args["limit"] = strconv.Itoa(limit)
	var output []string

	err := cl.Rpc(
		td.OpInvokeAction, "", directory.RetrieveAllThingsAction, args, &output)
	if err != nil {
		return nil, err
	}

	tdList = make([]*td.TD, 0, len(output))
	for _, tdJson := range output {
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
	err = cl.Rpc(
		td.OpInvokeAction, thingID, directory.RetrieveThingAction, nil, &tdoc)
	if err != nil {
		return nil, err
	}

	cl.cache.ImportTD(tdoc)
	return tdoc, err
}

// set the TDD of the directory server
func (cl *DirectoryHttpClient) SetTDD(tdd *td.TD) {
	cl.dirTD = tdd
}

// Create a Thing in the directory
func (cl *DirectoryHttpClient) UpdateThing(tdoc *td.TD) (err error) {

	// do not marshal the TD twice, so send it as an object.
	// the thingID is the ID to be created.
	err = cl.Rpc(td.OpInvokeAction, tdoc.ID, directory.UpdateThingAction, tdoc, nil)
	return err
}

// NewDirectoryHttpClient creates a new client for accessing a Thing Directory
// over http using the provided directory TDD.
//
// Call Connect() to connect to the directory service and Close() to release resources.
//
//	dirTD is the discovered TD of the directory
//	rootCAs are the available CAs to validate the directory server certificate
//	timeout is the timeout for sending messages
func NewDirectoryHttpClient(dirTD *td.TD, rootCAs *x509.CertPool, timeout time.Duration) (*DirectoryHttpClient, error) {

	if dirTD == nil {
		err := fmt.Errorf("NewDirectoryHttpClient: no TD provided")
		return nil, err
	}
	httpClient, err := httpbasic_client.NewHttpBasicClient(dirTD, rootCAs)
	if err != nil {
		return nil, err
	}
	httpClient.SetTimeout(timeout)

	cl := &DirectoryHttpClient{
		ITransportClient: httpClient,
		cache:            clientimpl.NewDirectoryCacheImpl(),
		dirTD:            dirTD,
	}
	var _ api.IConnection = cl            // interface check
	var _ api.IHiveCell = cl              // interface check
	var _ directory.IDirectoryClient = cl // interface check
	return cl, nil
}
