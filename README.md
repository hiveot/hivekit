# HiveKit - HiveOT Development Kit

HiveKit provides modules for building lightweight IoT applications that can interact with the Web of Things.

Modules included are WoT transport protocols, authentication, authorization, message routing, thing discovery, thing directory, data storage, history storage, digital twin, and more.

HiveKit modules interact using _RRN_ Request-Response and publish-subscribe Notification messages. HiveKit combines the strengths of these two messaging patterns into a simple and easy to use messaging framework for module integration. RRN messages define an envelope that describes an operation, the Thing to address, the name of the message, and its payload, as described in the [W3C WoT standard](https://www.w3.org/TR/wot-thing-description11/).

The modules in HiveKit are intended to be used to construct an application. They are not applications themselves.

The 'pipeline' module provides a quick and easy way to construct an application using recipes. A recipe describes the modules to be used and how they are linked together. Custom modules can be included to provide the custom functionality of the application.

HiveKit components were originally part of the HiveOT Hub and have been extracted to facilitate reuse in different applications.

[Overview](docs/hivekit.png)

## Project Status

(updated march 2026)

Hivekit is in alpha. Modules with a checkmark are functional but breaking changes can still be expected.

Transport modules:

| status | module               | description                   | stage |
| :----: | -------------------- | ----------------------------- | ----- |
|   ✔️   | transport/direct     | Simulation of transport       | alpha |
|   ✔️   | transport/discovery  | WoT mDNS device discovery     | alpha |
|   ✔️   | transport/grpc       | HiveOT gRPC protocol          | alpha |
|   ✔️   | transport/httpbasic  | WoT HTTP basic protocol       | alpha |
|   ✔️   | transport/httpserver | HTTP server for sub protocols | alpha |
|   ✔️   | transport/ssesc      | HiveOT HTTP/SSE-SC protocol   | alpha |
|   ✔️   | transport/wss        | WoT Websocket protocol        | alpha |
|   ⬛   | transport/coap       | WoT CoAP protocol             | n/a   |
|   ⬛   | transport/mqtt       | WoT MQTT protocol             | n/a   |

Core Service modules:

| status | module      | description                 | stage |
| :----: | ----------- | --------------------------- | ----- |
|   ✔️   | authn       | Client authentication       | alpha |
|   ✔️   | authz       | Role based authorization    | alpha |
|   ✔️   | bucketstore | Key-value data storage      | alpha |
|   ✔️   | certs       | Certificate management      | alpha |
|   ✔️   | digitwin    | Digital twin                | alpha |
|   ✔️   | directory   | Thing directory             | alpha |
|   ✔️   | history     | Message history recorder    | alpha |
|   ✔️   | logging     | Basic messaging logging     | alpha |
|   ✔️   | router      | Message routing to devices  | alpha |
|   ✔️   | vcache      | Value cache                 | alpha |
|   ⬛   | pipeline    | Dynamic module pipeline     | dev   |
|   ⬛   | jsscript    | Javascript based automation | todo  |
|   ⬛   | rules       | Rule based automation       | todo  |

Integration Binding Modules:

| status | module   | description                     | stage |
| :----: | -------- | ------------------------------- | ----- |
|   ⬛   | ipnet    | IP Network monitor              | todo  |
|   ⬛   | isy99x   | ISY 99 gateway binding          | todo  |
|   ⬛   | owserver | 1-wire owserver gateway binding | todo  |
|   ⬛   | zwavejs  | ZWave binding using zwave-js    | todo  |
|   ⬛   | weather  | Weather service bindings        | todo  |
|   ⬛   | lorawan  | LoRaWan gateway binding         | todo  |
|   ⬛   | canbus   | Canbus gateway binding          | todo  |
|   ⬛   | ...      | and many more...                | todo  |

## About HiveOT

Security is big concern with today's IoT devices. The Internet of Things contains billions of devices that when not properly secured can be hacked too easily. Unfortunately the reality is that the security of many of these devices leaves a lot to be desired. Many devices are vulnerable to attacks and are never upgraded with security patches. This problem is only going to get worse as more IoT devices are coming to market. A botnet of a billion IoT devices can bring parts of the Internet to its knees and cripple essential services. The cost to businesses and consumers reaches hundreds of millions of dollars yearly.

Exposing IoT devices to the internet for direct use by consumers is therefore simply a very very bad idea from a security point of view, and does not meet the needs of todays reality. And yet, for some reason every year more and more IoT devices hit the market that run their own server and are exposed to the internet.

While HiveKit lets you build individual IoT devices that run their own server (please don't), it should be clear by now that this is, well ..., a very very bad idea.

HiveOT aims to aid in improving security of the IoT ecosystem by:

1. Not run a server on IoT devices. Instead IoT devices connect to a secured gateway or hub. These devices have the RC (reverse connection) capability which is readily supported using HiveKit.
2. Offer an easy way to build a gateway or hub that supports RC capable devices.
3. Support an easy way to expand the application functionality with custom modules without having to be a security expert.
4. Support the W3C WoT standard for interacting with IoT devices including authentication, authorization, directory, history and other capabilities.
5. Define a development commitment (see below) when using HiveOT software.

HiveOT is based on the [W3C WoT TD 1.1 specification](https://www.w3.org/TR/wot-thing-description11/) for interaction between IoT devices and consumers. It aims to be compatible with this standard.

Integration with 3rd party IoT protocols is supported through the use of protocol binding modules. These modules translate between the 3rd party IoT protocols and RRN messages. The RRN messages can be linked to a WoT protocol for interaction with WoT compatible clients using properties, events and actions.

## Developer Commitment

This project is aimed at software developers for building secure IoT solutions. When adopting HiveKit, developers agree to:

1. Support the security mandate that individual IoT devices should remain isolated from the internet. See below for the motivation and rational of this critical aspect.
2. Support the use of RC (reverse connection) enabled devices that connect to a secured gateway or hub.
3. Agree to regularly provide security fixes with firmware updates if needed.

This probably needs a modified MIT license but that is beyond the scope of this project.

## Getting Started

### Build

This project uses golang 1.25 or newer.

To debug with vscode delve must be installed. To get the latest (on linux):

> go install github.com/go-delve/delve/cmd/dlv
> export $PATH=$PATH:~/go/bin
> go mod tidy

### Use

The easiest way to get started is to use one of the predefined recipes of the pipeline module. There are recipes for constructing stand-alone IoT devices, a WoT compatible gateway, a digital twin hub, and client applications.

... this section is under development...
