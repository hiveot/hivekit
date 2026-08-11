# Certificate Management Service Module

This module offers a service for managing CA and Server TLS certificates for use by servers and clients.

This supports self-signed certificate and 3rd party provided certificate providers such as certbot for use with lets-encrypt (*).

## Status

This module is in alpha. It has basic functionality but breaking changes can be expected.

(*) The Lets-encrypt provider currently simply serves the server TLS certificate that the administrator obtained from lets-encrypt. It is intended to work together with the certbot package. If there is a use-case then this can be modified to use the google's autocert as built-in solution.

(**) Should this module provide a certificate chain for use by servers instead of just a TLS certificate? TBD


## Summary

To function securely, HiveKit transport servers need a valid server TLS certificate, signed by a trusted CA. HiveKit standardizes access to these certificates through its AppEnvironment. This environment implements default rules for obtaining these certificates. By using the app environment certificates, modules can directly run a server or connect to one without the need for additional code.

The purpose of this certificate module is to make sure a valid CA and/or TLS certificates are available for the App Environment to load, and enable generating client certificates for connecting to HiveOT servers.

The server certificate is primarily intended for use by the HiveKit servers, although they can also be used by 3rd party servers with a matching name.

### Loading of CA Certificates

HiveKit loads the CA certificates from the system certificate pool into the the App Environment. 

In addition, a self-signed CA is automatically loaded and included if available. The default location of this file defaults to {certsDir}/caCert.pem. 

Use of the self-signed CA is optional and is intended for working with self-signed Server certificate and/or with generating client certificates for authentication.

Administrators can replace this certificate with their own CA if so desired. A corresponding private key is required if the service is asked to generate self-signed server certificates or generating client authentication certificates. 

The behavior depends on availability of the CA and its key:
* If no CA certificate and key are present, they will be generated.
* If no CA certificate is present but the CA private key is available, the CA will be generated using the loaded private key. Previous created TLS/Client certificates will continue to be valid for the generated CA.
* If a CA certificate but no key is present, the CA will be loaded into the pool but self-signed certificate creation will fail. This can be valid if certificate creation is done out of bound.


### Creating of the self-signed CA Certificate

If no CA certificate is present on startup but the CA private key is available, it will be used to generate the self-signed CA certificate. Previous TLS/Client self signed certificates will continue to be valid for the generated CA.

If no CA certificate is present and no private key is found, they will both be created. Existing self-signed certificates will no longer be valid.

If a CA certificate is present but no private key is found then self-signed certificates cannot be created. This mode is specific for users that manage self-signed certificates themselves. They must ensure that the CA, Server Certificate and private key, and client certificate and private key are installed in the {certs} folder.

Please note that even if Lets-Encrypt is used as the provider, the self-signed CA is still needed for creating authentication client certificates.


### Loading of Server TLS Certificate

TLS Server certificate/key-pairs can be loaded using the application environment GetTLSCert(serverName) method. 

This first attempts to locate a x509 certificate and corresponding private key from {certDir}/{serverName}Cert.pem and {certDir}/{serverName}Key.pem. If not found, the lets-encrypt provider is checked for its location at /etc/letsencrypt/live on linux.


### Creating TLS Server Certificate

Each provider has its own method to create the TLS server certificate.

Self-signed certificate provider:

When requested, the self-signed certificate provider locates or generates a self-signed TLS certificate using the self-signed CA. These are stored in the hiveot 'certs' directory. If a server key key exists it is re-used, otherwise it is newly generated. 

Built-in Lets-Encrypt provider:

The lets-encrypt provider uses the acme/autocert package to obtain and refresh a Lets-encrypt certificate. It stores these in the hiveot 'certs' directory.

Certbot provider:

Certbot is an external utility for getting certificates from Lets-Encrypt.

Refresh can also be called manually on the module, which forwards it to the selected provider.


## Usage

Prerequisite: To have permissions to read the certificates created by this module, the application must run as the same user as this module, or the certificate files ownership must be changed to that of the hivekit application if it differs.

### Manual Instantiation

To manually create the module instance:

```golang
testCertDir := "./certs"
m := module.NewCertsModule(certDir)
```

### Module Factory Instantiation

The certificate service can be added to the module factory using 'certs_service.CertsServiceFactory' method and the certs.CertsServerModuleType as the module type. This enables admin access to manage CA and server certificate configuration. Intended for gateways but can also be used on stand-alone devices.

To ensure certificates are created if they cannot be located by the app environment, a factory 'NewInitFactoryCerts' initializer module must be added to the start of the module chain:
> factory method: certs_service.NewInitFactoryCerts
> module type name: certs.InitFactoryCertsModuleType

This generates and loads self-signed CA certificate if it wasnt found by the app environment. In addition it generates and loads a TLS Server certificate with the serverID name. It does nothing if the CA/Server certificates are already loaded.

### Using certbot for Lets-Encypt

When the certbot utility is used, it automatically obtains and refreshes a server certificate. The CA is already included in the system cert pool so nothing to do here.

By default the letsencrypt provider locates the server certificate in the '/etc/letsencrypt/live' directory. The server needs the certificate chain for it to be recognized.





