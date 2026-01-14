package module

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/hiveot/hivekit/go/modules/transports/httptransport"
	"github.com/hiveot/hivekit/go/utils"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
)

// Create the middleware from the configuration.
// This follows the sequence(whereenabled): CORS-Logging-Recovery-StripSlashes-Compression
// A public route is always created
// A protected route is created when an authentication is enabled in config
//
// Note that without authentication, the context will not have clientID or sessionID set.
func (m *HttpTransportModule) addMiddleware(cfg *httptransport.HttpServerConfig) {
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
		if cfg.ValidateToken != nil {
			m.protRoute = r
			// authenticate requests in the protected routes
			//
			authWrap := func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var clientID string
					var sid string
					var err error

					if cfg.AuthenticateHandler != nil {
						clientID, sid, err = cfg.AuthenticateHandler(r)
					} else {
						clientID, sid, err = m.Authenticate(r)
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
					ctx := context.WithValue(r.Context(), httptransport.SessionContextID, sid)
					ctx = context.WithValue(ctx, httptransport.ClientContextID, clientID)
					next.ServeHTTP(w, r.WithContext(ctx))
				})

			}

			r.Use(authWrap)
		} else {
			slog.Warn("HTTP server does not have authentication configured. The protected route is not available.")
		}
	})
}

// The default Authentication handler that reads the bearer token from the request
// and uses the ValidateToken function to validate it.
func (m *HttpTransportModule) Authenticate(r *http.Request) (clientID string, sid string, err error) {

	bearerToken, err := utils.GetBearerToken(r)
	if err != nil {
		return "", "", err
	}
	if m.config.ValidateToken == nil {
		return "", "", fmt.Errorf("Authenticate: missing cfg.ValidateToken handler")
	}
	//check if the token is properly signed
	clientID, sid, err = m.config.ValidateToken(bearerToken)
	if err != nil {
		return "", "", err
	} else if clientID == "" {
		return "", "", fmt.Errorf("Missing ClientID")
	}

	return clientID, sid, nil
}

// AddSessionFromToken middleware decodes the bearer session token in the authorization header.
//
// Session tokens can be provided through a bearer token or a client cookie. The token
// must match with an existing session ID.
//
// This distinguishes two types of tokens. Those with and those without a session ID.
// If the token contains a session ID then that session must exist or the token is invalid.
// User tokens are typically session tokens. Closing the session (logout) invalidates the token,
// even if it hasn't yet expired. Sessions are currently only stored in memory so a service
// restart also invalidates all session tokens.
//
// Non-session tokens, are used by services and device agents. These tokens are generated
// on provisioning or token renewal and last until their expiry.
//
// The session can be retrieved from the request context using GetSessionFromContext()
//
// The client session contains the client ID, and stats for the current session.
// If no valid session is found this will reply with an unauthorized status code.
//
// pubKey is the public key from the keypair used in creating the session token.
func (m *HttpTransportModule) AddSessionFromToken() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			bearerToken, err := utils.GetBearerToken(r)
			if err != nil {
				// see https://w3c.github.io/wot-discovery/#exploration-secboot
				// response with unauthorized and point to using the bearer token method
				errMsg := "AddSessionFromToken: " + err.Error()
				w.Header().Add("WWW-Authenticate", "Bearer")
				http.Error(w, errMsg, http.StatusUnauthorized)
				slog.Warn(errMsg)
				return
			}
			//check if the token is properly signed
			clientID, sid, err := m.config.ValidateToken(bearerToken)
			if err != nil || clientID == "" {
				w.Header().Add("WWW-Authenticate", "Bearer")
				http.Error(w, err.Error(), http.StatusUnauthorized)
				slog.Warn("AddSessionFromToken: Invalid session token:",
					"err", err, "clientID", clientID)
				return
			} else if clientID == "" {
				w.Header().Add("WWW-Authenticate", "Bearer")
				http.Error(w, "missing clientID", http.StatusUnauthorized)
				slog.Warn("AddSessionFromToken: Missing clientID")
				return
			}

			// make session available in context
			//ctx := context.WithValue(r.Context(), subprotocols.SessionContextID, cs)
			ctx := context.WithValue(r.Context(), httptransport.SessionContextID, sid)
			ctx = context.WithValue(ctx, httptransport.ClientContextID, clientID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetProtectedRouter returns the router with protected accessible routes for this server.
// This router has cors protection enabled.
// This returns nil if authentication is not configured and will probably
// cause a panic when used.
func (m *HttpTransportModule) GetProtectedRoute() chi.Router {
	return m.protRoute
}

// GetPublicRouter returns the router with public accessible routes for this server.
// This router has cors protection enabled.
func (m *HttpTransportModule) GetPublicRoute() chi.Router {
	return m.pubRoute
}

// GetRequestParams
func (m *HttpTransportModule) GetRequestParams(r *http.Request) (httptransport.RequestParams, error) {
	return GetRequestParams(r)
}

// GetClientIdFromContext
func (m *HttpTransportModule) GetClientIdFromContext(r *http.Request) (string, error) {
	return GetClientIdFromContext(r)
}
