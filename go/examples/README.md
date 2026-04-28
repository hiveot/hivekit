# examples

These examples demonstrate how to build an ecosystem of IoT devices and services using HiveKit. The examples can be used on their own or together.

To build simple run "make examples". Binaries are created in the ./dist directory.

## Simple Examples

These first few examples are kept simple on purpose. They use a single protocol and lacks authentication, authorization and offers no history.

### Example 1: Standalone Device

The standalone device creates a standalone IoT device that runs a simple counter. It has a property with the current value, sends an event when it changes and has actions for increment and decrement.

This uses a factory recipe to create a server and link it to the counter module.
The counter can be queried with the CLI from example 2.

### Example 2. CLI

The commandline interface lets you discover and query discovered devices on the network. It works with standalone Things and with Things connected to a gateway.

### Example 3. Gateway

The gateway runs a server that both devices and consumers connect to. It includes a discovery server, a directory with discovered and registered devices and a router to forward requests from consumer to standandalone and RC devices.

### Example 4. RC Device (reverse connection)

This example constructs a RC device that uses a reverse connection to a gateway. It contains a test device and a client for a gateway.
