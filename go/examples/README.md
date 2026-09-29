# examples

These examples demonstrate how to build an ecosystem of IoT devices and services using HiveKit. The examples can be used on their own or together.

The examples can be run directly using:
> go run sadevice/main.go -home=~/bin/hiveot 

or by building and running:
> make 
> dist/sadevice -home=~/bin/hiveot

This uses the "~/bin/hiveot" directory as home directory for config, certificates, and data storage. 


## Basic Examples

These examples are intended to demonstrate how to build an application using HiveKit cells. They are not intended as production-ready applications. They do however help in getting started.  

These examples use two types of authentication, an 'admin' client account with an admin.token file in the certs directory, and an admin client certificate, signed by the self-signed CA. The standalone and gateway example add this client. The cli and tui examples expect either the admin.token or adminCert/Key.pem files to exist in the certs directory.

### sadevice: Run a Standalone Counter Device

Sadevice creates a standalone IoT device that runs a simple counter. It has a property with the current value, sends an event when it changes and has actions for increment and decrement.

This uses the StandaloneDeviceRecipe factory recipe to create a discoverable server and link it to the counter service. It includes authentication for an admin user. A CLI (next example) can be used to discover and read the device.

The 'ExposedThing' cell is used to create a counter example that can receive requests and emit notifications. The counter example device:
- Provide a websocket server for connecting to the device.
- Publishes an event each time the counter value changes.
- Provide actions for incrementing and decrementing the counter.
- Serves discovery of the device TD.
- Authenticate requests

usage: go run sadevice/main.go --home ~/bin/hiveot

### cli: Discovery CLI

A simple commandline consumer utility to discover Things and Directories on the network and optionally show their TD. Use -h to view available filter and display options.

usage: go run cli/main.go [-h] -home ~/bin/hiveot 

This shows the supported commands including discovery and thing status. 
Thing actions can be invoked. Capturing input parameters is not supported.


### tui: Console Text UI

tui is a text UI consumer showing discovered devices and their TD.

usage: go run tui/main.go -home ~/bin/hiveot

This starts with discovery of directories and devices on the network and lists a tree of the available Things. Selecting a Thing shows its TD with property values, if the credentials are available.
 

### gateway: Gateway Server

The gateway server runs multiple protocol servers that consumers connect to for access to standalone and RC devices. It uses the gateway recipe that includes a group of transport servers, a discovery server, a directory, and a router to forward requests from consumer to standandalone and RC devices. The authn service authenticates request from consumers. 
Supported transport protocols include: WoT websocket, HiveOT websocket, HiveOT SSE-SC and HiveOT gRPC.

This works with the cli and tui consumers, and the rcdevice examples.


### rcdevice: RC Device (reverse connection) [todo]

The rcdevice is the same device as the 'sadevice', but instead of running a server it connects to the gateway using reverse connection.

Since rc devices dont run servers, they are the simplest way to construct a device. They don't need servers, authentication and routing. Instead a reconnect cell automatically tries to reconnect the transport client.

This is the preferred way to create and connect devices in HiveOT. 

### appenv: show the application environment 



## What is a recipe?

A recipe is a list of cells organized in a chain, bus or star formation. It makes it 