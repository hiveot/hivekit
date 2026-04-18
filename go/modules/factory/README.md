# HiveKit Module Factory

The module factory creates module instances for the configured environment. Use of the factory is optional. It does make life easier though so give it a spin.

The factory is a module itself that passes requests to the chain of loaded modules and sends notifications.

## Status

This module is in alpha. It is functional but breaking changes can be expected.

## Summary

The purpose of the factory module is the simplify instantiation and linking of (golang) modules for a client or server application. It operates using a collection of registered modules. 3rd party modules can easily be added to the registry. By making modules for the customized logic, a complete application can be generated from the factory.

The factory can be used to obtain individual modules or a chain of modules following a 'recipe'. When obtaining individual modules there is no need to know the right order of module instantiation as each module can use the factory to load additional modules it depends on.

Some functionality, such as authentication, might not be available until later. For this case the factory uses a proxy implementation that can be linked to a module that offers the desired functionality. In most cases however the factory simply instantiates the registered module.

Each module is registered using a module-type. The module type identifies the implementation and can require that a specific interface is implemented. Modules can be replaced with custom functionality as long as the replacement implements the interface for that module type.

Applications can generate the module instances using 'GetModule(moduleType)'. The module uses the factory provided environment to obtain directory locations, certificates as needed.

## Usage

The easiest method to build an application is to use one of the predefined recipes and add the application specific module.

```go (tenative)
func main(){
    // 1. start with one of the factory templates and add the application
    recipe := templates.StandardServerRecipe()
    // 2. optionally register the application as a module or modify the recipe
    recipe.AddModule(MyAppModuleType, NewAppModuleFn)
    // 3. create and optionally modify the environment
    env := factory.NewAppEnvironment("", true)
    // 4. instantiate the factory and run the recipe
    f := factory.NewModuleFactory(env, nil)
    f.StartRecipe(recipe)
    // 5. wait for Control-C or other signal to end the application
    app.WaitForSignal()
    // 6. Graceful shutdown
    app.StopAll()
}
```

The factory includes predefined recipes for building client and server applications. The user can use one of these to generate a template and add to it.

## Recipe Creation

Recipes are the quickest way to build a client or server application or plugin. They specify wich modules are used and how they are chained.
Each recipe contains a map of module factory functions by their module type, and a list of modules in the order they are linked.

Use of recipes is optional as a user can also load modules and link them manually.

Alternatively, a chain created from a recipe can be expanded with a custom module by calling app.Append(factory).

3rd party modules can be embedded if they are written in golang. For 3rd party modules written in different languages it is better to define them as plugins. A javascript and python implementation of the factory is planned to simplify writing IoT applications and plugins in those languages.

## Application Environment

Since many modules operate in an environment that uses files, credentials or network access, it helps to centralize the configuration of this environment and instantiate module instances using this environment.

The first step is therefore to setup the environment:

> env := factory.NewAppEnvironment(homedir, withFlags)

Where 'withFlags' allows control of the home and other directories uses commandline flags.

After generating the environment it is used to instantiate the factory and its modules.

### Directory Structure

The homeDir is the root of application. This can follow two approaches, a user home or a system home directory.

When a user home directory is chosen this defines the following application folder structure:

```
~/bin/home
        |- bin               Application binaries, cli and launcher
        |- plugins           Plugin binaries controlled by the launcher
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
/opt/{appname}/plugins        Plugin binaries that are started and stopped using the launcher
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

The key file has the application-ID as the filename with ".key" as the suffix. The keys can be generated manually using a commandline utility or automatically through a launcher service if used. By default the application-ID is the name of the binary.

If the server side uses the authn module for authentication (recommended) then the keys must be generated using this module.
