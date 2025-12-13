# HiveOT Development Kit

HiveKit provide building blocks for constructing IoT applications.

[Overview](hivekit.png)

## Status

HiveKit is in early development (Dec 2025).

It is reworked from a library into a collection of reusable modules for easy construction of IoT applications using module pipelines. An outline of this concept is described below.

HiveKit is not an application but intended to offer the building blocks to easily construct IoT applications. It is used in the [HiveFlow](https://github.com/hiveot/hiveflow) and [HiveHub](https://github.com/hiveot/hub) applications. (currently being reworked)

## HiveKit Modules

A HiveKit module can be anything that implements the IHiveKit module interface. Most modules support communication using the HiveOT standard messaging envelope (SME). Each module is optionally also a WoT Thing that can be configured and controlled using messages.

Examples of modules:

- directory, history, authentication services.
- client side of a service
- a message processor such as a router, filter, logger, etc
- protocol server such as WoT HTTP, WSS, CoAP, MQTT
- protocol client adapter for a server - adapts to standard messaging envelope
- an IoT adapter such as ZWave, Insteon, CoAP, Zigbee, Shelley and so on.

Modules consists of an api, core, factory and tests.

The module api defines the native API of the module along with optional protocol specific API definitions. For example a WoT TM (Thing Model), OpenAPI definition, and protobuf definitions.

The module api includes an adapter that translates between the standardized messaging format (SME - Standard Messaging Envelope) and native API. The standardized messaging envelope is used to communicate between modules regardless where they are located.

Where required, the module's api includes adapters for other protocols such as openapi, or protobuf, supporting direct interfacing using these protocols without translation to SME messages.

The module core implements the module's logic and can be used as-is without HiveOT specific dependencies. The core must implement the native api as defined in the module api section.

The module factory provides a standard way to create and use modules through provided configuration. It provides a factory function to create module instances from configuration. These instances support SME messages for communication between modules to help construct module pipelines.

The module tests contains test methods to verify the correct behavior of the module.

## The IoT pipeline

A pipeline consists of chains of modules, called a recipe, that are linked using their SME message interface. It is intended to simplify building and testing WoT compatible IoT devices and to provide reusable capabilities for processsing IoT device information.

While HiveKit comes with a set of modules, 3rd party modules can be incorporated easily as part of a recipe.

A HiveOT module receives messages, processes them and passes them on to the next module in the pipeline. Modules are supposed to have a single responsibility, keeping them lightweight and simple. More complex modules can be created linking them into pipelines. HiveOT modules are capable of sending and receiving both data and control message streams between pipeline modules.

Modules have one or more message inputs and one or more message outputs. A module that generates messages is called a source. A module that consumes messages is a sink. Service modules often provide both source and sink capabilities.

Types of streams in the pipeline:

- A unidirectional data stream from source to sink, passing events and property updates. This supports fan-out with multiple receivers for data.

- A bidirectional control stream for actions and property configuration. Control streams are point-to-point between source and sink but can be relayed and processed by other modules.

Module linking can connect in-process or out-of-process modules. In-process modules communicate using the efficient SME messages. Modules running on different processes or devices use messaging modules that translate the SME (internal) message format to a specific protocol such as WoT Websocket, MQTT, CoAP or other supported messaging protocol.

Modules can be linked in several ways:

1. At compile time by adding a sink onto a source. Chain multiple modules to create a pipeline. In golang this creates a single binary that runs stand-alone.
2. Using the pipeline runtime module to link modules based on the provided yaml recipe. The pipeline recipe can be changed without the need to recompile. This is limited to the use of registered modules available in the pipeline runtime.
3. By compiling a pipeline using the provided yaml recipe using the hiveflow CLI. In golang this creates a single binary that has its recipe and required modules embedded.
4. Using messaging modules the pipeline can be linked to other pipelines running elsewhere.

Testing is facilitated by providing a testing runtime with tools to generate devices, consumers and messages. Modules can be tested in isolation or as part of a pipeline.

### HiveOT Module API

All modules support the HiveOT module API defined as IHiveModule. This API supports both the data stream and the control stream. Depending on the programming language static or dynamic linking can be used to implement the callback hooks.

```go
// The golang HiveOT module interface  (proposed)
type IHiveModule interface {
   // GetTM returns the module's [W3C WoT Thing Model](https://www.w3.org/TR/wot-thing-description11/#thing-model) in JSON describing its properties, actions and events. This is primarily intended for use by the pipeline runtime that converts it to a [TD](https://www.w3.org/TR/wot-thing-description11/).
   // The TM can be obtained after a successful start. If the module does not support a TM then this returns an empty string.
   GetTM() string

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

## Adding Modules

One of the goals of HiveKit is to make it easy to add compatible modules.

This is accomplished by:

1. Support embedded modules through a simple module API for supported languages. Golang modules are embedded at build time, javascript and python modules can be embedded at build or runtime.
2. Support stand-alone modules through a messaging interface. The messaging interface can be a WoT compliant messaging interface or use a high-efficiency IPC transport such as gRPC, unix pipes or others. The messages it carries are the HiveOT standardized request, response and notification message envelopes or one of the WoT message formats such as the websocket messages.
3. Modules are described using WoT TMs (Thing Models). Each module is also a WoT Thing and can process messages forwarded to it as part of the pipeline.
4. Complex modules can be configured through WoT properties as described in its TM.

To develop a module, implement its IHiveModule interface and hook it up as the sink of the previous module, or the messaging server module in case of standalone operation. Received messages are passed to the HandleRequest/HandleNotification methods and responses are returned to the sender, which passes it further down the pipeline.

To pass output of the module to the next in the pipeline use its AddSink method.
The pipeline runtime module can do this automatically from a provided configuration.
