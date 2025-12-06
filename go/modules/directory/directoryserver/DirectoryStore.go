package directoryserver

import (
	"log/slog"

	"github.com/hiveot/hivekit/go/lib/buckets"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/wot/td"
)

// DirectoryStore provides a directory API to the underlying bucket store.
type DirectoryStore struct {
	store buckets.IBucket
}

// DeleteThing removes a Thing TD document from the store
func (svc *DirectoryStore) DeleteThing(thingID string) error {
	err := svc.store.Delete(thingID)
	return err
}

//func (svc *DirectoryService) QueryThings(
//	senderID string, args digitwin.DirectoryQueryTDsArgs) (tdDocuments []string, err error) {
//	//svc.DtwStore.QueryDTDs(args)
//	return nil, fmt.Errorf("Not yet implemented")
//}

// RetrieveThing returns a JSON encoded TD document
func (svc *DirectoryStore) RetrieveThing(thingID string) (tdJSON string, err error) {
	tdBytes, err := svc.store.Get(thingID)
	tdJSON = string(tdBytes)
	return tdJSON, err
}

// RetrieveAllThings returns a batch of TD documents
// This returns a list of JSON encoded digital twin TD documents
func (svc *DirectoryStore) RetrieveAllThings(offset int, limit int) (tdList []string, err error) {
	tdList = make([]string, 0)

	cursor, err := svc.store.Cursor()
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = directory.DefaultLimit
	}
	itemsToRead := limit
	if offset != 0 {
		_ = cursor.Skip(offset)
	}

	for {
		// read in batches of defaultLimit TD documents
		readCount := min(directory.DefaultLimit, itemsToRead)
		itemsToRead -= readCount
		tdmap, itemsRemaining := cursor.NextN(uint(readCount))
		for _, tdBin := range tdmap {
			tdList = append(tdList, string(tdBin))
		}
		if !itemsRemaining || itemsToRead <= 0 {
			break
		}
	}
	return tdList, err
}

// UpdateThing replaces the TD in the store.
// If the thing doesn't exist in the store it is added.
func (svc *DirectoryStore) UpdateThing(tdJson string) error {

	// validate the TD
	tdDoc, err := td.UnmarshalTD(tdJson)
	if err != nil {
		return err
	}

	slog.Info("UpdateThing", slog.String("thingID", tdDoc.ID))
	err = svc.store.Set(tdDoc.ID, []byte(tdJson))
	return err
}

// NewDirectoryServer creates a new instance of the directory service
// using the given store.
// This is based on the W3C WoT Discovery draft specification: https://w3c.github.io/wot-discovery
func NewDirectoryStore(store buckets.IBucket) *DirectoryStore {

	dirSrv := &DirectoryStore{
		store: store,
	}

	return dirSrv
}
