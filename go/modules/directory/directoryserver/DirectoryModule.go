// package directoryserver with the module interface
package directoryserver

import (
	_ "embed"

	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hivekit/go/lib/buckets"
	"github.com/hiveot/hivekit/go/lib/messaging"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot"
	"github.com/hiveot/hivekit/go/wot/td"
	jsoniter "github.com/json-iterator/go"
	"gopkg.in/yaml.v3"
)

const moduleName = "directory"

// Embed the directory TM
//
//go:embed directory-tm.json
var DirectoryTMJson []byte

// DirectoryServer is a module for accessing a directory store.
//
// The module is configured using yaml.
// Any database supporting the IBucketStore interface can be used as underlying storage.
// A REST API is supported for accessing the directory over HTTP as per WoT specification
// using the endpoints defined in the TD. (directory.json)
type DirectoryServer struct {
	modules.HiveModuleBase

	// Optional override of the default data store. Default is moduleName.
	BucketName string `yaml:"bucketName"`

	// internal variables
	// the bucket store instance for this module chain
	bucketStore buckets.IBucketStore
	// router for rest api
	router *chi.Mux
	// the directory store itself
	store *DirectoryStore
	// the API servers if enabled
	restAPI *DirectoryRestAPI
}

// GetTD this module returns the directory TD
// It includes forms for http access through the REST API.
func (m *DirectoryServer) GetTD() (tdDoc *td.TD) {
	tdJson := DirectoryTMJson
	jsoniter.Unmarshal(tdJson, &tdDoc)

	return tdDoc
}

// HandleRequest for properties or actions
// If the request has an unknown operation it is forwarded to the sinks.
func (m *DirectoryServer) HandleRequest(req *messaging.RequestMessage) (resp *messaging.ResponseMessage) {
	if req.Operation == wot.OpInvokeAction {
		// directory specific operations
		switch req.Name {
		case directory.ActionCreateThing:
			resp = m.UpdateThing(req)
		case directory.ActionDeleteThing:
			resp = m.DeleteThing(req)
		case directory.ActionRetrieveThing:
			resp = m.RetrieveThing(req)
		case directory.ActionRetrieveAllThings:
			resp = m.RetrieveAllThings(req)
		case directory.ActionUpdateThing:
			resp = m.UpdateThing(req)
		}
	}
	if resp == nil {
		resp = m.HiveModuleBase.HandleRequest(req)
	}
	return resp
}

// DeleteThing remvoes a thing in the store
// req.Input is a string containing the Thing ID
func (m *DirectoryServer) DeleteThing(req *messaging.RequestMessage) (resp *messaging.ResponseMessage) {
	var thingID string
	err := utils.Decode(req.Input, &thingID)
	if err == nil {
		err = m.store.DeleteThing(thingID)
	}
	resp = req.CreateResponse(nil, err)
	return resp
}

// RetrieveAllThings returns a list of things
// Input: {offset, limit}
func (m *DirectoryServer) RetrieveAllThings(req *messaging.RequestMessage) (resp *messaging.ResponseMessage) {
	var tdList []string
	var err error
	var args directory.RetrieveAllThingsArgs

	err = utils.Decode(req.Input, &args)
	if err == nil {
		tdList, err = m.store.RetrieveAllThings(args.Offset, args.Limit)
	}
	resp = req.CreateResponse(tdList, err)
	return resp
}

// RetrieveThing gets the TD JSON for the given thingID from the directory store.
func (m *DirectoryServer) RetrieveThing(req *messaging.RequestMessage) (resp *messaging.ResponseMessage) {
	var thingID string
	var tdJSON string
	err := utils.Decode(req.Input, &thingID)
	if err == nil {
		tdJSON, err = m.store.RetrieveThing(thingID)
	}
	resp = req.CreateResponse(tdJSON, err)
	return resp
}

// Start readies the module for use using the given yaml configuration.
//
// This:
// - opens the bucket store using the configured name.
// - starts listening on the http server
//
// yamlConfig contains the settings to use.
func (m *DirectoryServer) Start(yamlConfig string) error {
	// config determines the store and server components to use (future)
	err := yaml.Unmarshal([]byte(yamlConfig), &m)
	if err != nil {
		return err
	}
	bucket := m.bucketStore.GetBucket(m.BucketName)
	m.store = NewDirectoryStore(bucket)
	m.restAPI = NewDirectoryRestAPI(m.store, m.router)

	return err
}

// Stop any running actions
func (m *DirectoryServer) Stop() {
}

// UpdateThing updates a new thing in the store
// req.Input is a string containing the TD JSON
func (m *DirectoryServer) UpdateThing(req *messaging.RequestMessage) (resp *messaging.ResponseMessage) {
	var tdJSON string
	err := utils.Decode(req.Input, &tdJSON)
	if err == nil {
		err = m.store.UpdateThing(tdJSON)
	}
	resp = req.CreateResponse(nil, err)
	return resp
}

// Create a new directory module. On start this creates the server and store.
// bucketStore is the store to use for this module chain.
func NewDirectoryServer(bucketStore buckets.IBucketStore, router *chi.Mux) *DirectoryServer {

	m := &DirectoryServer{
		// default module configuration
		BucketName: moduleName,
		//
		bucketStore: bucketStore,
		router:      router,
	}
	return m
}
