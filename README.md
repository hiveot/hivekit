# HiveKit - HiveOT Development Kit

HiveKit is the HiveOT development kit for building lightweight IoT applications that integrates with the Web of Things. HiveKit is based on specifications from the W3C Web of Things (WoT).

HiveKit provides [*cells*](docs/cells.md) that provide desired capabilities, ranging from communication to data storage. Each cell provides a single capability following the [separation of concern principle](https://en.wikipedia.org/wiki/Separation_of_concerns). Cells can be used on their own, or linked together.

Applications are build by linking cells. Interactive cells are 'Things' that define their capabilities using a W3C Thing Description (TD) document. Cells can be linked in a chain, star, or bus *formation*. Cells can handle request messages or forward them downstream to the linked cell. Cells emit notifications for events and property updates which are forwarded to the linked upstream cell. 


## Getting Started

This project uses golang 1.25 or newer.

Make sure you understand the concepts of cells, linking, factory and recipes, as described in [cells](docs/cells.md). 

An application can be constructed by linking one or more individual cells to your application. However, the quickest way to construct an application is to use the cell factory or one of the factory recipes. The [examples](go/examples) section provides working examples of each of the recipes:

* [example1](go/examples/sadevice): build a stand-alone IoT device
* [example2](go/examples/cli): build a commandline interface (CLI)
* [example3](go/examples/tui): build a text UI 
* [example4](go/examples/gateway): build an IoT gateway
* [example5](go/examples/rcdevice): build a secure RC (reverse connection) IoT device
* [appenv](go/examples/appenv): show the application environment of cells.

Each of these examples use one of the factory recipes for constructing the example. [See the factory for details.](go/factory/README.md)




## Project Status

HiveKit is in alpha development (September 2026). 

Below a list of available cells and their status. Most cells are implemented in Golang. Javascript and Python integration is planned. 

Cells with a checkmark are functional. Breaking changes can still be expected for those marked as alpha. Cells in beta are unlikely to have breaking changes. Versioned cells remain backwards compatible.

### Core service cells and client companion

| status | cells       | description                                  | stage |
| :----: | ----------- | -------------------------------------------- | ----- |
|   ✔️    | authn       | Authentication service                       | alpha |
|   ✔️    | authz       | Role based authorization                     | alpha |
|   ✔️    | bucketstore | Key-value data storage                       | alpha |
|   ✔️    | certs       | Certificate management                       | alpha |
|   ✔️    | consumer    | Consumer and ConsumedThing                   | alpha |
|   ✔️    | digitwin    | Digital twins of Things                      | alpha |
|   ✔️    | directory   | Thing directory server & client              | alpha |
|   ✔️    | thing       | Exposed Thing base                           | alpha |
|   ✔️    | factory     | Cell factory                                 | alpha |
|   ✔️    | history     | Message history recorder                     | alpha |
|   ✔️    | logging     | Basic messaging logging                      | alpha |
|   ✔️    | reconnect   | Restore dropped client connections           | alpha |
|   ✔️    | router      | Message routing to remote devices            | alpha |
|   ✔️    | rcrouter    | Message routing to reverse-connected devices | alpha |
|   ✔️    | vcache      | Value cache                                  | alpha |
|   ⬛    | jsscript    | Javascript based automation                  | todo  |
|   ⬛    | launcher    | Launch cell recipes                          | todo  |
|   ⬛    | rules       | Rule based automation                        | todo  |

### [Message Transport Cells](docs/transport.md)

Transport cells come with a server and a client cell.

| status | cell                | description                               | stage |
| :----: | ------------------- | ----------------------------------------- | ----- |
|   ⬛    | transport/canbus    | Canbus message transport protocol         |       |
|   ✔️    | transport/discovery | WoT mDNS device discovery                 | alpha |
|   ✔️    | transport/grpc      | HiveOT gRPC fast message streaming        | alpha |
|   ✔️    | transport/httpbasic | WoT HTTP basic message transport          | alpha |
|   ⬛    | transport/lorawan   | LoRaWan message transport                 |       |
|   ⬛    | transport/mqtt      | WoT MQTT client/server transport protocol |       |
|   ✔️    | transport/tlsclient | HTTP client for sub-protocols             | alpha |
|   ✔️    | transport/tlsserver | HTTP server for sub-protocols             | alpha |
|   ✔️    | transport/ssesc     | HiveOT HTTP/SSE-SC message transport      | alpha |
|   ✔️    | transport/wss       | WoT Websocket message transport           | alpha |

### Integration Binding Cells (this will move to the HiveOT Hub)

| status | cell     | description                     | stage |
| :----: | -------- | ------------------------------- | ----- |
|   ⬛    | ipnet    | IP Network monitor              | todo  |
|   ⬛    | isy99x   | ISY 99 gateway binding          | todo  |
|   ⬛    | owserver | 1-wire owserver gateway binding | todo  |
|   ⬛    | zwavejs  | ZWave binding using zwave-js    | todo  |
|   ⬛    | weather  | Weather service bindings        | todo  |
|   ⬛    | ...      | and many more...                | todo  |


## About HiveOT

Security is big concern with today's IoT devices. The Internet of Things contains billions of devices that when not properly secured can be hacked too easily. Unfortunately the reality is that the security of many of these devices leaves a lot to be desired. Many devices are vulnerable to attacks and are never upgraded with security patches. This problem is only going to get worse as more IoT devices are coming to market. A botnet of a billion IoT devices can bring parts of the Internet to its knees and cripple essential services. The cost to businesses and consumers reaches hundreds of millions of dollars yearly.

Exposing IoT devices to the internet for direct use by consumers is therefore simply a very very bad idea from a security point of view, and does not meet the needs of todays reality. And yet, for some reason every year more and more IoT devices hit the market that run their own server and are exposed to the internet.

While HiveKit lets you build individual IoT devices that run their own server (please don't), it should be clear by now that this is, well ..., a very very bad idea.

HiveOT aims to aid in improving security of the IoT ecosystem by:

1. Not run a server on IoT devices. Instead IoT devices connect to a secured gateway or hub. These devices have the RC (reverse connection) capability which is readily supported by all HiveKit transport cells. Just swap a server cell for its client counterpart.
2. Offer an easy way to build a gateway or hub that supports RC capable devices. This is equivalent to building a server that forwards request to connected clients using the router cell.
3. Support an easy way to expand the application functionality with custom cells without having to be a security expert.
4. Support the W3C WoT standard for interacting with IoT devices including authentication, authorization, directory, history and other capabilities.
5. Define a development commitment (see below) when using HiveOT software.

HiveOT is based on the [W3C WoT TD 1.1 specification](https://www.w3.org/TR/wot-thing-description11/) for interaction between IoT devices and consumers. It aims to be compatible with this standard.

Integration with 3rd party IoT protocols is supported through the use of protocol binding cells. These cells translate between the 3rd party IoT protocols and RRN (request/response/notification) messages. The RRN messages can be linked to a WoT message transport for interaction with WoT compatible clients using properties, events and actions.

## Developer Commitment

This project is aimed at software developers for building secure IoT solutions. When adopting HiveKit, developers agree to:

1. Support the security mandate that individual IoT devices should remain isolated from the internet. See above for the motivation and rational of this critical aspect.
2. Support the use of RC (reverse connection) enabled devices that connect to a secured gateway or hub. When possible, promote this with the WoT working group when you agree to this approach.
3. Agree to regularly provide security fixes with firmware updates of your product if needed.

This probably needs a modified MIT license but that is beyond the scope of this project.

