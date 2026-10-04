package authn_filestore

import (
	"github.com/hiveot/hivekit/go/cells/authn"
	"github.com/hiveot/hivekit/go/cells/authn/filestore/internal"
)

// open the authn user and password store
func OpenAuthnFileStore(passwdFile string, hashAlgo string) (authn.IAuthnStore, error) {
	store := internal.NewAuthnFileStoreImpl(passwdFile, hashAlgo)
	err := store.Open()
	return store, err
}
