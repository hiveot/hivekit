# HiveKit Factory

The factory is a convenience utility to create module instances for the configured environment. Use of the factory is optional. It does make life easier though so give it a spin.

## Status

This module is in development.

## Summary

The purpose of the factory module is the simplify instantiation and linking of modules for a server application or for a client.

Applications generate module instances using 'CreateInstance(moduleID)'. Module configuration can be defined in the configuration file. See also 'Environment Configuration' below.

This factory module comes with an module factory table that lists all available modules along with the handlers to instantiate them. Applications can expand this table with their own modules or create their own table to keep the binary size to a minimum.

To generate an application a recipe can be used to define the modules and how they are linked. A few commonly used recipes for building clients and servers are included. The recipe defines the moduleID, its request sink and its notification sink. If the application module is registered then the main method is as simple as:

```go (tenative)
func main(){
    env := NewAppEnvironment("", true)
    app := factory.CreateFromRecipe(env, recipe)
    app.WaitForSignal()
}
```

### Environment Configuration

Since many modules operate in an environment that uses files, credentials or network access, it helps to centralize the configuration of this environment and instantiate module instances using this environment.

The first step is therefore to setup the environment:

> env := factory.NewAppEnvironment(homedir, withFlags)

After generating the environment, any properties can be modified at will before passing it to the factory.

### Directory Structure

The homeDir is the root of application. This can follow two approaches, a user home or a system home directory.

When a user home directory is chosen this defines the following application folder structure:

```
~/bin/home
        |- bin               Application binaries, cli and launcher
        |- plugins           Plugin binaries
        |- config            Service configuration yaml files
        |- certs             CA and service certificates
        |- logs              Logging output
        |- run               PID files and sockets
        |- stores
            |- {service}    Data storage for services such as authn
```

When a system home directory is chosen it should be a directory /opt/{appname}. This defines the following folder structure:

```
/opt/{appname}/bin            Application binaries, cli and launcher
/opt/{appname}/plugins        Plugin binaries
/etc/{appname}/conf.d         Service configuration yaml files
/etc/{appname}/certs          CA and service certificates
/var/log/{appname}/           Logging output
/run/{appname}/               PID files and sockets
/var/lib/{appname}/{service}  Storage of service data
```

### Commandline arguments

When building an application it is not uncommon to be able to specify different directories from the commandline.

NewAppEnvironment uses the golang 'flag' library to allow overriding the directories with a corresponding flag:

```
-home         select a different application home directory
-config       select a different configuration file directory
-configFile   select the primary configuration file that holds all module configurations
-logLevel     logging level, debug, info, warn (default), error
-clientID     application clientID when authenticating with a server (for clients)
-serverURL    select a different server (for clients)

```

### Certificates

Servers need certificates and certificates need to be created. The environment defined in the previous paragraph expects certificates to exist in the 'certs' directory. If they don't exist during initialization a set of self-signed certificates will be created.

```
- caCert.pem     - the CA certificate.
- caKey.pem      - the generated CA key for self-signed certificate.
- serverCert.pem - the server x509 certificate in PEM format used by the transports.
- serverKey.pem  - the server private key in PEM format.
```

### Keys

Services that run stand-alone and connect to a server need keys to authenticate. These are stored in the certs directory and are read-only to the user that runs the factory.

The key file has the moduleID as the filename with ".key" as the suffix. The keys can be generated manually using a commandline utility or automatically through a launcher service if used.

If the server side uses the authn module for authentication (recommended) then the keys must be generated using this module.
