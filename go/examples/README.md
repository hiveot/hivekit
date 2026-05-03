# examples

These examples demonstrate how to build an ecosystem of IoT devices and services using HiveKit. The examples can be used on their own or together.

To build run "make examples". Binaries are created in the ./dist directory.

## Simple Examples

These first few examples are kept simple on purpose. They use a single protocol and lacks authentication, authorization and offers no history.

### Example 1: Run a Standalone Counter Device

Example 1 creates a standalone IoT device that runs a simple counter. It has a property with the current value, sends an event when it changes and has actions for increment and decrement.

- This uses a factory recipe to create a server and link it to the counter module.
- This publishes an event each time the counter value changes.
- This offers actions for incrementing and decrementing the counter.
- This serves Thing discovery of the module TD and can be discovered with example 2.

usage: go run example1/main.go

### Example 2. Discovery

A simple commandline utility to discover Things and Directories on the network and optionally show their TD. Use -h to view available filter and display options.

usage: go run example2/main.go [-h] [-td] [-txt] [-type=Thing|Directory] [-addr=192.168.x.y]

### Example 3. CLI

The commandline interface interacts with discovered devices. Show properties, latest event and invoke actions (only those with a single input).

usage: go run example3/main.go

This displays a menu with options. Commands:

- discover devices
- read directory

### Example 4. Gateway

The gateway runs a server that both devices and consumers connect to. It includes a discovery server, a directory with discovered and registered devices and a router to forward requests from consumer to standandalone and RC devices.

### Example 5. RC Device (reverse connection)

This example constructs a RC device that uses a reverse connection to a gateway. It contains a test device and a client for a gateway.

## Usage

Simply start ./dist/example1 to run it. Press Ctrl-C to terminate.

To view the commandline options:

> ./dist/example1 -h

HiveKit looks for certificates and keys in the {home}/certs directory which defaults to ~/bin/hiveot/certs.
A different home directory can be passed using -home=/path/to/my/home.

If no CA or server certificate exists, a self-signed certificate will be created. This is kept in-memory only so in order to have the client and server of these examples recognize the same certificate.

To create a self-signed CA certificate in the certs directory run:

> dist/example2 createca

To create a server certificate:

> dist/example2 createservercert

Support for lets-encrypt is planned for the future.
