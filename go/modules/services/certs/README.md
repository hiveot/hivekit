# Certificate Management Service Module

This module offers services for creating keys and certificates for use by clients and servers.

## Configuration

This module can be configured to read certs and keys from file, generate self-signed certs, obtain it from Lets-Encrypt or other provider.

## Usage

There are two ways to create a certificate module instance: manually using a certs module, or using the zeroconfig pipeline factory.

### Pipeline Factory

When using the pipeline factory, the certificate module is automatically instantiated using the pipeline configuration.

### Manual Setup
