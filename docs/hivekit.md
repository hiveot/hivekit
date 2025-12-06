# HiveOT Development Kit

HiveKit provide building blocks for constructing IoT applications.

[Overview](hivekit.png)

## Status

HiveKit is currently an operational library (Nov 2025).

It is reworked into reusable modules for easy construction of IoT applications by creating a pipeline these modules. An outline of this concept is described below.

## The IoT pipeline

A pipeline consists of a chain of modules, called a recipe, that each operate on a message. It is intended to simplify building WoT compatible IoT devices and to provide capabilities for processsing IoT device information.

While HiveKit comes with a set of modules, 3rd party modules can be incorporated as part of a recipe.

A pipeline module receives messages, processes them and passes them on to the next module in the pipeline. Modules have a single responsibility. More complex modules can be created linking them into a pipeline. Pipeline modules are capable of sending and receiving both data and control message streams between pipeline modules.

Modules have one or more message inputs and one or more message outputs. A module that generates messages is called a source. A module that consumes messages is a sink. Service modules often provide both source and sink capabilities.

Types of streams in the pipeline:

- A unidirectional data stream from source to sink, passing events and property updates.

- A bidirectional control stream for actions and property configuration.

Pipelines are created by linking one or more modules. Modules can run in-process, or by using messaging modules, a pipeline can be extended to out-of-process modules and other pipelines, including those running on other hosts.

Modules can be linked in several ways:

1. At compile time by adding a sink onto a source. Chain multiple modules to create a pipeline. In golang this creates a single binary that runs stand-alone.
2. Using the pipeline runtime module to link modules based on the provided yaml configuration.The pipeline configuration can be changed without the need to recompile it. This is limited to registered modules available in the pipeline runtime.
3. By compiling a pipeline using the provided yaml configuration using the hiveflow CLI. In golang this creates a single binary that has its configuration embedded.
4. Using messaging modules the pipeline can be linked to other pipelines running elsewhere.

Testing is facilitated by providing a testing runtime with tools to generate devices, consumers and messages. Modules can be tested in isolation or as part of a pipeline.

### Pipeline Module API

All modules support the pipeline module API defined as IHiveModule. This API supports both the data stream and the control stream. Depending on the programming language static or dynamic linking can be used to implement the callback hooks.

```go
// The golang pipeline module interface  (proposed)
type IHiveModule interface {
   // GetTM returns the module's TM describing its properties, actions and events.
   // If supported, the TM can be obtained after a successful start. If no TM is supported then this returns nil.
   // The runtime converts the TM into a TD by adding the forms needed to interface with the module, as determined by the available protocols.
   // For example, the Directory module provides the directory TM for use in discovery.
   GetTM() *TD

   // HandleRequest processes or forwards a request message. This returns a response
   // containing the delivery status and optionally a result.
   HandleRequest(request *RequestMessage) *ResponseMessage

   // HandleNotification processes or forwards a notification message.
   // Notification messages consists of subscribed events and observed properties.
   HandleNotification(*NotificationMessage)

   // AddSink sets the destination sink to forward messages to, to send the processing result to, or both.
   // Modules can support a single or multiple sinks. If no more sinks can be added an error is returned.
   // AddSink can be invoked before or after start is called.
   AddSink(sink IHiveModule) error

   // Start readies the module for use using the given yaml configuration.
   // Start must be invoked before passing messages.
   Start(yamlConfig string) error
   Stop()
}
```

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

## Adding Modules

One of the goals of HiveKit is to make it easy to add modules compatible with other kit modules. Each module is also a WoT Thing and can optionally be configured and controlled using WoT messages.

This is accomplished by:

1. Support embedded modules through a simple module API for supported languages. Golang modules are embedded at build time, javascript and python modules can be embedded at build or runtime.
2. Support stand-alone modules through a messaging interface. The messaging interface can be a WoT compliant messaging interface or use a high-efficiency IPC transport such as gRPC, unix pipes or others. The messages it carries are the HiveOT standardized request, response and notification message envelopes or one of the WoT message formats such as the websocket messages.
3. Modules are described using WoT TMs (Thing Models). Each module is also a WoT Thing and can process messages forwarded to it as part of the pipeline.
4. Complex modules can be configured through WoT properties as described in its TM.

To develop a module, implement its IHiveModule interface and hook it up as the sink of the previous module, or the messaging server module in case of standalone operation. Received messages are passed to the HandleRequest/HandleNotification methods and responses are returned to the sender, which passes it further down the pipeline.

To pass output of the module to the next in the pipeline use its AddSink method.
The pipeline runtime module can do this automatically from a provided configuration.
