package authn

import (
	"log/slog"
	"path"

	"github.com/hiveot/hivekit/go/api"
)

// Session token validity for client types
const (
	DefaultConsumerTokenValidityDays = 30
	DefaultDeviceTokenValidityDays   = 90
	DefaultServiceTokenValidityDays  = 365
)

// supported password hashes
const (
	PWHASH_ARGON2id = "argon2id"
	PWHASH_BCRYPT   = "bcrypt" // fallback in case argon2id cannot be used
)

// DefaultPasswordFile is the default password filename for user account storage
const DefaultPasswordFile = "hiveot.passwd"

// AuthnConfig contains the auth service configuration
type AuthnConfig struct {
	// PasswordFile with the file based password store.
	// Use a relative path for using the default $HOME/stores/authn location
	// Use "" for default defined in 'authnstore.DefaultPasswordFile'
	PasswordFile string `yaml:"passwordFile,omitempty"`
	// Encryption of passwords: "argon2id" (default) or "bcrypt"
	Encryption string `yaml:"encryption,omitempty"`

	// predefined accounts
	// Location of client keys and auth tokens
	KeysDir string `yaml:"certsDir,omitempty"`

	// The admin account ID to create on startup. "" to not create an admin account.
	AdminUserID string `yaml:"adminAccountID,omitempty"`

	// The admin token validity when created/saved on startup. 0 to not create a token.
	AdminTokenValidityDays int `yaml:"adminTokenValidityDays,omitempty"`
}

// Setup ensures config is valid
//
//	storesDir is the default storage root directory ($HOME/stores/authn)
func (cfg *AuthnConfig) Setup(keysDir, storageDir string) {

	if cfg.PasswordFile == "" {
		cfg.PasswordFile = DefaultPasswordFile
	}
	if !path.IsAbs(cfg.PasswordFile) {
		cfg.PasswordFile = path.Join(storageDir, cfg.PasswordFile)
	}

	if cfg.Encryption == "" {
		cfg.Encryption = PWHASH_ARGON2id
	}
	if cfg.Encryption != PWHASH_BCRYPT && cfg.Encryption != PWHASH_ARGON2id {
		slog.Error("unknown password encryption method. Reverting to ARGON2id", "Encoding", cfg.Encryption)
		cfg.Encryption = PWHASH_ARGON2id
	}

	if cfg.AdminUserID == "" {
		cfg.AdminUserID = api.DefaultAdminUserID
	}

	cfg.KeysDir = keysDir

}

// NewAuthnConfig creates a new AuthnConfig with default values and applies the Setup to
// ensure the config is valid.
//
// Location of client keys and tokens
//
//	storesDir is the authentication data storage directory ($HOME/stores/authn)
func NewAuthnConfig(keysDir string, storageDir string) AuthnConfig {
	cfg := AuthnConfig{
		// default password encryption method
		Encryption: PWHASH_ARGON2id,
	}
	cfg.Setup(keysDir, storageDir)
	return cfg
}
