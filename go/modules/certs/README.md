# Certificate Management Service Module

This module offers services for creating keys and certificates for use by other modules.

## Configuration

This module can be configured to read certs and keys from file, generate self-signed certs, obtain it from Lets-Encrypt or other provider.

## Usage

To manually create the module instance:

```golang
testCertDir := "./certs"
m := module.NewCertsModule(certDir)
```

### Pipeline Factory

// todo

### Manual Setup
