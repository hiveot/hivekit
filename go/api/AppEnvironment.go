package api

import (
	"crypto"
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

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/wot"
	"github.com/hiveot/hivekit/go/utils"
	"gopkg.in/yaml.v3"
)

// DefaultAdminUserID is the default administrator client ID
// Used by authn and certs services to create an admin account on startup.
const DefaultAdminUserID = "admin"

// The default timeout for rpc calls
const DefaultRPCTimeout = time.Second * 3

// certificate file names
const (
	// CA key for creating self signed certificates
	DefaultCaKeyFile = "caKey.pem"

	// clients and server need the CA
	DefaultCaCertFile = "caCert.pem"

	// certificate name suffix {name}Cert.pem
	DefaultCertFileSuffix = "Cert.pem"

	// DefaultKeyFileSuffix defines the filename suffix under which public/private keys are
	// stored in the certs directory.
	DefaultKeyFileSuffix = "Key.pem"

	// default server name to use if none is provided
	DefaultServerName = "server"

	DefaultTokenFileSuffix = ".token"
)

// AppEnvironment holds the running environment naming conventions.
// Intended for devices, services, or client applications.
// This contains folder locations, CA certificate and application clientID
type AppEnvironment struct {
	// AppID is the application type ID derived from the binary name.
	// This is not the instance ID.
	// This can be used to load matching configuration yaml files, logfiles, and
	// other application related configuration.
	// This is also the default clientID.
	AppID string `yaml:"appID"`

	// Directories
	BinDir     string `yaml:"binDir,omitempty"`     // Application binary folder, e.g. launcher, cli, ...
	HomeDir    string `yaml:"homeDir,omitempty"`    // Home folder, default this is the parent of bin, config, certs and logs
	ConfigDir  string `yaml:"configDir,omitempty"`  // config folder with application and configuration files
	CertsDir   string `yaml:"certsDir,omitempty"`   // Certificates and keys location
	LogsDir    string `yaml:"logsDir,omitempty"`    // Logging output
	LogLevel   string `yaml:"logLevel,omitempty"`   // logging level: error, warning, info, debug
	PluginsDir string `yaml:"pluginsDir,omitempty"` // Plugins
	StoresDir  string `yaml:"storesDir,omitempty"`  // Root of the service stores

	// Override the https port used for the http server instantiation
	HttpsPort int `yaml:"httpsPort"`

	// RpcTimeout is the communication timeout for use by transport client and server cells
	RpcTimeout time.Duration

	//--- Client specific settings ---

	// The clientID used in authentication.
	// Intended for authenticating with servers along with auth token or client cert.
	ClientID string `yaml:"clientID,omitempty"`

	// Optional directory TD for connecting a directory client to a directory server.
	//
	// A discovery client sets this when DirTDURL is determined.
	dirTD *wot.TD `yaml:"-"`

	// Optional directory TD URL for downloading a specific directory TD.
	//
	// A discovery client can set this to a locally discovered directory, or
	// it can be provided manually to select a specific local or remote directory.
	DirTDURL string `yaml:"dirTDURL,omitempty"`

	// Manual override of the server to connect to. Intended for client factory that
	// otherwise uses a directory. Intended mainly for testing and offering the
	// consumer the ability to force a connection without TD/forms.
	ServerURL string `yaml:"serverURL,omitempty"`

	//--- loaded or generated settings. See the Get... methods.

	// AuthToken contains the client authentication token for connecting to the server.
	// This can be set manually or loaded with GetAuthToken()
	// See also GetAuthToken which will attempt to load it on first use.
	authToken string `yaml:"-"`

	// The cached self-signed CA and key used to sign the client certificate.
	// Intended for clients to validate the connection with the server.
	// If loaded this will be included in the RootCAs.
	caCert *x509.Certificate `yaml:"-"`
	caKey  crypto.Signer     `yaml:"-"`

	// The cached self-signed client certificate
	// If available it is loaded from {clientID}Cert|{clientID}Key.pem
	// Intended for clients that authenticate using a certificate.
	clientCert *tls.Certificate `yaml:"-"`

	// Optional root CA pool. This defaults to nil.
	// If CaCert is loaded then this is set to the system CA's + the CaCert.
	rootCAs *x509.CertPool `yaml:"-"`

	// The server certificate loaded from ServerCert|ServerKey.pem
	// Intended for devices, gateway or hub that runs a server.
	// Use RunInitFactoryCerts in the factory chain to ensure a self signed server
	// cert is created if needed, or place a cert manually.
	serverCert *tls.Certificate `yaml:"-"`
}

// Create all missing directories
// this returns an error if one of them doesnt exist and cant be created
func (env *AppEnvironment) CreateAllDirs() (err error) {
	if err2 := env.CreateDir(env.HomeDir, 0750); err2 != nil {
		err = err2
	}
	if err2 := env.CreateDir(env.BinDir, 0750); err2 != nil {
		err = err2
	}
	if err2 := env.CreateDir(env.CertsDir, 0750); err2 != nil {
		err = err2
	}
	if err2 := env.CreateDir(env.ConfigDir, 0750); err2 != nil {
		err = err2
	}
	if err2 := env.CreateDir(env.LogsDir, 0750); err2 != nil {
		err = err2
	}
	// if err2 := env.CreateDir(env.PluginsDir, 0750); err2 != nil {
	// err = err2
	// }
	if err2 := env.CreateDir(env.PluginsDir, 0750); err2 != nil {
		err = err2
	}
	if err2 := env.CreateDir(env.StoresDir, 0700); err2 != nil {
		err = err2
	}
	return err
}

// Create a missing directory
func (env *AppEnvironment) CreateDir(path string, mode os.FileMode) error {
	info, err := os.Stat(path)
	if err != nil {
		err = os.MkdirAll(path, mode)
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("CreateDir, '%s' already exists but is not a directory", path)
	}
	// path already exists and is a directory
	return nil
}

// GetAuthToken returns the application authentication token.
//
// By default this is compatible with the authn session manager Save/LoadToken
// so that tokens created with the session manager can be loaded, if sufficient
// permissions exist.
//
// If no auth token is set then this is loaded from {clientID}.token.
// This returns an error if the token isn't set and the file cant be loaded.
func (env *AppEnvironment) GetAuthToken() (string, error) {
	if env.authToken != "" {
		return env.authToken, nil
	}
	tokenFile := path.Join(env.CertsDir, env.ClientID+DefaultTokenFileSuffix)
	token, err := os.ReadFile(tokenFile)
	if err != nil {
		return "", err
	}
	env.authToken = string(token)
	return env.authToken, nil
}

// GetCACert returns a self signed CA certificate.
//
// If a CA is cached then it is returned along with the cached CA key if set,
// otherwise attempt to load it from {certsDir}/caCert.pem
//
// This returns nil if the CA is not set and cannot be loaded.
func (env *AppEnvironment) GetCACert() (
	caCert *x509.Certificate, caKey crypto.Signer, err error) {

	if env.caCert != nil {
		return env.caCert, env.caKey, nil
	}
	certPath := filepath.Join(env.CertsDir, DefaultCaCertFile)
	keyPath := filepath.Join(env.CertsDir, DefaultCaKeyFile)
	env.caCert, err = utils.LoadCACert(certPath)
	env.caKey, _ = utils.LoadPrivateKey(keyPath)
	return env.caCert, env.caKey, err
}

// GetClientCert return the client authentication certificate.
//
// This loads the file {certsDir}/{ClientID}Cert.pem and {ClientID}Key.pem.
//
//	clientID is the client whose cert to load. Use "" for env.ClientID.
//
// This returns the TLS certificate or an error if none is found.
func (env *AppEnvironment) GetClientCert(clientID string) (cert *tls.Certificate, err error) {
	if clientID == "" {
		clientID = env.ClientID
	}
	if env.clientCert != nil && clientID == env.ClientID {
		return env.clientCert, nil
	}
	certPath := filepath.Join(env.CertsDir, clientID+DefaultCertFileSuffix)
	keyPath := filepath.Join(env.CertsDir, clientID+DefaultKeyFileSuffix)
	clientCert, err := utils.LoadTLSCert(certPath, keyPath)
	return clientCert, err
}

// Return the discovered directory TDD
// This is only available if the TDD URL is discovered.
func (env *AppEnvironment) GetDirTD() *wot.TD {
	return env.dirTD
}

// Return the root CAs collection from the system cert pool
// This includes the self signed CA cert if available.
func (env *AppEnvironment) GetRootCAs() (rootCAs *x509.CertPool) {
	if env.rootCAs != nil {
		return env.rootCAs
	}
	env.rootCAs, _ = x509.SystemCertPool()
	caCert, _, _ := env.GetCACert()
	if caCert != nil {
		env.rootCAs.AddCert(caCert)
	}
	return env.rootCAs
}

// GetServerCert return the application server TLS certificate.
//
// serverName is the optional name of the server. When using "" the default
// server name "server" is used and the value is cached.
//
// On first use this attempts to load the certificate using the names
// {serverName}Cert.pem and "ServerKey.pem".
//
// This returns the cert or an error if none is found.
func (env *AppEnvironment) GetServerCert(serverName string) (cert *tls.Certificate, err error) {
	if serverName == "" {
		serverName = DefaultServerName
		if env.serverCert != nil {
			return env.serverCert, nil
		}
		certPath := filepath.Join(env.CertsDir, DefaultServerName+DefaultCertFileSuffix)
		keyPath := filepath.Join(env.CertsDir, DefaultServerName+DefaultKeyFileSuffix)
		cert, err = utils.LoadTLSCert(certPath, keyPath)
		// store it for next time
		env.serverCert = cert

	} else {
		certPath := filepath.Join(env.CertsDir, serverName+DefaultCertFileSuffix)
		keyPath := filepath.Join(env.CertsDir, serverName+DefaultKeyFileSuffix)
		cert, err = utils.LoadTLSCert(certPath, keyPath)
	}
	return cert, err
}

// Return the directory where a cell stores its data.
// This does not create the directory.
func (env *AppEnvironment) GetStorageDir(cellType string) string {
	storeDir := filepath.Join(env.StoresDir, cellType)
	return storeDir
}

// LoadConfig loads the application/plugin configuration from {configDir}/{name}.yaml
//
// This returns an error if loading or parsing the config file fails.
// Returns nil if the config file doesn't exist or is loaded successfully.
func (env *AppEnvironment) LoadConfig(name string, cfg interface{}) error {
	configFile := name
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

// Print the environment to the console
//
// Use verbose to display additional details.
// 1: essentials: AppID, ClientID, home, auth token, has certs
// 2: Directories, timeout, URLs, Cert details
func (env *AppEnvironment) Print(verbose int) {
	if verbose >= 1 {
		fmt.Println("App Environment:")
		fmt.Println("* AppID:          ", env.AppID)
		fmt.Println("* ClientID:       ", env.ClientID)
		fmt.Println("* Home directory: ", env.HomeDir)

		if verbose >= 2 {
			fmt.Println("")
			fmt.Println("* Bin directory:    ", env.BinDir)
			fmt.Println("* Certs directory:  ", env.CertsDir)
			fmt.Println("* Config directory: ", env.ConfigDir)
			fmt.Println("* Logs directory:   ", env.LogsDir)
			fmt.Println("* Plugins directory:", env.PluginsDir)
			fmt.Println("* Stores directory: ", env.StoresDir)

			fmt.Println("")
			fmt.Println("* Directory TD URL: ", env.DirTDURL)
			fmt.Println("* Rpc timeout:      ", env.RpcTimeout)
		}

		_, err := env.GetAuthToken()
		if err != nil {
			fmt.Println("! No auth token found for client: ", env.ClientID)
		} else {
			fmt.Println("* Found auth token for login as", env.ClientID)
		}

		caCert, _, err := env.GetCACert()
		if err != nil {
			fmt.Println("! CA cert cannot be loaded", err.Error())
		} else {
			fmt.Println("* CA cert was successfully loaded")
			if verbose >= 2 {
				env.ShowCertDetails(caCert)
			}
		}
		serverTLS, err := env.GetServerCert("")
		if err != nil {
			fmt.Println("* No server cert set")
		} else {
			fmt.Println("* Server cert successfully loaded")
			serverCert, err := x509.ParseCertificate(serverTLS.Certificate[0])
			if err == nil && verbose >= 2 {
				env.ShowCertDetails(serverCert)
			}
		}

		tlsCert, err := env.GetClientCert(env.ClientID)
		if err != nil {
			fmt.Printf("* No client cert for '%s'\n", env.ClientID)
		} else {
			chain, _ := utils.TLSCertToX509(tlsCert)
			clientCert := chain[0]
			fmt.Printf("* Client Cert for '%s' successfully loaded\n", env.ClientID)
			certCN := clientCert.Subject.CommonName
			if certCN != env.ClientID {
				fmt.Printf("! Client cert CN '%s' doesn't match ClientID\n", certCN)
			}
			if verbose >= 2 {
				env.ShowCertDetails(clientCert)
			}
		}
	}
	fmt.Println("")
}

// ShowCertDetails shows details for the given certs and identifies expired certs
func (env *AppEnvironment) ShowCertDetails(cert *x509.Certificate) {
	validUntilStr := cert.NotAfter.Local().Format(time.RFC1123Z)
	if time.Now().After(cert.NotAfter) {
		validUntilStr = fmt.Sprintf("%s%s (Expired!)%s", utils.CORed, validUntilStr, utils.COReset)
	} else {
		validUntilStr = utils.COGreen + validUntilStr + utils.COReset
	}

	fmt.Println("  IsCA        : ", cert.IsCA)
	fmt.Println("  Version     : ", cert.Version)
	fmt.Println("  Created     : ", cert.NotBefore.Local().Format(time.RFC1123Z))
	fmt.Println("  Valid until : ", validUntilStr)
	fmt.Println("  Subject     : ", cert.Subject.String())
	fmt.Println("  Issuer      : ", cert.Issuer.String())
	fmt.Println("  DNS names   : ", cert.DNSNames)
	fmt.Println("  IP addresses: ", cert.IPAddresses)
}

// SetAuthToken Sets the client authentication token.
//
// Use to manually set a token for this connection
func (env *AppEnvironment) SetAuthToken(newToken string) {
	env.authToken = newToken
}

// Set and save an updated CA certificate and add it to the cert pool.
// This updates the environment root CA pool.
//
//	caCert is the CA certificate to use instead of loading from file.
//	caKey is needed if a self-signed client or server certificate is generated
//		or validated using this CA.
func (env *AppEnvironment) SetCACert(caCert *x509.Certificate, caKey crypto.Signer) error {
	env.caKey = caKey
	env.caCert = caCert
	env.rootCAs, _ = x509.SystemCertPool()
	env.rootCAs.AddCert(caCert)

	certsDir := env.CertsDir
	caKeyPath := path.Join(certsDir, DefaultCaKeyFile)
	caCertPath := path.Join(certsDir, DefaultCaCertFile)
	err := utils.SaveX509Cert(caCert, caCertPath)
	if err == nil {
		err = utils.SavePrivateKey(caKey, caKeyPath)
	}
	return err
}

// Set or replace the client certificate used by the environment
//
// This must pass validation against the rootCA pool, which includes the loaded CA.
// Also, the cert common name must match the clientID. If the clientID is empty then
// it will be updated with the cert common name.
// On success, GetClientCert will return this certificate instead of loading it from file.
func (env *AppEnvironment) SetClientCert(cert *tls.Certificate) error {

	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		err = fmt.Errorf("SetClientCert: invalid certificate. Ignored: %w", err)
		slog.Error(err.Error())
		return err
	}
	opts := x509.VerifyOptions{
		Roots:     env.rootCAs,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	_, err = x509Cert.Verify(opts)
	if err != nil {
		err = fmt.Errorf("SetClientCert: Certificate failed verification. Ignored: %w", err)
		slog.Error(err.Error())
		return err
	}
	// adopt the client cert CN as the clientID
	if env.ClientID == "" {
		env.ClientID = x509Cert.Subject.CommonName
	} else if x509Cert.Subject.CommonName != env.ClientID {
		err = fmt.Errorf("SetClientCert: Certificate doesn't match clientID. Ignored.")
		slog.Error(err.Error(),
			"clientID", env.ClientID,
			"cert common name", x509Cert.Subject.CommonName)
		return err
	}
	env.clientCert = cert
	return nil
}

// Set the directory TD to use with clients
func (env *AppEnvironment) SetDirTD(tdoc *wot.TD) {
	env.dirTD = tdoc
}

// Set and save the cached server certificate used by the environment.
//
// This certificate must successfully verify against the CA cert pool.
func (env *AppEnvironment) SetServerCert(cert *tls.Certificate) error {

	certsDir := env.CertsDir
	serverName := DefaultServerName
	keyPath := path.Join(certsDir, serverName+DefaultKeyFileSuffix)
	certPath := path.Join(certsDir, serverName+DefaultCertFileSuffix)

	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		err = fmt.Errorf("SetServerCert: invalid certificate: %s", err.Error())
		slog.Error(err.Error())
		return err
	}
	opts := x509.VerifyOptions{
		Roots:     env.GetRootCAs(),
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	_, err = x509Cert.Verify(opts)
	if err != nil {
		err = fmt.Errorf("SetServerCert: Certificate failed verification: %s", err.Error())
		slog.Error(err.Error())
		return err
	}
	env.serverCert = cert
	err = utils.SaveTLSCert(cert, certPath, keyPath)
	return err
}

// NewAppEnvironment returns an application environment including folders for use by hive cells.
// This creates the neccesary folder structure.
//
// Optionally parse commandline flags:
//
//	-home  	      alternative home directory. Default is the parent folder of the app binary
//	-clientID     Client ID to use. Needed for loading token and client cert.
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

	showEnv := 0
	env := &AppEnvironment{
		LogLevel: "warn",
		// PluginsDir:   pluginsDir,
		RpcTimeout: DefaultRPCTimeout,
	}

	// The default appID is the binary name. This allows for multiple instances
	// by linking instance IDs to the binary.
	env.AppID = path.Base(os.Args[0])
	logLevel := os.Getenv("LOGLEVEL")
	if logLevel != "" {
		env.LogLevel = logLevel
	}

	// TODO: get default config from environment
	os.Environ()

	// default home folder is the parent of the core or plugin binary
	if env.HomeDir == "" {
		binDir := filepath.Dir(os.Args[0])
		if !path.IsAbs(binDir) {
			cwd, _ := os.Getwd()
			binDir = path.Join(cwd, binDir)
		}
		env.HomeDir = filepath.Join(binDir, "..")
	}

	// commandline options to set env manually
	if withFlags {
		// handle commandline options
		flag.StringVar(&homeDir, "home", homeDir, "Application home directory")
		flag.StringVar(&env.CertsDir, "certs", env.CertsDir, "Certificate and keys directory")
		flag.StringVar(&env.ConfigDir, "config", env.ConfigDir, "Configuration directory")
		flag.StringVar(&env.ClientID, "clientID", env.ClientID, "clientID to authenticate with")
		flag.StringVar(&env.LogLevel, "loglevel", env.LogLevel, "logging level: debug, warning, info, error")
		flag.StringVar(&env.DirTDURL, "dirTD", env.DirTDURL, "url where to download the directory TD")
		flag.IntVar(&showEnv, "env", showEnv, "print environment info: 1 basic, 2 verbose")
		flag.StringVar(&env.ServerURL, "server", env.ServerURL, "Force client connection to this server URL.")
		flag.DurationVar(&env.RpcTimeout, "timeout", env.RpcTimeout, "Communication timeout.")
		// if flag.Usage == nil {
		// 	flag.Usage = func() {
		// 		fmt.Println("Usage: app [options] ")
		// 		fmt.Println()
		// 		fmt.Println("Options:")
		// 		flag.PrintDefaults()
		// 	}
		// }
		// note that no args is not an error and should not show help
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
		if env.BinDir == "" {
			env.BinDir = filepath.Join("/opt", "hiveot")
		}
		if env.PluginsDir == "" {
			env.PluginsDir = filepath.Join(env.BinDir, "plugins")
		}
		if env.ConfigDir == "" {
			env.ConfigDir = filepath.Join("/etc", "hiveot", "conf.d")
		}
		if env.CertsDir == "" {
			env.CertsDir = filepath.Join("/etc", "hiveot", "certs")
		}
		if env.LogsDir == "" {
			env.LogsDir = filepath.Join("/var", "log", "hiveot")
		}
		if env.StoresDir == "" {
			env.StoresDir = filepath.Join("/var", "lib", "hiveot")
		}
	} else { // use application user dir under ~/bin/hiveot
		if env.BinDir == "" {
			env.BinDir = filepath.Join(homeDir, "bin")
		}
		if env.PluginsDir == "" {
			env.PluginsDir = filepath.Join(homeDir, "plugins")
		}
		if env.CertsDir == "" {
			env.CertsDir = filepath.Join(homeDir, "certs")
		}
		if env.LogsDir == "" {
			env.LogsDir = filepath.Join(homeDir, "logs")
		}
		if env.StoresDir == "" {
			env.StoresDir = filepath.Join(homeDir, "stores")
		}
		if env.ConfigDir == "" {
			env.ConfigDir = filepath.Join(homeDir, "config")
		}
	}
	env.HomeDir = homeDir

	// utils.SetLogging(logLevel, "")
	if showEnv != 0 {
		env.Print(showEnv)
	}

	// AppEnvironment controls the use of cell timeout
	msg.CellTimeout = env.RpcTimeout

	return env
}
