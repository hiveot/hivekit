# Certificate Management Service

This service manages CA, Server and client TLS certificates for use by servers and clients.

This supports self-signed certificates and 3rd party certificate providers such as certbot for use with lets-encrypt (*).  

## Status

This service is in alpha. It has basic functionality but breaking changes can be expected.

(*) The Lets-encrypt provider currently simply serves the server TLS certificate that the administrator obtained from lets-encrypt. It does not (yet) manage the certificates using lets-encrypt directly.


## Summary

To function securely, HiveKit transport servers need a valid server TLS certificate, signed by a trusted CA. HiveKit standardizes access to these certificates through its application environment (AppEnvironment). This environment implements default rules for obtaining these certificates. By using the app environment certificates, cells can directly run a server or connect to one without the need for additional code to manage certificates.

HiveKit expects applications to use the application environment for loading certificates. The application environment only loads certificates from the configured 'certs' directory. It does not create certificates. 

The purpose of this 'certs' service is to make valid CA, server and client certificates available for loading by the App Environment.


### Loading of Certificates

This section describes how the application environment loads CA, server and client certificates. The application environment uses two sources of CA certificates. 

1. The first source is the system 'root' CA pool, which houses the same CA certificates as browsers and other applications use. When using an external certificate provider, it is expected that the CA of this provider is loaded in the system pool and no further action is needed. 

2. The second source is the hivekit '{certs}' directory. This directory is determined by the application environment on startup and can be modified through commandline options or environment variables. This defaults to {apphome}/certs where {apphome} defaults to ~/bin/hiveot on Linux.

On instantiation, the application environment determines the location of the {certs} directory, loads the self-signed CA certificate from {certs}/caCert.pem, and adds it to the CA cert pool. If a private key is present in {certs}/caKey.pem it will be loaded as well. See below on when and how this is used.

The application environment also provides methods to load a server TLS certificate, and client TLS certificates. This simply locates the TLS certificate in {certs}/{name}Cert.pem and {certs}/{name}Key.pem where name is the server certificate name, or the clientID in case of client certificates. As a convention  this is the 'cn' field in the certificate.

If a certificate provider is used for provide a Server certificate, the service stores the certificate in the {certs} directory to allow the application environment to load it along with the key used to generate the certificate.


### Creating of the self-signed CA Certificate

On startup this service will create a self-signed CA certificate and corresponding private key in the {certs} directory unless a valid CA certificate already exists. If the private key is missing it will be created along with a new self-signed CA certificate.

If no valid CA certificate is present on startup of this 'certs' service but the CA private key is available, it will be used to generate the self-signed CA certificate. Re-using the private key will allow previous self-signed server and client certificates to be used.

Please note that even if Lets-Encrypt is used as the provider, the self-signed CA and key are still needed for creating authentication client certificates. If client certificates are not used then the CA will still be created but not used.

### Creating The TLS Server Certificate

On startup this service will check if a valid TLS server certificate exists in the application environment. If no valid certificate exists then it will be created and stored in the {certs} directory where the application environment loads it from. The default server certificate name is the application ID as expected by the application environment. 

The method of creating the TLS server certificate depends on whether an external provider is used. Without a provider, this service will generate a self-signed certificate using the self-signed CA. 

If a external provider, like lets-encrypt, is used then it will be asked to create the server certificate using the provided private key. This service will save the certificate and key in the certs directory where it can be loaded by the servers using the application environment.

### Creating A Client Certificate

This service supports creating self-signed client certificates that work with the built-in self-signed CA. Application servers that support authentication through client certificates should use the CA pool from the application environment so client certificates will automatically be recognized.

To create a client certificate, an administrator user invokes the 'CreateClientCert' service method, providing the client loginID, certificate validity period and client public key. 

Invoking the CreateClientCert method must be done using an application that ensures only administrators with proper permissions can invoke this method. 


## Usage

Prerequisite: To have permissions to read the certificates created by this service, the application must run as the same user as this service, or the certificate files ownership must be changed to that of the hivekit application if it differs.

### Manual Instantiation

To manually create the service instance:

```golang
testCertDir := "./certs"
svc := certs_service.NewCertsService(certDir)
```

### Factory Instantiation

The certificate service can be added to the cell factory using the 'certs_service.NewCertsServiceFactory' method and the certs.CertsServiceCellType as the cell type. This enables admin access to manage CA and server certificate configuration. Intended for gateways but can also be used on stand-alone devices.

To ensure certificates are created if they cannot be located by the app environment, a factory 'NewInitFactoryCerts' initializer cell must be added to the start of the cell chain:
> factory method: certs_service.NewInitFactoryCerts
> cell type name: certs.InitFactoryCertsCellType

This generates and loads self-signed CA certificate if it wasnt found by the app environment. In addition it generates and loads a TLS Server certificate with the serverID name. It does nothing if the CA/Server certificates are already loaded.

### Using Lets-Encypt

When the lets-encrypt provider is used it can automatically obtain a server certificate. The CA is already included in the system cert pool so nothing to do here.

The letsencrypt provider locates server certificates in the '/etc/letsencrypt/live' directory and stores it in the {certs} directory. Further automation is planned for the future.





