# Transports HTTPS Server Module

The HTTP TLS Server Module is intended for use by http based transport protocols. This server includes support for common middleware such as cors, logging, recovery, compression. authentication and file server and provides callback hooks for logging and authentication interaction.

The server provides two convenient routers for adding endpoints, a secured router which requires authentication and an unsecured router.

## Configuration

To operate a TLS server this requires:

- CA public certificate
- TLS certificate (x509 certificate + private key) signed by a CA

CORS configuration is only needed when serving web browsers. If enabled:

- CORS requests from 127.0.0.1 or localhost are always allowed. Intended for testing.
- CORS requests from https://{server-addr}/ same origin is allowed
- CORS requests from configured origins are allowed. Anything else is blocked.
- Default headers: "Origin", "Accept", "Content-Type", "Authorization", "Headers"
- Default methods: GET, PUT, POST, PATCH, DELETE, OPTIONS

Callback hooks:

- config.Authenticate is a function to authenticate incoming requests for the protected routes.
- config.Logger is the handler for logging http requests. Default is the chi middleware.Logger.

Other configuration:

- Address: default "" (any)
- Port: default 8444
- Logger: defaults to chi's logger
- Recovery: enabled
- Compression: 5, gzip, ... or brotli
- StripSlashes: disabled

## Usage

There are two ways to create a HttpsBase module instance: using the pipeline factory or manually.

### Pipeline Factory

When using the pipeline factory, the server is automatically instantiated when a http based message transport is needed.
This uses the default configuration, listening on port 8444 using pipeline keys and certificate. The pipeline obtains its keys from the Certs module. This module can be configured to read certs and keys from file, generate self-signed certs, obtain it from Lets-Encrypt or other provider.

### Manual Setup

Manual HTTPS Server creation requires configuration with a listening port, server TLS certificate, CA certificate, and authenticator.

The certificates can be loaded manually or obtained from the certs module.
The authenticator can be supplied manually or obtained from the authn module.

See NewHttpsBaseOptions for defaults.

```go
 config := NewHttpsBaseOptions()
 config.CaCert = certsModule.GetCACert(),
 config.ServerCert = certsModule.GetDefaultServerCert(),
 config.Authenticator = authnModule.GetAuthenticator()
 module := NewHttpsBaseModule(config)
 err := module.Start()
 prouter := module.GetPublicRouter()
 srouter := module.GetSecuredRouter()
```

The routers can directly be used in the transport modules.
