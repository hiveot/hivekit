package factoryapi

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

	certsapi "github.com/hiveot/hivekit/go/modules/certs/api"
	"github.com/hiveot/hivekit/go/modules/certs/certutils"
	"gopkg.in/yaml.v3"
)

// DirectoryURL_Arg is the optional commandline argument name with the URL of the directory TD
const DirectoryURL_Arg = "directoryURL"

// ServerURL_Arg is the optional commandline argument name with the connection URL of the digitwin server
const ServerURL_Arg = "serverURL"

// AppEnvironment holds the running environment naming conventions.
// Intended for devices, services, or client applications.
// This contains folder locations, CA certificate and application clientID
type AppEnvironment struct {
	// Directories
	BinDir       string `yaml:"binDir,omitempty"`       // Application binary folder, e.g. launcher, cli, ...
	PluginsDir   string `yaml:"pluginsDir,omitempty"`   // Plugin folder
	HomeDir      string `yaml:"homeDir,omitempty"`      // Home folder, default this is the parent of bin, config, certs and logs
	ConfigDir    string `yaml:"configDir,omitempty"`    // config folder with application and configuration files
	ConfigFile   string `yaml:"configFile,omitempty"`   // Application configuration file. Default is clientID.yaml
	CertsDir     string `yaml:"certsDir,omitempty"`     // Certificates and keys location
	LogsDir      string `yaml:"logsDir,omitempty"`      // Logging output
	LogLevel     string `yaml:"logLevel,omitempty"`     // logging level: error, warning, info, debug
	StoresDir    string `yaml:"storesDir,omitempty"`    // Root of the service stores
	ServerURL    string `yaml:"serverURL,omitempty"`    // forced server to connect to: scheme://host/path or "" for auto
	DirectoryURL string `yaml:"directoryURL,omitempty"` // Discovery URL of the directory

	// The CA public certificate that signed the server certificate.
	// Intended for clients to validate the connection with the server.
	CaCert *x509.Certificate `yaml:"-"` // default cert if loaded

	// the server certification for transport modules if applicable
	// Intended for gateways or hub that runs a server.
	// Also usable for devices that run a server.
	ServerCert *tls.Certificate `yaml:"-"`

	// RpcTimeout is the communication timeout for use by transport client and server modules
	RpcTimeout time.Duration

	//--- ID and credentials for running as a client or using reverse connections ---

	// AppID is the application instance ID derived from the binary
	// A device or service can use this as the clientID for reverse connections to the hub.
	AppID string `yaml:"appID"`

	// KeyFile is the file that holds the private/public keys of the application.
	// Can be used by client applications to authenticate connect to a hub/gateway.
	// Intended for encryption and for client cert authentication when using reverse connections.
	// This is derived from the AppID: {certsDir}/{AppID}.key
	KeyFile string `yaml:"keyFile"` // app's key pair file location

	// TokenFile holds the authentication token of the application.
	// Intended for device authentication when using reverse connections.
	// This is derived from the AppID: {certsDir}/{AppID}.token
	TokenFile string `yaml:"tokenFile"` // app's auth token file location
}

// Get the CA used with the servers.
// This will load the certificate on first use.
// This returns nil if the CA cannot be loaded.
func (env *AppEnvironment) GetCA() (caCert *x509.Certificate, err error) {
	if env.CaCert != nil {
		return env.CaCert, nil
	}
	caCertPath := filepath.Join(env.CertsDir, certsapi.DefaultCaCertFile)
	env.CaCert, err = certutils.LoadX509CertFromPEM(caCertPath)
	return env.CaCert, err
}

// Get the server TLS cert when needed.
// This will load the certificate on first use.
// This returns nil if the server certificate cannot be loaded.
func (env *AppEnvironment) GetServerCert() (cert *tls.Certificate, err error) {
	if env.ServerCert != nil {
		return env.ServerCert, nil
	}
	serverCertPath := filepath.Join(env.CertsDir, certsapi.DefaultServerCertFile)
	serverKeyPath := filepath.Join(env.CertsDir, certsapi.DefaultServerKeyFile)
	env.ServerCert, err = certutils.LoadTLSCertFromPEM(serverCertPath, serverKeyPath)
	return env.ServerCert, err
}

// Return the directory where a module stores its data.
// This does not create the directory.
func (env *AppEnvironment) GetStorageDir(moduleType string) string {
	storeDir := filepath.Join(env.StoresDir, moduleType)
	return storeDir
}

// LoadConfig loads the application configuration from {configDir}/{clientID}.yaml
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
//	-home  		alternative home directory. Default is the parent folder of the app binary
//	-clientID  	alternative clientID. Default is the application binary name.
//	-config     alternative config directory. Default is home/certs
//	-configFile alternative application config file. Default is {clientID}.yaml
//	-loglevel   debug, info, warning (default), error
//	-server     optional server URL or "" for auto-detect
//	-core       optional server core or "" for auto-detect
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
// The default clientID is based on the binary name using os.Args[0].
//
//	homeDir to override the auto-detected or commandline paths. Use "" for defaults.
//	withFlags parse the commandline flags for -home and -clientID
func NewAppEnvironment(homeDir string, withFlags bool) *AppEnvironment {
	var configFile string
	var configDir string
	var binDir string
	var pluginsDir string
	var certsDir string
	var logsDir string
	var storesDir string
	var directoryURL string
	var serverURL string

	// The default appID is the binary name. This allows for multiple instances
	// by linking instance IDs to the binary.
	appID := path.Base(os.Args[0])
	logLevel := os.Getenv("LOGLEVEL")
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
		flag.StringVar(&configDir, "config", configDir, "Configuration directory")
		flag.StringVar(&configFile, "configFile", configFile, "Configuration file")
		flag.StringVar(&appID, "clientID", appID, "Application clientID to authenticate with")
		flag.StringVar(&logLevel, "logLevel", logLevel, "logging level: debug, warning, info, error")
		flag.StringVar(&directoryURL, DirectoryURL_Arg, directoryURL, "url of directory TD")
		flag.StringVar(&serverURL, ServerURL_Arg, serverURL, "connection url for server")
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
		binDir = filepath.Join("/opt", "hiveot")
		pluginsDir = filepath.Join(binDir, "plugins")
		configDir = filepath.Join("/etc", "hiveot", "conf.d")
		certsDir = filepath.Join("/etc", "hiveot", "certs")
		logsDir = filepath.Join("/var", "log", "hiveot")
		storesDir = filepath.Join("/var", "lib", "hiveot")
	} else { // use application user dir under ~/bin/hiveot
		binDir = filepath.Join(homeDir, "bin")
		pluginsDir = filepath.Join(homeDir, "plugins")
		certsDir = filepath.Join(homeDir, "certs")
		logsDir = filepath.Join(homeDir, "logs")
		storesDir = filepath.Join(homeDir, "stores")

		if configDir == "" {
			configDir = filepath.Join(homeDir, "config")
		}
	}
	if configFile == "" {
		configFile = path.Join(configDir, appID+".yaml")
	}
	// load the CA cert if found
	caCertFile := path.Join(certsDir, certsapi.DefaultCaCertFile)
	caCert, _ := certutils.LoadX509CertFromPEM(caCertFile)

	// determine the expected location of the service auth key and token
	tokenFile := path.Join(certsDir, appID+".token")
	keyFile := path.Join(certsDir, appID+".key")

	return &AppEnvironment{
		BinDir:       binDir,
		CaCert:       caCert,
		AppID:        appID,
		ConfigDir:    configDir,
		ConfigFile:   configFile,
		CertsDir:     certsDir,
		DirectoryURL: directoryURL,
		HomeDir:      homeDir,
		KeyFile:      keyFile,
		LogsDir:      logsDir,
		LogLevel:     logLevel,
		PluginsDir:   pluginsDir,
		ServerURL:    serverURL,
		StoresDir:    storesDir,
		TokenFile:    tokenFile,
	}
}
