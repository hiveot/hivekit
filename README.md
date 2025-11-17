# HiveOT gocore

The HiveOT go-core provides building blocks for building Web of Thing applications using the golang language.

It is used in go2wot to construct a WoT compatible IoT concentrator, in agents that convert 3rd party IoT protocols to a WoT compatible interface, and in the HiveOT hub to provide WoT enabled services.

Gocore components were originally part of the HiveOT Hub and have been extracted to encourage use by 3rd parties.

## Project Status

Nov 2025: Kickoff, documentation and extract components from the hiveot hub.

## Audience

This project is aimed at software developers for building secure IoT solutions. HiveOT users support the security mandate that IoT devices should be isolated from the internet and end-users should not have direct access to IoT devices. Instead, all access operates via the Hub.

## What Is The Hiveot Gocore

The objective of the hiveot gocore is to simplify building IoT components and solution that are compatible with the W3C WoT (Web of Things) standard and protocols.

The gocore provides server components for serving WoT compatible interfaces, client building blocks for connecting to WoT devices and directories, a message router for handling actions, events, properties and TDs.

## About HiveOT

The gocore is part of HiveOT.

Security is big concern with today's IoT devices. The Internet of Things contains billions of devices that when not properly secured can be hacked. Unfortunately the reality is that the security of many of these devices leaves a lot to be desired. Many devices are vulnerable to attacks and are never upgraded with security patches. This problem is only going to get worse as more IoT devices are coming to market. Imagine a botnet of a billion devices on the Internet ready for use by unscrupulous actors.

The goal of HiveOT is to support a way to improve security of IoT devices, mainly by isolating them from the rest of the network and providing a single secure endpoint.

The secondary goal of HiveOT is to simplify integration and usage of IoT devices by providing a single consist standardized way of interacting with all IoT devices including authentication, authorization, directory, history and other capabilities.

HiveOT is based on the [W3C WoT TD 1.1 specification](https://www.w3.org/TR/wot-thing-description11/) for interaction between IoT devices and consumers. It aims to be compatible with this standard.

Integration with 3rd party IoT devices is supported through the use of IoT plugins. These plugins translate between the WoT protocol and 3rd party IoT protocols, interacting using properties, events and actions.

The HiveOT Hub uses goore and go2wot as its runtime and adds a digital twin model, service launcher, history, dashboard, and other features such as bridging in a single package.
