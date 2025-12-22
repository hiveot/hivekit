package service

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

	"github.com/hiveot/hivekit/go/modules/transports/httpserver"
	"github.com/lmittmann/tint"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
)

// HttpsServer is a simple TLS transport supporting CORS and authentication
type HttpsServer struct {
	config     *httpserver.HttpServerConfig
	httpServer *http.Server
	rootRouter *chi.Mux
	pubRoutes  chi.Router
	protRoutes chi.Router

	//
	logger *log.Logger
}

// Create the middleware from the configuration.
// This follows the sequence(whereenabled): CORS-Logging-Recovery-StripSlashes-Compression
// The public and protected routes are added after this chain.
func (srv *HttpsServer) addMiddleware(cfg *httpserver.HttpServerConfig) {
	rootRouter := srv.rootRouter

	// handle CORS using the cors plugin
	// see also: https://stackoverflow.com/questions/43871637/no-access-control-allow-origin-header-is-present-on-the-requested-resource-whe
	// TODO: add configuration for CORS origin: allowed, sameaddress, exact
	if cfg.CorsEnabled {
		corsMiddleware := cors.New(cors.Options{

			// return the origin as allowed origin
			// AllowOriginFunc: func(orig string) bool {
			// 	// local requests are always allowed, even over http (for testing) - todo: disable in production
			// 	if strings.HasPrefix(orig, "https://127.0.0.1") || strings.HasPrefix(orig, "https://localhost") ||
			// 		strings.HasPrefix(orig, "http://127.0.0.1") || strings.HasPrefix(orig, "http://localhost") {
			// 		slog.Debug("TLSServer.AllowOriginFunc: Cors origin Is True", "origin", orig)
			// 		return true
			// 	} else if strings.HasPrefix(orig, "https://"+cfg.Address) {
			// 		slog.Debug("TLSServer.AllowOriginFunc: Cors origin Is True", "origin", orig)
			// 		return true
			// 	} else if orig == "" {
			// 		// same-origin is allowed
			// 		return true
			// 	}
			// 	slog.Warn("TLSServer.AllowOriginFunc: Cors: invalid origin:", "origin", orig)
			// 	// for testing just warn about invalid origin
			// 	return true
			// },
			AllowedOrigins: cfg.CorsAllowedOrigins,
			// default allowed headers is "Origin", "Accept", "Content-Type", "X-Requested-With" (missing authorization)
			AllowedHeaders: []string{"Origin", "Accept", "Content-Type", "Authorization", "Headers"},
			//AllowedHeaders: []string{"*"},
			// ExposedHeaders: []string{"Link"},
			// default is get/put/patch/post/delete/head
			AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
			Debug:          false,
			//Debug:            true, // the AllowOriginFunc above does the reporting
			AllowCredentials: true,
			Logger:           srv.logger,
			MaxAge:           300, // Preflight request cache duration in seconds
		})
		rootRouter.Use(corsMiddleware.Handler)
	}
	if cfg.Logger != nil {
		rootRouter.Use(cfg.Logger)
	}
	if cfg.Recoverer != nil {
		rootRouter.Use(cfg.Recoverer)
	}
	if !cfg.StripSlashesEnabled {
		rootRouter.Use(middleware.StripSlashes) // /dashboard/(missing id) -> /dashboard
	}
	//router.Use(csrfMiddleware)
	if cfg.GZipEnabled {
		rootRouter.Use(middleware.Compress(
			cfg.GZipLevel, cfg.GZipContentTypes...))
	}

	//--- public routes do not require a Hub connection
	rootRouter.Group(func(r chi.Router) {
		srv.pubRoutes = r
		// run a file server if ServeFilesDir is set
		// if filepath.IsAbs(cfg.ServeFilesDir) {
		// 	staticFileServer := http.FileServer(http.Dir(cfg.ServeFilesDir))
		// 	r.Get(cfg.ServeFilesEndpoint, staticFileServer.ServeHTTP)
		// }
	})

	//--- private routes that requires an authenticated client
	rootRouter.Group(func(r chi.Router) {
		srv.protRoutes = r
		if cfg.Authenticate != nil {
			// authenticate requests in the protected routes
			authWrap := func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					clientID, sid, err := cfg.Authenticate(r)
					if err != nil {
						// see https://w3c.github.io/wot-discovery/#exploration-secboot
						// response with unauthorized and point to using the bearer token method
						w.Header().Add("WWW-Authenticate", "Bearer")
						http.Error(w, "missing clientID", http.StatusUnauthorized)
						slog.Warn("HttpsServer Authenticate; Missing clientID;",
							"path", r.RequestURI)
						return
					}
					ctx := context.WithValue(r.Context(), httpserver.SessionContextID, sid)
					ctx = context.WithValue(ctx, httpserver.ClientContextID, clientID)
					next.ServeHTTP(w, r.WithContext(ctx))
				})

			}

			r.Use(authWrap)
		}
		// add the authenticator
	})
}

// GetProtectedRouter returns the router with protected accessible routes for this server.
// This router has cors protection enabled.
func (srv *HttpsServer) GetProtectedRouter() chi.Router {
	return srv.protRoutes
}

// GetPublicRouter returns the router with public accessible routes for this server.
// This router has cors protection enabled.
func (srv *HttpsServer) GetPublicRouter() chi.Router {
	return srv.pubRoutes
}

// Start the TLS server using the provided configuration.
// This fails if ca or server certificates are missing, unless NoTLS is enabled.
//
// This configures handling of CORS requests to allow:
//   - any origin by returning the requested origin (not using wildcard '*').
//   - any method, eg PUT, POST, GET, PATCH,
//   - headers "Origin", "Accept", "Content-Type", "X-Requested-With"
func (srv *HttpsServer) Start() error {
	var err error
	var tlsConf *tls.Config
	cfg := srv.config

	slog.Info("Starting TLS server", "address", cfg.Address, "port", cfg.Port)
	if cfg.CaCert == nil || cfg.ServerCert == nil {
		//no TLS possible
		if cfg.NoTLS == false {
			err := fmt.Errorf("missing CA or server certificate")
			slog.Error(err.Error())

			return err
		}
	} else {
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
	srv.logger = slog.NewLogLogger(logHandler, slog.LevelDebug)

	srv.rootRouter = chi.NewRouter()
	srv.addMiddleware(cfg)

	srv.httpServer = &http.Server{
		Addr: fmt.Sprintf("%s:%d", cfg.Address, cfg.Port),
		// ReadTimeout:  5 * time.Minute, // 5 min to allow for delays when 'curl' on OSx prompts for username/password
		// WriteTimeout: 10 * time.Second,
		Handler:   srv.rootRouter,
		TLSConfig: tlsConf,
		//ErrorLog:  log.Default(),
	}
	lisn, err := net.Listen("tcp", srv.httpServer.Addr)
	if err != nil {
		return err
	}
	// mutex to capture error result in case startup in the background failed
	go func() {
		// serverTLSConf contains certificate and key
		err2 := srv.httpServer.ServeTLS(lisn, "", "")
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

// Stop the TLS server and close all connections
// this waits until for up to 3 seconds for connections are closed. After that
// continue.
func (srv *HttpsServer) Stop() {
	slog.Info("Stopping TLS server")

	if srv.httpServer != nil {
		// note that this does not (cannot?) close existing client connections
		ctx, cancelFn := context.WithTimeout(context.Background(), time.Second*3)
		err := srv.httpServer.Shutdown(ctx)
		if err != nil {
			slog.Error("Stop: TLS server graceful shutdown failed. Forcing Remove", "err", err.Error())
			_ = srv.httpServer.Close()
		}
		cancelFn()
	}
}

// NewHttpsServer creates a new TLS Server instance with authentication support.
// This returns the chi-go router which can be used to add routes and middleware.
// This server supports the "message-id" header for received requests.
//
// Use Start() to start listening.
//
// The middleware handlers included with the server can be used for authentication.
//
//	address        server listening address
//	port           listening port
//	serverCert     Server TLS certificate
//	caCert         CA certificate to verify client certificates
//
// returns TLS server and router for handling requests
func NewHttpsServer(config *httpserver.HttpServerConfig) *HttpsServer {

	srv := &HttpsServer{config: config}

	//// support for CORS response headers
	//srv.router.Use(mux.CORSMethodMiddleware(srv.router))

	return srv
}
