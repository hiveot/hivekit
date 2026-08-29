# HiveKit Cell Factory

The Cell Factory provides the cells to create an application using the provided environment. 

While cells can be created and use on their own, use of *factory recipes* simplifies  construction of an application, as only the application specific logic itself needs to be added. Capabilities like discovery, communication, user management, storage and others are included using the matching recipe.

Recipes are a companion to the factory that constructs a chain, star and/or bus formation from a declared recipe. A recipe is a cell itself that passes requests to the cells in the recipe and returns notifications from the recipe cells. Recipes can be nested.

![chaining](../../docs/cell-chain.png)

## Status

The factory and recipe cells are in alpha. They are functional but breaking changes can be expected.

Roadmap:
* support launching and linking of cells and recipes written in javascript and python.
* support for recipe upload for dynamic application reconfiguration.


## Summary

The purpose of the factory is to simplify instantiation and linking of cells for a client or server applications along with the needed environment. It operates using a collection of registered cells. 3rd party cells can easily be added to the registry. 

To develop an application using the factory, the application logic can be placed in a cell itself and linked to a recipe. The recipe handles the needed capabilities for discovery, communication, storage and much more.

Each cell is registered using a cell type-name and a default implementation. The cell type identifies the interface of the cell implementation. Cells can be replaced with custom functionality as long as the replacement implements the interface for that cell type.

Applications can instantiate a cell using 'GetCell(cellType)'. The cell uses the factory provided environment for directory locations, certificates and auth info as needed. In case of clients the environment offers the server URL which can be set manually or by the discovery service.

The recipes folder contains a set of convenient cookie-cutter recipies for building a consumer, Thing or gateway. See also the examples to see how they are used for creating a test device and a consumer cli.

### Recipe Creation

Recipes are the quickest way to build a client or server application or plugin. They specify which cells are used and how they are linked.

A recipe contains a map of used cells by their cell type, and a list of cells in the order used by their formation. A formation defines how cells are linked. Provided formations are a chain, star or bus. An application is instantiated by invoking recipe.Start(factoryInstance).

Recipes and formations can be used in combination with manually loading cells using the factory GetCell(cellType) method and link them manually using SetRequestHandler and SetResponseHandler. This is best done for linking to the start or end of the recipe.

### Inter-process and Multi-Language Recipes

The factory is written in golang and can only instantiate cells running in the same process. Cells written in a different program language or running on a different host cannot be started directly.

A future feature is to include a launcher for javascript and python cells/recipes and link them through a client/server cell using one of the available transport protocols. 

Alternatively, using the gateway recipe it can accept connections from javascript or python recipes that run on the same or separate hosts. 


### Including 3rd party cells

3rd party cells can be included if they are written in golang. For 3rd party cells written in different languages it is better to define them as plugins. A javascript and python implementation of the factory is planned to simplify writing IoT applications and plugins in those languages.

## Application Environment

Since many cells operate in an environment that uses files, credentials or network access, it helps to centralize the configuration of this environment and instantiate cell instances using this environment.

The first step is therefore to setup the environment:

> env := api.NewAppEnvironment(homedir, withFlags)

Where 'withFlags' allows control of the home and other directories uses commandline flags.

After generating the environment it is used to instantiate the factory and its cells.

### Directory Structure

The homeDir is the root of application. This can follow two approaches, a user home or a system home directory.

When a user home directory is chosen this defines the following application folder structure (on Linux):

```
~/bin/hiveot
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

A Windows directory structure can be accomodated by setting the paths manually. A default structure still needs to be defined. (contributions are welcome)

### Commandline arguments

When building an application it can be neccesary to specify different directories from the commandline.

NewAppEnvironment uses the golang 'flag' library to allow overriding the directories with a corresponding flag:

```
-home         select a different application home directory
-config       select a different configuration file directory
-configFile   select the primary configuration file that holds all cell configurations
-logLevel     logging level, debug, info, warn (default), error
-clientID     application clientID when authenticating with a server (for clients)
-serverURL    select a different server (for clients)

```

### Certificates

Servers need certificates and these certificates need to be created somehow. The environment expects certificates to exist in the configured 'certs' directory.
If they don't exist during initialization a set of self-signed CA and server certificates will be created when a transport server is instantiated.

A self-signed CA certificate is always generated and added to the system pool of available CA's. This is intended for issuing client certificates for authentication, and to create a self-signed server certificate if not provided. Names are standardized as per below. Keys and certificates are in PEM format.

```
- caCert.pem         - the generated CA certificate.
- caKey.pem          - the generated CA private key.
- serverCert.pem     - the server x509 certificate in PEM format used by the transport.
- serverKey.pem      - the server private key.
- {clientID}Cert.pem - Client TLS certificate for client authentication.
```

### Auth Tokens and Certificates

Consumers need authentication (bearer) tokens to connect with HiveOT stand-alone devices and gateways. These tokens can be obtained using a login request send to the gateway/device. 

If a server includes the authn service for authentication (recommended) then the tokens must be generated using this service. The issued tokens are session tokens and need to be renewed periodically. The consumer application can store them in the certs directory for re-use until they expire or are renewed. Note that if the authentication service restarts, sessions tokens become invalid and a re-login is required.


Services and admin user that run stand-alone and connect to a server need a client TLS certificate to authenticate. These are stored in the certs directory and are read-only to the user that runs the factory.

The client cert file has the application-ID or client-ID as the filename with "Cert.pem" suffix. The client certificate can be generated manually using a commandline utility or automatically through a launcher service if used. 


## Application Example

The easiest method to build an application is to use one of the predefined recipes and add the application specific cells. Below some pseudocode for illustration. See also the examples section.

```go (tenative)
func main(){
    // collect the cells to include. Predefined recipes already contain the cells for common use-cases.
	env := api.NewAppEnvironment("~/bin/hiveot", true)
	f := factory_service.NewCellFactory(env, nil)
    recipe := NewStandAloneDeviceRecipe(f)
    // register the recipe cells with the factory and start them.
    err = recipe.Start()
    if err != nil {
        return 
    }

    // create your application and link it to the recipe
    appCell := NewMyAppCell()
    // A: have the recipe handle requests (consumers)
    appCell.SetRequestSink(recipe)
    recipe.SetNotificationSink(appCell)
    // B: have the recipe pass requests (IoT devices and services)
    recipe.SetRequestSink(appCell)
    appCell.SetNotificationSink(recipe)

    appCell.Start()

    // wait for Control-C or other signal to end the application
    f.WaitForSignal(context.Background())
    // Graceful shutdown of all cells in the factory
    f.Stop()
}
```
This is all that is needed to include hivekit and other cells in your application. The developer only needs to provide 'appCell' which provides the application logic and interacts with request handler and notification handlers.


## Future Ideas

In theory it is possible to include all known modules and create a recipe dynamically from an uploaded configuration file. The end result is a single executable that can be configured at runtime as a consumer, device, gateway, hub, depending on the modules that are included. 

The concept of cells can also be applied to a user interface where windows and widgets are dynamically incorporated. This would allow generating a user interface purely through configuration as long as the widget cells are included.