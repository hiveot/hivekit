package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	jsoniter "github.com/json-iterator/go"
)

const CredStoreFilename = "credstore.data"

// Login credentials for known devices
type ThingAccount struct {
	ClientID string `json:"clientID"`

	// Secret password or token
	Secret string `json:"secret"`

	// the credentials type as used in td SecurityScheme.Scheme
	// eg, apikey, digest, bearer, ...
	CredType string `json:"type"`
}

// Device credentials storage
type CredentialsStore struct {
	mux sync.RWMutex
	// device account by device thingID
	thingAccounts map[string]ThingAccount
	storageFile   string
}

// Add the secret to access a Thing.
func (store *CredentialsStore) AddCredentials(
	thingID string, clientID string, secret string, credType string) {
	store.mux.Lock()
	defer store.mux.Unlock()
	store.thingAccounts[thingID] = ThingAccount{
		ClientID: clientID,
		Secret:   secret,
		CredType: credType,
	}
}

// Close the store.
// If a storage file is set then save.
func (store *CredentialsStore) Close() {
	store.mux.Lock()
	defer store.mux.Unlock()

	store.save()
}

// Remove the secret to access a Thing
func (store *CredentialsStore) DeleteCredentials(thingID string) {
	store.mux.Lock()
	defer store.mux.Unlock()
	delete(store.thingAccounts, thingID)
}

// Obtain the connection credentials for connection to the GetCredentials
func (store *CredentialsStore) GetCredentials(thingID string) (
	clientID string, token string, credType string, err error) {

	store.mux.RLock()
	defer store.mux.RUnlock()
	acct, found := store.thingAccounts[thingID]
	if !found {
		return "", "", credType, fmt.Errorf("No credentials for thing with ID '%s'", thingID)
	}
	return acct.ClientID, acct.Secret, acct.CredType, err
}

// HasDeviceCredentials returns a flag if credentials are set for a Thing
func (store *CredentialsStore) HasCredentials(thingID string) bool {
	store.mux.RLock()
	defer store.mux.RUnlock()
	_, found := store.thingAccounts[thingID]
	return found
}

// Reload the credentials from the store into memory and replace the existing
// in-memory credentials.
//
// Returns an error if the file could not be opened.
func (store *CredentialsStore) load() (err error) {
	accounts := make(map[string]ThingAccount)

	// only load if the filename is set
	if store.storageFile != "" {
		dataBytes, err := os.ReadFile(store.storageFile)
		if errors.Is(err, os.ErrNotExist) {
			// nothing to load
			err = nil
		} else if err != nil {
			err = fmt.Errorf("error reading Thing credentials file: %w", err)
			return err
		} else if len(dataBytes) == 0 {
			// nothing to do
		} else {
			err = jsoniter.Unmarshal(dataBytes, &accounts)
			if err != nil {
				err = fmt.Errorf("error while parsing password file: %w", err)
			}
		}
	}
	if err == nil {
		store.thingAccounts = accounts
	}
	return err
}

// Open the store.
// This reads the password file and subscribes to file changes
// If no storage directory is set then this starts with an empty store.
func (store *CredentialsStore) Open() (err error) {
	store.mux.Lock()
	defer store.mux.Unlock()
	err = store.load()
	return err
}

// save the credentials to file
// if the storage folder doesn't exist it will be created
func (store *CredentialsStore) save() error {
	// only save if the filename is set
	if store.storageFile == "" {
		return nil
	}

	// ensure the location exists
	storageDir := filepath.Dir(store.storageFile)
	err := os.MkdirAll(storageDir, 0700)
	if err != nil {
		return err
	}
	tmpPath, err := store.writeToTempFile(storageDir)
	if err != nil {
		err = fmt.Errorf("writing password file to temp failed: %w", err)
		return err
	}
	// rename the temp file if it was successfully created
	err = os.Rename(tmpPath, store.storageFile)
	if err != nil {
		err = fmt.Errorf("rename to password file failed: %w", err)
		return err
	}
	return err
}

// WriteToTempFile write the credentials to a temp file of the storage directory
// This returns the name of the new temp file.
func (store *CredentialsStore) writeToTempFile(storageDir string) (tempFileName string, err error) {

	file, err := os.CreateTemp(storageDir, "hive-tmp-credfile")

	// file, err := os.OpenFile(path, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		err := fmt.Errorf("failed open temp password file: %s", err)
		return "", err
	}
	tempFileName = file.Name()

	defer file.Close()
	pwData, err := json.Marshal(store.thingAccounts)
	if err == nil {
		_, err = file.Write(pwData)
	}

	return tempFileName, err
}

// Create a new credentials store
func NewCredentialsStore(storageDir string) *CredentialsStore {
	storageFile := ""
	if storageDir != "" {
		storageFile = filepath.Join(storageDir, CredStoreFilename)
	}
	store := &CredentialsStore{
		storageFile: storageFile,
	}
	return store
}
