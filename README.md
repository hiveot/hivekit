# HiveKit - HiveOT Development Kit

HiveKit provides building blocks for building W3C Web of Thing compatible applications using the golang language. (javascript and python modules are planned)

HiveKit provides components for serving device WoT interfaces, routing of action, event and property messages, logging of messages, a factory for constructing and reading TDs and certificate and authentication key creation and validation.

The modules in HiveKit are intended to be used to construct an application. They are not applications themselves.

HiveKit is used in HiveFlow, the IoT Concentrator; for building agents that provide a WoT interface to 3rd party IoT protocols; and in the hub to provide a digital of IoT devices and WoT enabled services.

HiveKit components were originally part of the HiveOT Hub and have been extracted to encourage use in different applications.

[Overview](docs/hivekit.png)

## Project Status

Hivekit is in alpha. It is functional but breaking changes can still be expected.
Usable modules have a status checkmark.

Transport modules:

| status | module               | description                   |
| :----: | -------------------- | ----------------------------- |
|   ✔️   | transport/direct     | simulation of transport       |
|   ✔️   | transport/discovery  | WoT mDNS device discovery     |
|   ✔️   | transport/httpbasic  | WoT HTTP basic protocol       |
|   ✔️   | transport/httpserver | HTTP server for sub protocols |
|   ✔️   | transport/ssesc      | HiveOT HTTP/SSE-SC protocol   |
|   ✔️   | transport/wss        | WoT Websocket protocol        |

Service modules:

| status | module      | description                  |
| :----: | ----------- | ---------------------------- |
|   ✔️   | authn       | authentication               |
|   ✔️   | authz       | role based authorization     |
|   ✔️   | bucketstore | key-value data storage       |
|   ✔️   | certs       | certificate management       |
|   ✔️   | digitwin    | digital twin (gateway\*)     |
|   ✔️   | directory   | Thing Directory              |
|   ✔️   | history     | message history recorder     |
|   ⬛   | jsscript    | javascript based automation  |
|   ✔️   | logging     | basic message logging        |
|   ⬛   | pipeline    | dynamic application pipeline |
|   ⬛   | router      | message routing (gateway\*)  |
|   ⬛   | rules       | rule based automation        |
|   ✔️   | vcache      | value cache (gateway\*)      |

\*) gateway modules are useful for building hub and gateway applications

Protocol Integration Binding Modules:

| status | module   | description                  |
| :----: | -------- | ---------------------------- |
|   ⬛   | ipnet    | IP Network monitor           |
|   ⬛   | isy99x   | ISY 99 protocol binding      |
|   ⬛   | owserver | 1-wire owserrver binding     |
|   ⬛   | weather  | Weather service bindings     |
|   ⬛   | zwavejs  | ZWave binding using zwave-js |
|   ⬛   | ...      | and many more...             |

## Audience

This project is aimed at software developers for building secure IoT solutions. HiveOT users support the security mandate that IoT devices should be isolated from the internet and end-users should not have direct access to IoT devices. Instead, all access operates via a secured gateway of sorts.

## About HiveOT

Security is big concern with today's IoT devices. The Internet of Things contains billions of devices that when not properly secured can be hacked too easily. Unfortunately the reality is that the security of many of these devices leaves a lot to be desired. Many devices are vulnerable to attacks and are never upgraded with security patches. This problem is only going to get worse as more IoT devices are coming to market. A botnet of a billion IoT devices can bring parts of the Internet to its knees and cripple essential services. The idea of exposing IoT devices for direct use by consumers is considered ill advised and does not meet the needs of todays reality.

To two main goals of HiveOT are:

- Aid in improving security of IoT devices by isolating them from bad actors and providing a single secure endpoint.
- Simplify integration and usage of IoT devices by providing a single consistent standardized way of interacting with all IoT devices including authentication, authorization, directory, history and other capabilities.

HiveOT is based on the [W3C WoT TD 1.1 specification](https://www.w3.org/TR/wot-thing-description11/) for interaction between IoT devices and consumers. It aims to be compatible with this standard.

Integration with 3rd party IoT protocols is supported through the use of IoT plugins. These plugins translate between the WoT protocol and 3rd party IoT protocols, interacting using properties, events and actions.
