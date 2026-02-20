package module

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/httpserver"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
)

// Create the middleware from the configuration.
// This follows the sequence(whereenabled):
//  1. CORS
//  2. Logging
//  3. Recovery
//  4. StripSlashes
//  5. Compression
//  6. Ping
//
// A public route is always created
// A protected route is created when an authentication is enabled in config
//
// # This includes middleware for ping health check
//
// Note that without authentication, the context will not have clientID or sessionID set.
func (m *HttpServerModule) addMiddleware(cfg *httpserver.HttpServerConfig) {
	rootRouter := m.rootRouter

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
			Logger:           m.logger,
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

	// TODO: add csrf support in posts
	//csrfMiddleware := csrf.Protect(
	//	[]byte("32-byte-long-auth-key"),
	//	csrf.SameSite(csrf.SameSiteStrictMode))
	//router.Use(csrfMiddleware)

	if !cfg.StripSlashesEnabled {
		rootRouter.Use(middleware.StripSlashes) // /dashboard/(missing id) -> /dashboard
	}
	//router.Use(csrfMiddleware)
	if cfg.GZipEnabled {
		rootRouter.Use(middleware.Compress(
			cfg.GZipLevel, cfg.GZipContentTypes...))
	}

	// support health monitor
	rootRouter.Use(middleware.Heartbeat(transports.DefaultPingPath))

	//--- public routes do not require a Hub connection
	rootRouter.Group(func(r chi.Router) {
		m.pubRoute = r

		// run a file server if ServeFilesDir is set
		// if filepath.IsAbs(cfg.ServeFilesDir) {
		// 	staticFileServer := http.FileServer(http.Dir(cfg.ServeFilesDir))
		// 	r.Get(cfg.ServeFilesEndpoint, staticFileServer.ServeHTTP)
		// }
	})

	//--- protected routes that requires an authenticated client
	rootRouter.Group(func(r chi.Router) {
		m.protRoute = r
		// authenticate requests in the protected routes
		authWrap := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var clientID string
				var err error

				if cfg.AuthenticateHandler != nil {
					clientID, err = cfg.AuthenticateHandler(r)
				} else {
					clientID, err = m.DefaultAuthenticate(r)
				}
				if err != nil {
					// see https://w3c.github.io/wot-discovery/#exploration-secboot
					// response with unauthorized and point to using the bearer token method
					w.Header().Add("WWW-Authenticate", "Bearer")
					http.Error(w, "Invalid bearer token", http.StatusUnauthorized)
					slog.Warn("HttpsServer Authenticate; ",
						"error", err.Error(),
						"path", r.RequestURI)
					return
				}
				ctx := r.Context()
				ctx = context.WithValue(ctx, transports.ClientIDContextID, clientID)
				next.ServeHTTP(w, r.WithContext(ctx))
			})

		}

		r.Use(authWrap)
	})
}

// GetProtectedRouter returns the router with protected accessible routes for this server.
// This router has cors protection enabled.
// This returns nil if authentication is not configured and will probably
// cause a panic when used.
func (m *HttpServerModule) GetProtectedRoute() chi.Router {
	return m.protRoute
}

// GetPublicRouter returns the router with public accessible routes for this server.
// This router has cors protection enabled.
func (m *HttpServerModule) GetPublicRoute() chi.Router {
	return m.pubRoute
}

// GetRequestParams
func (m *HttpServerModule) GetRequestParams(r *http.Request) (transports.RequestParams, error) {
	return GetRequestParams(r)
}

// GetClientIdFromContext
func (m *HttpServerModule) GetClientIdFromContext(r *http.Request) (string, error) {
	return GetClientIdFromContext(r)
}
