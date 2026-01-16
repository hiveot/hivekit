// Package module with IDirectoryService implementation
package module

import (
	"log/slog"

	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/wot/td"
)

// CreateThing adds or replaces the TD in the store.
func (svc *DirectoryModule) CreateThing(tdJson string) error {

	// validate the TD
	tdi, err := td.UnmarshalTD(tdJson)
	if err != nil {
		return err
	}

	slog.Info("CreateThing", slog.String("thingID", tdi.ID))
	err = svc.bucket.Set(tdi.ID, []byte(tdJson))
	return err
}

// DeleteThing removes a Thing TD document from the store
func (svc *DirectoryModule) DeleteThing(thingID string) error {
	err := svc.bucket.Delete(thingID)
	return err
}

//func (svc *DirectoryService) QueryThings(
//	senderID string, args digitwin.DirectoryQueryTDsArgs) (tdDocuments []string, err error) {
//	//svc.DtwStore.QueryDTDs(args)
//	return nil, fmt.Errorf("Not yet implemented")
//}

// RetrieveThing returns a JSON encoded TD document
func (svc *DirectoryModule) RetrieveThing(thingID string) (tdJSON string, err error) {
	tdBytes, err := svc.bucket.Get(thingID)
	tdJSON = string(tdBytes)
	return tdJSON, err
}

// RetrieveAllThings returns a batch of TD documents
// This returns a list of JSON encoded digital twin TD documents
func (svc *DirectoryModule) RetrieveAllThings(offset int, limit int) (tdList []string, err error) {
	tdList = make([]string, 0)

	cursor, err := svc.bucket.Cursor()
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

// Stop the service and close the storage bucket
// The bucketStore itself is not closed on Stop.
// func (svc *DirectoryModule) Stop() error {
// 	err := svc.bucket.Close()
// 	return err
// }

// UpdateThing replaces the TD in the store.
// If the thing doesn't exist in the store it is added.
func (svc *DirectoryModule) UpdateThing(tdJson string) error {

	// validate the TD
	tdi, err := td.UnmarshalTD(tdJson)
	if err != nil {
		return err
	}

	slog.Info("UpdateThing", slog.String("thingID", tdi.ID))
	err = svc.bucket.Set(tdi.ID, []byte(tdJson))
	return err

}

// StartDirectoryService creates a new instance of the directory service and open
// the storage bucket for use. Call Stop() to close the bucket(s) when done.
// The bucketStore itself is not closed on Stop.
//
// bucketStore is the store to open/create the directory bucket into.
// bucketName is the name of the bucket to use. Default is "directory".
// func StartDirectoryService(bucketStore bucketstore.IBucketStore, bucketName string) (*DirectoryService, error) {
// 	if bucketName == "" {
// 		bucketName = "directory"
// 	}
// 	dirSrv := &DirectoryService{
// 		bucket:     bucketStore.GetBucket(bucketName),
// 		bucketName: bucketName,
// 	}

// 	return dirSrv, nil
// }
