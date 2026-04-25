# examples

These examples demonstrate how to build an ecosystem of IoT devices and services using HiveKit. The examples can be used on their own or together.

## Simple Examples

These first few examples are kept simple on purpose. They use a single protocol and lacks authentication, authorization and offers no history.

1. Standalone Device

The standalone device runs a server and defines a module for a simple counter device. It contains a property with the counter value that increases, sends an event each time the counter changes and has an action to set the counter. The counter device module is also used in the rcdevice example.

2. CLI

The commandline interface lets you discover and query discovered devices on the network. It works with standalone Things and with Things connected to a gateway.

3. Gateway

The gateway runs a server that both devices and consumers connect to. It includes a discovery server, a directory with discovered and registered devices and a router to forward requests from consumer to standandalone and RC devices.

4. RC Device (reverse connection)

This example constructs a RC device that uses a reverse connection to a gateway. It contains a test device and a client for a gateway.
