package internal

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/hiveot/hivekit/go/utils"
	jsoniter "github.com/json-iterator/go"
)

const CredKeyFilename = "credstore.key"
const CredStoreFilename = "credstore.data"

// Login credentials for known devices
type ConnectCredentials struct {
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

	// credentials by connectURL
	connectCredentials map[string]ConnectCredentials
	// filepaths for encryption key and data storage
	keyFile     string
	storageFile string

	// the data file encryption key
	encKey string

	// The cache of client certificates.
	// Populated if a thing credential is used and a client cert is present.
	// clientCertCache map[string]*tls.Certificate
}

// Add the secret to access a Thing.
//
// use "" for connectURL to set the default credentials.
// When credType is Cert then secret must include the TLS certificate in PEM format
//
// connectURL for which the credentials apply
// creds credentials to authenticate with.
func (store *CredentialsStore) AddCredentials(connectURL string, creds ConnectCredentials) error {
	store.mux.Lock()
	defer store.mux.Unlock()
	store.connectCredentials[connectURL] = creds
	err := store.save()
	return err
}

// Close the store.
// If a storage file is set then save.
func (store *CredentialsStore) Close() {
	store.mux.Lock()
	defer store.mux.Unlock()

	// store.save()
}

// Remove the secret to access a Thing
func (store *CredentialsStore) DeleteCredentials(connectURL string) error {
	store.mux.Lock()
	defer store.mux.Unlock()
	delete(store.connectCredentials, connectURL)
	err := store.save()
	return err

}

// GetCredentials returns the credentials for connecting to a Thing.
//
// Note: Credentials are for connections so the thingID must be mapped to the
// connectURL which links to the credentials.
//
// If no credentials are set for the given connectURL then try the default credentials
// for connectURL "".
// If no credentials are found this returns found = false
func (store *CredentialsStore) GetCredentials(connectURL string) (
	clientID string, token string, credType string, found bool) {

	store.mux.RLock()
	defer store.mux.RUnlock()
	cred, found := store.connectCredentials[connectURL]
	// fallback to the default credentials if available
	if !found {
		cred, found = store.connectCredentials[""]
	}
	return cred.ClientID, cred.Secret, cred.CredType, found
}

// HasDeviceCredentials checks if credentials for a device exists.
// This returns the credential type and a flag is found or not found.
func (store *CredentialsStore) HasCredentials(connectURL string) (credType string, found bool) {
	store.mux.RLock()
	defer store.mux.RUnlock()
	cred, found := store.connectCredentials[connectURL]
	if !found {
		// try the fallback credentials
		cred, found = store.connectCredentials[""]
	}
	return cred.CredType, found
}

// Reload the credentials from the store into memory and replace the existing
// in-memory credentials.
//
// Returns an error if the file could not be opened.
func (store *CredentialsStore) load() (err error) {
	credentials := make(map[string]ConnectCredentials)

	var encKey []byte

	// a key file must exist
	if store.keyFile != "" {
		encKey, err = os.ReadFile(store.keyFile)
		if errors.Is(err, os.ErrNotExist) {
			// nothing to load
			return nil
		} else if err != nil {
			return err
		}
		store.encKey = string(encKey)
	}

	// only load if the filename is set
	if store.storageFile != "" {
		encryptedData, err := os.ReadFile(store.storageFile)
		if errors.Is(err, os.ErrNotExist) {
			// nothing to load
			err = nil
		} else if err != nil {
			err = fmt.Errorf("error reading Thing credentials file: %w", err)
			return err
		} else if len(encryptedData) == 0 {
			// nothing to do
		} else {
			dataBytes, err := utils.Decrypt(string(encryptedData), encKey)
			if err == nil {
				err = jsoniter.Unmarshal(dataBytes, &credentials)
			}
			if err != nil {
				err = fmt.Errorf("error while parsing credentials file: %w", err)
				slog.Error(err.Error())
			}
		}
	}
	if err == nil {
		store.connectCredentials = credentials
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
func (store *CredentialsStore) save() (err error) {
	// only save if the filename is set
	if store.storageFile == "" || store.keyFile == "" {
		return nil
	}

	// ensure the key exists
	if store.encKey == "" {
		storageDir := filepath.Dir(store.keyFile)
		err = os.MkdirAll(storageDir, 0700)
		if err != nil {
			return err
		}

		newKey := make([]byte, 24)
		_, _ = rand.Read(newKey)
		store.encKey = base64.StdEncoding.EncodeToString(newKey)
		// if the key file exists, it is replaced
		_ = os.Remove(store.keyFile)
		err = os.WriteFile(store.keyFile, []byte(store.encKey), 0400)
		if err != nil {
			slog.Error("save: Unable to save credentials key", "err", err.Error())
			return err
		}
	}

	storageDir := filepath.Dir(store.storageFile)
	err = os.MkdirAll(storageDir, 0700)
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
	pwData, err := json.Marshal(store.connectCredentials)
	if err == nil {
		var encData string
		encData, err = utils.Encrypt(pwData, []byte(store.encKey))

		if err == nil {
			_, err = file.Write([]byte(encData))
		}
	}

	return tempFileName, err
}

// Create a new credentials store
func NewCredentialsStore(storageDir string) *CredentialsStore {
	keyFile := ""
	storageFile := ""
	if storageDir != "" {
		storageFile = filepath.Join(storageDir, CredStoreFilename)
		keyFile = filepath.Join(storageDir, CredKeyFilename)
	}
	store := &CredentialsStore{
		keyFile:            keyFile,
		storageFile:        storageFile,
		connectCredentials: make(map[string]ConnectCredentials),
	}
	return store
}
