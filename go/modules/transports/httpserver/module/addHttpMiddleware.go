package module

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/hiveot/hivekit/go/modules/transports/httpserver"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
)

// Create the middleware from the configuration.
// This follows the sequence(whereenabled): CORS-Logging-Recovery-StripSlashes-Compression
// The public and protected routes are added after this chain.
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
		m.pubRoute = r
		// run a file server if ServeFilesDir is set
		// if filepath.IsAbs(cfg.ServeFilesDir) {
		// 	staticFileServer := http.FileServer(http.Dir(cfg.ServeFilesDir))
		// 	r.Get(cfg.ServeFilesEndpoint, staticFileServer.ServeHTTP)
		// }
	})

	//--- private routes that requires an authenticated client
	rootRouter.Group(func(r chi.Router) {
		m.protRoute = r
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
func (m *HttpServerModule) GetProtectedRoute() chi.Router {
	return m.protRoute
}

// GetPublicRouter returns the router with public accessible routes for this server.
// This router has cors protection enabled.
func (m *HttpServerModule) GetPublicRoute() chi.Router {
	return m.pubRoute
}

// GetRequestParams
func (m *HttpServerModule) GetRequestParams(r *http.Request) (httpserver.RequestParams, error) {
	return GetRequestParams(r)
}

// GetClientIdFromContext
func (m *HttpServerModule) GetClientIdFromContext(r *http.Request) (string, error) {
	return GetClientIdFromContext(r)
}
