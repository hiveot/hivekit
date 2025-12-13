# HiveKit Examples

This is under heavy development and will be offering concrete examples on building IoT applications using HiveKit and 3rd party modules.

We're not there yet so for now this explains the concepts.

## Concepts and Usage Examples

### Constructing a basic IoT Sensor or Actuator

Most IoT devices have in common that they contain logic to read their current state and update writable state such as configuration or actuator value.

Connected devices also have the ability to export their state or import new state,through a web interface or some kind of messaging protocol.

Most devices will have some form of authentication built-in to ensure only people with the sufficient permissions can read and modify the device.

Sometimes there is a logging capability to assist users in troubleshooting.

These steps are common across devices and, except for the first step, can be standardized as reusable modules:

1. read/write hardware - this is device specific
2. run a server to receive connections and messages
3. authenticate the client initiating the connection
4. decode protocol messages into the internal format for further processing
5. handle requests to read the current state (read properties)
6. handle requests to update configuration (write properties)
7. handle requests to invoke actuators (invoke action)
8. notify of changes (consumer subscription of events and properties)
9. log requests to a configured destination

WoT capable modules can export a TD describing its capabilities, although this can also be defined out of band.

Other potential capabilities of IoT devices are authorization, rate control, connection reversal, publishing a TD in a discovered directory, and more.

IoT devices can be simplified by using connection reversal. The device connects to a gateway or hub instead of the other way around. Once connected the gateway subscribes to updates and passes requests from consumers to the device. There is no need to run a server on the device, nor to manage clients, which greatly improves security and reduces required resources at the same time. The device management shifts to the hub which provides a consistent interface for all the devices it manages.

### Constructing a Consumer

Consumers in the form of a commandline or web client can simply invoke a sensor's API using the WoT protocol to retrieve status or invoke commands. They can also be much more complex like for example the Home Assistant system.

Common steps simple consumers take:

1. Obtain authentication credentials. This is typically handled out of band.
2. Obtain the TD of the desired device. Usually some kind of directory is involved.
3. Connect to the device using the supported protocol as described by the device TD.
4. Subscribe to events and properties of interest.
5. Receive updates.
6. Process update or present it to the user.

Having multiple devices each running their own server has several drawbacks for consumers. Each new device opens the attack surface of the network. The weakest link can compromise it. This is aggravated when allowing connections over the internet. Other drawbacks are the need to manage the login accounts for each device and deal with the differences between manufacturers. To make things worse, there is a multitude of incompatible IoT protocols each with different controllers and capabilities.

While the W3C WoT (Web of Things) is working on providing a common standard for interacting with devices, it is not widely supported. Use of a gateway that translates between native protocols and the WoT standard simplifies the use of multiple devices significantly. HiveKit can be used to construct such a gateway.

To construct a simple consumer of WoT devices the following recipe can be used.

[presentation] -> [directory client]
[presentation] -> [multi-client] => [device client]

(\*the concept and notation of recipies to construct pipelines is still in development)

The modules used here are:

- [presentation]: There are various presentation modules, ranging from a simple commandline interface to web components to full blown web or desktop applications. This is an entire topic on its own. HiveKit supports several basic presentation modules that can be used separately or concurrently. Presentation modules can retrieve device TD's (Thing Description document) from a directory and connect to selected devices using their TD to present and control devices.
- [directory client]: Consumers can select from a list of available devices provided by a directory client. The directory client module obtains these from a discovered directory service.
- [multi-client]: The multi-client module provides the capability to establish multiple connections to one or more devices. It can also include the capability to re-use an existing connection if multiple devices can be reached through that one connection. This module is useful when presentation connects to more than one device.
- [device client]: The client module connects to a device to read status and to subscribe to updates. The HiveKit WoT client module supports multiple protocols to connect with.

### Constructing a Gateway

A gateway is useful when using multiple IoT devices of different protocols, when devices are hidden on their own subnet, or simply when a single endpoint is desired that provides its own directory. Such a gateway can be constructed with modules from this Kit.

A typical gateway implements the following steps:

1. Discover devices on the network. This can be done manually or automatically.
2. Update the device TD with gateway endpoints.
3. Add discovered devices to a directory. The directory can be part of the gateway or operate externally.
4. Publish gateway discovery by consumers.
5. Serve incoming connections from consumers.
6. Authenticate consumers
7. Track server connected consumers to be able to forward notification messages.
8. Serve directory requests from consumers.
9. Use a gateway client to connect to devices whose information is requested.
10. Forward event and property subscription requests to devices.
11. Notify consumers of event and property updates they subscribed to.
12. Forward action requests to devices and return responses.
13. Log requests to a configured logger.
14. Drop device connections that are no longer needed.

Although not part of the WoT standard, the use of reverse connections by supporting devices closes a big attack vectors as devices can no longer be access directly. This is an approach favored by HiveOT and supported through HiveKit modules. For gateways that support reverse connections:

1. Serve incoming connections from devices.
2. Track connected devices in order to forward consumer requests.

Optionally, to enhance the functionality:

1. Authorize requests based on consumer roles.
2. Store device property, event and/or action history.

Modules are available for storing most of these steps.

The recipe for this setup could look like:

```
[gateway]
  |- [directory store]
  |      |- [directory server] -> [directory TD]
  |      |- discovery server
  |
  |- [WoT websocket server] -> [server connections]
  |                                 |- [authenticator]
  |                                 |- [logger]
  |
  |- [server connection] -> messages
  |    |- [authorizer]
  |    |- [router]
  |          |- [multiclient] - [device client]
  |          |- directory server
```

Where,

- the message router uses the directory store to determine request destinations and create client connections to the devices using a device client.
- requests to read the directory are routed to the directory server
- requests for clients are passed to the multi-client which establishes client connections to connect to devices

### Constructing a Digital Twin Hub

A digital twin hub takes the gateway to the next level. Instead of consumers interacting with devices via a gateway, they interact directly with a digital twin that represents the device. This:

- makes device status available even if they are asleep or temporary offline.
- greatly reduces the traffic with devices themselves as all read operations are handled through the digital twin. This in turn reduces latency and increases consumer performance.
- supports device simulation as part of a test program.
- improves security as the device location and access remains hiddden from the consumer.
- opens the option to substitute or replace devices without affecting consumers.

The extra modules for the digital twin include:

- an internal directory storage module that contains the digital twins TDs. Device TD's are converted to the digital twin equivalent.
- a value store containing the last known property, event and action values as reported by devices.
- an updated router that forwards read/query request to the value store for digital twin devices
- an action store that tracks action progress.
- optionally a simulator module that intercepts messages for specific devices.
