# examples

These examples demonstrate how to build an ecosystem of IoT devices and services using HiveKit. The examples can be used on their own or together.

The examples can be run directly using:
> go run example1/main.go ---home ~/bin/hiveot 

or by building and running:
> make examples
> dist/example1 --home ~/bin/hiveot

This uses the "~/bin/hiveot" directory as home directory for config, certificates, and data storage. 


## Simple Examples

These first few examples are kept simple on purpose. They use a single protocol and lacks authentication, authorization and offers no history.

### Example 1: Run a Standalone Counter Device

Example 1 creates a standalone IoT device that runs a simple counter. It has a property with the current value, sends an event when it changes and has actions for increment and decrement.

- This uses a factory recipe to create a server and link it to the counter service.
- This publishes an event each time the counter value changes.
- This offers actions for incrementing and decrementing the counter.
- This serves Thing discovery of the service TD and can be discovered with example 2.

usage: go run example1/main.go --home ~/bin/hiveot

### Example 2. Discovery CLI

A simple commandline utility to discover Things and Directories on the network and optionally show their TD. Use -h to view available filter and display options.

usage: go run example2/main.go [-h] -home ~/bin/hiveot 

This shows the supported commands including discovery and status.

### Example 3. Browser TUI

Example 3 is a text UI shows discovered devices and their TD.

usage: go run example3/main.go -home ~/bin/hiveot

This displays a menu with options. Commands:

- discover devices and directories
- list discovered TDs
- show details of a selected TD
- when client cert or token authentication is available :
   - view device status with property and latest event values
   - invoke actions (no input yet)
 

### Example 4. Gateway Server

The gateway server runs a server that consumers connect to for access to standalone and RC devices. It uses the gateway recipe that includes a discovery server; a directory with discovered and registered devices; a router to forward requests from consumer to standandalone and RC devices, and the authn service for authentication of consumers and RC devices. 


This can be used with example 1, 2, 3 and 5.



### Example 5. RC Device (reverse connection) [todo]

This example constructs a RC device that uses a reverse connection to a gateway. It contains a test device and a client for a gateway.

This is the preferred way to create and connect devices in hiveot. It does require the gateway from example 4. Note that the hiveot Hub is intended as an out-of-the-box gateway.


## Usage

Simply start ./dist/example1 to run it. Press Ctrl-C to terminate.

To view the commandline options:

> ./dist/example1 -h

HiveKit looks for certificates and keys in the {home}/certs directory which defaults to ~/bin/hiveot/certs.
A different home directory can be passed using -home=/path/to/my/home.

If no CA or server certificate exists, a self-signed certificate will be created. This is kept in-memory only so in order to have the client and server of these examples recognize the same certificate.

To create a self-signed CA certificate in the certs directory run: (todo)

> dist/example3 createca

To create a server certificate:

> dist/example3 createservercert

Support for lets-encrypt is planned for the future.
