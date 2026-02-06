package module

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/httpserver"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/lmittmann/tint"
)

// HttpServerModule is a module providing a TLS HTTPS server.
// Intended for use by HTTP based application protocols.
// This implements IHttpServer and IHiveModule interfaces.
//
// Note that this does not implement the ITransportModule interface as this module provides the
// http server for use by transport modules.
type HttpServerModule struct {
	modules.HiveModuleBase

	// HTTP authentication handler.
	authenticateHandler func(req *http.Request) (clientID string, role string, err error)

	config     *httpserver.HttpServerConfig
	connectURL string

	// the actual golang HTTP/TLS server
	httpServer *http.Server

	rootRouter *chi.Mux
	pubRoute   chi.Router
	protRoute  chi.Router

	logger *log.Logger

	// certificate handler for running the server
	caCert     *x509.Certificate
	serverCert *tls.Certificate

	// The router available for this TLS server
	// Intended for Http modules to add their routes
	router *chi.Mux

	// the RRN messaging API
	// msgAPI *api.HttpMsgHandler
}

// The default token authentication handler extracts the bearer token from the authorization header
// and passes it to the configured token validator.
func (m *HttpServerModule) DefaultAuthenticate(req *http.Request) (
	clientID string, clientRole string, err error) {

	if m.config.ValidateToken == nil {
		err := fmt.Errorf("DefaultAuthenticate: Missing ValidateToken handler in configuration")
		return "", "", err
	}
	bearerToken, err := utils.GetBearerToken(req)
	if err != nil {
		return "", "", err
	}
	//check if the token is properly signed and still valid
	clientID, role, validUntil, err := m.config.ValidateToken(bearerToken)
	if err != nil {
		return "", "", err
	}
	_ = validUntil
	return clientID, role, err
}

// Provide the HTTP base URL to connect to the server. Eg "https://addr:port/""
func (m *HttpServerModule) GetConnectURL() string {
	return m.connectURL
}

// Set the handler that validates tokens.
// This will enable the protected routes.
func (m *HttpServerModule) SetAuthValidator(validator transports.IAuthValidator) {
	m.config.ValidateToken = validator.ValidateToken
}

// Start readies the module for use.
// This starts a http server instance and sets-up a public and protected route.
//
// Starts a HTTPS TLS service
func (m *HttpServerModule) Start() (err error) {
	var tlsConf *tls.Config
	cfg := m.config
	m.connectURL = fmt.Sprintf("https://%s:%d", cfg.Address, cfg.Port)

	slog.Info("Starting HTTP server module", "address", cfg.Address, "port", cfg.Port)
	if cfg.CaCert == nil || cfg.ServerCert == nil {
		//no TLS possible
		if cfg.NoTLS == false {
			err := fmt.Errorf("Start aborted. Missing CA or server certificate.")
			slog.Error(err.Error())

			return err
		}
	} else {
		// setup TLS
		caCertPool := x509.NewCertPool()
		caCertPool.AddCert(cfg.CaCert)
		tlsConf = &tls.Config{
			Certificates:       []tls.Certificate{*cfg.ServerCert},
			ClientAuth:         tls.VerifyClientCertIfGiven,
			ClientCAs:          caCertPool,
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: false,
		}
	}

	logHandler := tint.NewHandler(os.Stdout, &tint.Options{
		AddSource: true, Level: slog.LevelInfo, TimeFormat: "Jan _2 15:04:05.0000"})
	m.logger = slog.NewLogLogger(logHandler, slog.LevelDebug)

	m.rootRouter = chi.NewRouter()
	m.addMiddleware(cfg)

	// setup listener
	m.httpServer = &http.Server{
		Addr: fmt.Sprintf("%s:%d", cfg.Address, cfg.Port),
		// ReadTimeout:  5 * time.Minute, // 5 min to allow for delays when 'curl' on OSx prompts for username/password
		// WriteTimeout: 10 * time.Second,
		Handler:   m.rootRouter,
		TLSConfig: tlsConf,
		//ErrorLog:  log.Default(),
	}
	lisn, err := net.Listen("tcp", m.httpServer.Addr)
	if err != nil {
		slog.Info("Start - Listen return an error", "err", err.Error())
		return err
	}

	// finally run the server in the background
	go func() {
		// serverTLSConf contains certificate and key
		slog.Info("TLSServer - Listening")
		err2 := m.httpServer.ServeTLS(lisn, "", "")
		//err2 := srv.httpServer.ListenAndServeTLS("", "")
		if err2 != nil && !errors.Is(err2, http.ErrServerClosed) {
			err = fmt.Errorf("TLS Server start error: %s", err2.Error())
			slog.Error(err.Error())
		} else {
			slog.Info("TLSServer stopped")
		}
	}()

	return err
}

// Stop the TLS server and close all connections.
// this waits until for up to 3 seconds for connections are closed. After that
// continue.
func (m *HttpServerModule) Stop() {

	if m.httpServer != nil {
		// note that this does not (cannot?) close existing client connections
		ctx, cancelFn := context.WithTimeout(context.Background(), time.Second*30)
		err := m.httpServer.Shutdown(ctx)
		if err != nil {
			slog.Error("Stop: TLS server graceful shutdown failed. Forcing Remove", "err", err.Error())
			_ = m.httpServer.Close()
		}
		cancelFn()
		slog.Info("Stopped HttpTransportModule")
	} else {
		slog.Info("Stop HttpTransportModule - not running")
	}
	// give some time to complete the shutdown
	time.Sleep(time.Millisecond)
}

// Create a new Https server module instance.
//
// moduleID is the module's instance identification.
// config MUST have been configured with a CA and server certificate unless
// NoTLS is set.
func NewHttpServerModule(moduleID string, config *httpserver.HttpServerConfig) *HttpServerModule {

	if moduleID == "" {
		moduleID = transports.DefaultHttpServerModuleID
	}

	m := &HttpServerModule{
		config: config,
	}
	m.authenticateHandler = config.AuthenticateHandler
	if m.authenticateHandler == nil {
		m.authenticateHandler = m.DefaultAuthenticate
	}
	m.SetModuleID(moduleID)
	var _ transports.IHttpServer = m // interface check
	return m
}
