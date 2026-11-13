# Certificate Management Service

This service manages CA, Server and client TLS certificates for use by servers and clients.

This supports self-signed certificates and 3rd party certificate providers such as certbot for use with lets-encrypt (*).  

## Status

This service is in alpha. It has basic functionality but breaking changes can be expected.

(*) The Lets-encrypt provider currently simply serves the server TLS certificate that the administrator obtained from lets-encrypt. It does not (yet) manage the certificates using lets-encrypt directly.


## Summary

To function securely, HiveKit transport servers need a valid server TLS certificate, signed by a trusted CA. HiveKit standardizes access to these certificates through its application environment (AppEnvironment). This environment implements default rules for obtaining these certificates. By using the app environment certificates, cells can directly run a server or connect to one without the need for additional code to manage certificates.

HiveKit expects applications to use the application environment for loading certificates. The application environment only loads certificates and keys from the configured 'certs' directory. It does not create certificates. 

An alternative certificate management service can be used by having it load up the application environment with CA, server and/or client certificates.

The purpose of this 'certs' service is to make valid CA, server and client certificates available for loading by the App Environment.


### Loading of Certificates

This section describes how the application environment loads CA, server and client certificates. The application environment uses two sources of CA certificates. 

1. The first source is the system 'root' CA pool, which houses the same CA certificates as browsers and other applications use. When using an external certificate provider, it is expected that the CA of this provider is loaded in the system pool and no further action is needed. 

2. The second source is the hivekit '{certs}' directory. This directory is determined by the application environment on startup and can be modified through commandline options or environment variables. This defaults to {apphome}/certs where {apphome} defaults to ~/bin/hiveot on Linux.

On instantiation, the application environment determines the location of the {certs} directory, loads the self-signed CA certificate from {certs}/caCert.pem, and adds it to the CA cert pool. If a private key is present in {certs}/caKey.pem it will be loaded as well. See below on when and how this is used.

The certs directory can also hold a server certificate with private key if used, and client certificate with private key if used. 

Use of the certs service is not needed if the self-signed and server certificates are provisioned out-of-band.


### Creating of the self-signed CA Certificate

On startup this service will create a self-signed CA certificate and corresponding private key in the {certs} directory unless a valid self signed CA certificate with key already exists. 

If the private key is missing it will be created along with a new self-signed CA certificate.

If no valid CA certificate is present but the CA private key is available, it will be used to generate a new self-signed CA certificate. Re-using the private key will allow previous self-signed server and client certificates to remain valid.

The self-signed CA is used by this service to generate a client certificate for the administrator, for RC connected devices and plugins, and to generate a server certificate if needed. 

### Creating The TLS Server Certificate

On instantiation this service will check if a valid TLS server certificate exists in the application environment. If no valid certificate exists then it will be created and stored in the {certs} directory and updated in the application environment. 

The method of creating the TLS server certificate depends on whether an external provider is used. Without a provider, this service will generate a self-signed certificate using the self-signed CA. 

If a external provider, like lets-encrypt, is used then it will be asked to create the server certificate using a generated private key. This service will save the certificate and key in the certs directory where it can be loaded by the servers using the application environment.

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

When used in a recipe running a server, a certificate is needed before the server starts. Therefore initialization and linking need to happen at different places in the recipe. The solution is to include this service twice, once before the server is started and once after. The factory will return the same instance the second time it is used. The first instance should have the "NoLink:true" parameter set in the recipe. This instructs the recipe to initialize the service but not link it.

### Using Lets-Encypt

When the lets-encrypt provider is used, it can automatically obtain a server certificate. The CA is already included in the system cert pool so nothing to do here.

Manual out-of-band provisioning of lets-encrypt certificates can be done by copying the server certificate and key from the '/etc/letsencrypt/live' directory to the HiveOT {certs} directory as 'serverCert.pem' and 'serverKey.pem'. This can easily be automated using a cron job.

Only the server cert needs to be deployed this way. Client certificates always use the self-signed CA. Servers are provided with a CA cert pool containing both the lets-encrypt and self-signed CA.

Further automation is planned for the future.
