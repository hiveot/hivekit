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
type ThingCredentials struct {
	ClientID string `json:"clientID"`

	// Secret password or token
	Secret string `json:"secret"`

	// The credentials type of the secret as defined in td SecurityScheme.Scheme
	// eg, apikey, digest, bearer, ...
	CredType string `json:"type"`

	// Optional CA certificate for use with this client. It will be added to the app root cert pool.
	// This is needed when the device uses its own CA not in the app cert pool
	// CaCertPEM string `json:"caCert"`

	// Optional client certificate for mutual authentication
	// This is only usable when the device supports client certificate authentication
	// ClientCertPEM string `json:"clientCert"`
}

// Device credentials storage
// Authentication credentials can be stored for devices by their thingID
type CredentialsStore struct {
	mux sync.RWMutex

	// credentials by device thingID
	thingCredentials map[string]ThingCredentials
	//
	storageFile string

	// The cache of client certificates.
	// Populated if a thing credential is used and a client cert is present.
	// clientCertCache map[string]*tls.Certificate
}

// Add the secret to access a Thing.
//
// use "" for thingID to set the default credentials.
// When credType is Cert then secret must include the TLS certificate in PEM format
//
// thingID for which the credentials apply
// creds credentials to authenticate with.
func (store *CredentialsStore) AddCredentials(thingID string, creds ThingCredentials) {
	store.mux.Lock()
	defer store.mux.Unlock()
	store.thingCredentials[thingID] = creds
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
	delete(store.thingCredentials, thingID)
}

// GetCredentials returns the account credentials for connecting to a Thing.
//
// If no credentials are set for the given thingID then try the default credentials
// for thingID "".
// If no credentials are found this returns an error
func (store *CredentialsStore) GetCredentials(thingID string) (
	clientID string, token string, credType string, found bool) {

	store.mux.RLock()
	defer store.mux.RUnlock()
	cred, found := store.thingCredentials[thingID]
	// fallback to the default credentials if available
	if !found {
		cred, found = store.thingCredentials[""]
	}
	return cred.ClientID, cred.Secret, cred.CredType, found
}

// HasDeviceCredentials checks if credentials for a device exists.
// This returns the credential type and a flag is found or not found.
func (store *CredentialsStore) HasCredentials(thingID string) (credType string, found bool) {
	store.mux.RLock()
	defer store.mux.RUnlock()
	cred, found := store.thingCredentials[thingID]
	if !found {
		// try the fallback credentials
		cred, found = store.thingCredentials[""]
	}
	return cred.CredType, found
}

// Reload the credentials from the store into memory and replace the existing
// in-memory credentials.
//
// Returns an error if the file could not be opened.
func (store *CredentialsStore) load() (err error) {
	thingCredentials := make(map[string]ThingCredentials)

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
			err = jsoniter.Unmarshal(dataBytes, &thingCredentials)
			if err != nil {
				err = fmt.Errorf("error while parsing password file: %w", err)
			}
		}
	}
	if err == nil {
		store.thingCredentials = thingCredentials
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

// save the credentials to file.
// if the storage folder doesn't exist it will be created.
// FIXME: this file should be encrypted!
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
	pwData, err := json.Marshal(store.thingCredentials)
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
