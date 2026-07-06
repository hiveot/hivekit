package api

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/utils"
	"gopkg.in/yaml.v3"
)

// certificate file names
const (
	// CA key for self signed certificates
	DefaultCaKeyFile = "caKey.pem"

	// clients and server need the CA
	DefaultCaCertFile = "caCert.pem"

	// client side certificate
	DefaultCertFileSuffix = "Cert.pem"
	DefaultKeyFileSuffix  = "Key.pem"
)

// AppEnvironment holds the running environment naming conventions.
// Intended for devices, services, or client applications.
// This contains folder locations, CA certificate and application clientID
type AppEnvironment struct {
	// Directories
	BinDir     string `yaml:"binDir,omitempty"`     // Application binary folder, e.g. launcher, cli, ...
	PluginsDir string `yaml:"pluginsDir,omitempty"` // Plugin folder
	HomeDir    string `yaml:"homeDir,omitempty"`    // Home folder, default this is the parent of bin, config, certs and logs
	ConfigDir  string `yaml:"configDir,omitempty"`  // config folder with application and configuration files
	ConfigFile string `yaml:"configFile,omitempty"` // Application configuration file. Default is clientID.yaml
	CertsDir   string `yaml:"certsDir,omitempty"`   // Certificates and keys location
	LogsDir    string `yaml:"logsDir,omitempty"`    // Logging output
	LogLevel   string `yaml:"logLevel,omitempty"`   // logging level: error, warning, info, debug
	StoresDir  string `yaml:"storesDir,omitempty"`  // Root of the service stores

	// For clients: forced server to connect to: scheme://host/path, or "" for auto.
	// This can be useful to point to a gateway if the directory can't be discovered
	// or runs on a different server.
	ServerURL string `yaml:"serverURL,omitempty"`

	// The provided URL of the directory for a direct connection. This is not the
	// exploration http endpoint but the directory server itself. This endpoint will
	// accept requests for reading the directory using action names from the directory
	// specification. See also the directory.IDirectory api for these method names.
	// This is empty if a directory is not available.
	DirectoryURL string `yaml:"directoryURL,omitempty"`

	// The CA public certificate that signed the server certificate.
	// Intended for clients to validate the connection with the server.
	CaCert *x509.Certificate `yaml:"-"` // default cert if loaded

	// the client certification for transport client modules if applicable
	// Intended for clients that authenticate using a certificate.
	ClientCert *tls.Certificate `yaml:"-"`

	// The grpc URL used for the grpc server instantiation - part of grpc config
	// eg: "unix:///path/to/sock"  (yes triple slash)
	// GrpcURL string `yaml:"grpcURL"`

	// Override the https port used for the http server instantiation
	HttpsPort int `yaml:"httpsPort"`

	// the server certification for transport server modules if applicable
	// Intended for gateways or hub that runs a server.
	// Also usable for devices that run a server.
	TLSCert *tls.Certificate `yaml:"-"`

	// RpcTimeout is the communication timeout for use by transport client and server modules
	RpcTimeout time.Duration

	//--- ID and credentials for running as a client or using reverse connections ---

	// AppID is the application instance ID derived from the binary
	// Used as the default clientID
	AppID string `yaml:"appID"`

	// AuthToken contains the client authentication token for connecting to the server.
	// This can be set manually or loaded with GetAuthToken()
	AuthToken string `yaml:"-"`

	// The clientID used to authenticate, certificate filename and token name.
	// By default the clientID is the same as the appID unless changed.
	ClientID string `yaml:"clientID"`

	// The directory TD for bootstrapping a client.
	// This can be provided by discovery or set manually.
	DirTDD *td.TD `yaml:"-"`

	// KeyFile is the file that holds the private/public keys of the application.
	// Can be used by client applications to authenticate connect to a hub/gateway.
	// Intended for encryption and for client cert authentication when using reverse connections.
	// This is derived from the AppID: {certsDir}/{AppID}.key
	KeyFile string `yaml:"keyFile"` // app's key pair file location

}

// GetAuthToken returns the application authentication token.
// If no auth token is set then this is loaded from {clientID}.token.
// This returns an error if the token isn't set and the file cant be loaded.
func (env *AppEnvironment) GetAuthToken() (string, error) {
	if env.AuthToken != "" {
		return env.AuthToken, nil
	}
	tokenFile := path.Join(env.CertsDir, env.ClientID+".token")
	token, err := os.ReadFile(tokenFile)
	if err != nil {
		return "", err
	}
	env.AuthToken = string(token)
	return env.AuthToken, nil
}

// GetCACert returns the CA public certificate used with the servers using the
// default CA cert name.
// If not yet set this will load the certificate on first use.
//
// This returns nil if the CA is not set and cannot be loaded.
func (env *AppEnvironment) GetCACert() (caCert *x509.Certificate, err error) {
	if env.CaCert != nil {
		return env.CaCert, nil
	}
	caCertPath := filepath.Join(env.CertsDir, DefaultCaCertFile)
	env.CaCert, err = utils.LoadX509CertFromPEM(caCertPath)
	return env.CaCert, err
}

// Return the configured clientID
// This defaults to the appID, unless a different ID was provided via the commandline
func (env *AppEnvironment) GetClientID() string {
	return env.ClientID
}

// Get the server URL when needed - intended for starting clients when using the factory.
// This returns the preconfigured or commandline provided URL.
//
// This URL can also be set using the discovery client module configured for a specific protocol.
func (env *AppEnvironment) GetServerURL() string {
	return env.ServerURL
}

// Return the directory where a module stores its data.
// This does not create the directory.
func (env *AppEnvironment) GetStorageDir(moduleType string) string {
	storeDir := filepath.Join(env.StoresDir, moduleType)
	return storeDir
}

// GetTLSCert return the application TLS cert.
// If no cert is set yet, an attempt is made to load it from file.
// For servers this is the server app certificate.
// For clients this is the client certificate for authentication using certificates.
// This loads the {certsDir}/{clientID}Cert.Pem and {clientID}Key.pem files
//
// This returns the cert or an error if none is found.
func (env *AppEnvironment) GetTLSCert() (cert *tls.Certificate, err error) {
	if env.TLSCert != nil {
		return env.TLSCert, nil
	}
	certPath := filepath.Join(env.CertsDir, env.ClientID+DefaultCertFileSuffix)
	keyPath := filepath.Join(env.CertsDir, env.ClientID+DefaultKeyFileSuffix)
	env.TLSCert, err = utils.LoadTLSCertFromPEM(certPath, keyPath)
	return env.TLSCert, err
}

// LoadConfig loads the application/plugin configuration from {configDir}/{clientID}.yaml
//
// This returns an error if loading or parsing the config file fails.
// Returns nil if the config file doesn't exist or is loaded successfully.
func (env *AppEnvironment) LoadConfig(cfg interface{}) error {
	configFile := env.ConfigFile
	if !path.IsAbs(configFile) {
		configFile = path.Join(env.CertsDir, configFile)
	}
	if _, err := os.Stat(configFile); err != nil {
		slog.Info("Configuration file not found. Ignored.", "configFile", configFile)
		return nil
	}

	cfgData, err := os.ReadFile(configFile)
	if err != nil {
		err = fmt.Errorf("loading config failed: %w", err)
		return err
	} else {
		slog.Info("Loaded configuration file", "configFile", configFile)
		err = yaml.Unmarshal(cfgData, cfg)
	}
	return err
}

// NewAppEnvironment returns an application environment including folders for use by modules.
//
// Optionally parse commandline flags:
//
//	-home  	      alternative home directory. Default is the parent folder of the app binary
//	-clientID     alternative clientID. Default is the application binary name (appID).
//	-config       alternative config directory. Default is home/certs
//	-loglevel     debug, info, warning (default), error
//	-serverURL    optional device or gateway server URL or "" for auto-detect
//	-directoryURL optional directory URL or "" for auto-detect
//
// The default 'user based' structure is:
//
//		home
//		  |- bin                Core binaries
//	      |- plugins            Plugin binaries
//		  |- config             Service configuration yaml files
//		  |- certs              CA and service certificates
//		  |- logs               Logging output
//		  |- run                PID files and sockets
//		  |- stores
//		      |- {service}      Store for service
//
// The system based folder structure is used when launched from a path starting
// with /usr or /opt:
//
//	/opt/hiveot/bin            Application binaries, cli and launcher
//	/opt/hiveot/plugins        Plugin binaries
//	/etc/hiveot/conf.d         Service configuration yaml files
//	/etc/hiveot/certs          CA and service certificates
//	/var/log/hiveot            Logging output
//	/run/hiveot                PID files and sockets
//	/var/lib/hiveot/{service}  Storage of service
//
// This uses os.Args[0] application path to determine the home directory, which is the
// parent of the application binary.
// The default appID/clientID is based on the binary name using os.Args[0].
// This is used to load client certificate and/or token, if available in the certs directory.
//
//	homeDir to override the auto-detected or commandline paths. Use "" for defaults.
//	withFlags parse the commandline flags for -home and -clientID
func NewAppEnvironment(homeDir string, withFlags bool) *AppEnvironment {
	var binDir string
	var certsDir string
	var clientID string
	var configDir string
	var configFile string
	var logLevel string
	var logsDir string
	var pluginsDir string
	var storesDir string
	var directoryURL string
	var serverURL string

	// The default appID is the binary name. This allows for multiple instances
	// by linking instance IDs to the binary.
	appID := path.Base(os.Args[0])
	clientID = appID
	logLevel = os.Getenv("LOGLEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	// TODO: get default config from environment
	os.Environ()

	// default home folder is the parent of the core or plugin binary
	if homeDir == "" {
		binDir = filepath.Dir(os.Args[0])
		if !path.IsAbs(binDir) {
			cwd, _ := os.Getwd()
			binDir = path.Join(cwd, binDir)
		}
		homeDir = filepath.Join(binDir, "..")
	}

	if withFlags {
		// handle commandline options
		flag.StringVar(&homeDir, "home", homeDir, "Application home directory")
		flag.StringVar(&certsDir, "certs", certsDir, "Certificate and keys directory")
		flag.StringVar(&configDir, "config", configDir, "Configuration directory")
		flag.StringVar(&configFile, "configFile", configFile, "Configuration file")
		flag.StringVar(&pluginsDir, "plugins", pluginsDir, "Plugins directory")
		flag.StringVar(&clientID, "clientID", clientID, "clientID to authenticate with")
		flag.StringVar(&logLevel, "logLevel", logLevel, "logging level: debug, warning, info, error")
		flag.StringVar(&directoryURL, "directoryURL", directoryURL, "url of directory TD")
		flag.StringVar(&serverURL, "serverURL", serverURL, "connection url for server")
		if flag.Usage == nil {
			flag.Usage = func() {
				fmt.Println("Usage: " + appID + " [options] ")
				fmt.Println()
				fmt.Println("Options:")
				flag.PrintDefaults()
			}
		}
		flag.Parse()
	}
	if strings.HasPrefix(homeDir, "~") {
		usr, _ := user.Current()
		homeDir = path.Join(usr.HomeDir, homeDir[1:])
	} else if !path.IsAbs(homeDir) {
		cwd, _ := os.Getwd()
		homeDir = path.Join(cwd, homeDir)
	}

	// Try to be smart about whether to use the system structure.
	// If the path starts with /opt or /usr then use
	// the system folder configuration. This might be changed in future if it turns
	// out not to be so smart at all.
	// Future: make this work on windows
	useSystem := strings.HasPrefix(homeDir, "/opt")

	if useSystem {
		homeDir = filepath.Join("/var", "lib", "hiveot")
		if binDir == "" {
			binDir = filepath.Join("/opt", "hiveot")
		}
		if pluginsDir == "" {
			pluginsDir = filepath.Join(binDir, "plugins")
		}
		if configDir == "" {
			configDir = filepath.Join("/etc", "hiveot", "conf.d")
		}
		if certsDir == "" {
			certsDir = filepath.Join("/etc", "hiveot", "certs")
		}
		if logsDir == "" {
			logsDir = filepath.Join("/var", "log", "hiveot")
		}
		if storesDir == "" {
			storesDir = filepath.Join("/var", "lib", "hiveot")
		}
	} else { // use application user dir under ~/bin/hiveot
		if binDir == "" {
			binDir = filepath.Join(homeDir, "bin")
		}
		if pluginsDir == "" {
			pluginsDir = filepath.Join(homeDir, "plugins")
		}
		if certsDir == "" {
			certsDir = filepath.Join(homeDir, "certs")
		}
		if logsDir == "" {
			logsDir = filepath.Join(homeDir, "logs")
		}
		if storesDir == "" {
			storesDir = filepath.Join(homeDir, "stores")
		}
		if configDir == "" {
			configDir = filepath.Join(homeDir, "config")
		}
	}
	if configFile == "" {
		configFile = path.Join(configDir, clientID+".yaml")
	}
	// load the CA cert if found
	caCertFile := path.Join(certsDir, DefaultCaCertFile)
	caCert, _ := utils.LoadX509CertFromPEM(caCertFile)

	// determine the expected location of the service auth key and token
	keyFile := path.Join(certsDir, clientID+".key")

	slog.Info("NewAppEnvironment",
		slog.String("appID", appID),
		slog.String("clientID", clientID),
		slog.String("home", homeDir),
		slog.String("certsDir", certsDir),
		slog.String("configDir", configDir),
		slog.String("pluginsDir", pluginsDir),
		slog.String("serverURL", serverURL),
	)

	utils.SetLogging(logLevel, "")

	return &AppEnvironment{
		BinDir:       binDir,
		CaCert:       caCert,
		AppID:        appID,
		ClientID:     clientID,
		ConfigDir:    configDir,
		ConfigFile:   configFile,
		CertsDir:     certsDir,
		DirectoryURL: directoryURL,
		// GrpcURL:    grpctransport.DefaultGrpcURL,
		// HttpsPort:  transport.DefaultHttpsPort,
		HomeDir:    homeDir,
		KeyFile:    keyFile,
		LogsDir:    logsDir,
		LogLevel:   logLevel,
		PluginsDir: pluginsDir,
		ServerURL:  serverURL,
		StoresDir:  storesDir,
	}
}
