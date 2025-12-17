# Transports TLS Module

The TLS Module provides a basic TLS server with standard configuration.

It is intended for use by http based transport protocols. These protocols only need a router to function so it is easy to substitute this TLS server with another.

## Configuration

To operate a TLS server this requires:

- Public and Private key-pair, used in creating a certificate request
- TLS certificate signed by a CA
- CA public certificate

Fixed configuration:

- CORS requests from 127.0.0.1 or localhost are always allowed. Intended for testing.
- CORS requests from https://server-addr/ same origin is allowed
- CORS requests from "" no origin is allowed but is logged as a warning
- Allowed headers: "Origin", "Accept", "Content-Type", "Authorization", "Headers"
- Allowed methods: GET, PUT, POST, PATCH, DELETE, OPTIONS

## Usage

There are two ways to create a TLS server module instance: manually using a certs module, or using the zeroconfig pipeline factory.

### Pipeline Factory

When using the pipeline factory, the router is automatically instantiated when a http based message transport is used.
This uses the default configuration, listening on port 443 using pipeline keys and certificate. The pipeline obtains its keys from the Certs module. This module can be configured to read certs and keys from file, generate self-signed certs, obtain it from Lets-Encrypt or other provider.

### Manual Setup

Manual TLS Server creation requires a port configuration, listening address, public key, private key, TLS certificate and CA certificate used to create the TLS certificate.
The key and certificates are obtained from a certs module.

Defaults: port: 443
Addr: "" any address

```go
certsModule := NewCertsModule(...)
module := NewTLSModule(port, addr, certsModule)
err := module.Start()
router := module.GetRouter()
```

The router can directly be used in the transport modules.
