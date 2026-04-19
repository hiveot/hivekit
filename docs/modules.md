# HiveKit Modules

This document provides a basic introduction to modules and how to create and use them.

## What is a HiveKit Module

![Module](hivekit-module.png)

A [HiveKit module](hivekit-module.png) is anything that supports the IHiveModule interface. This interface governs the interaction with the module and enables the ability to add their functionality to a chain of modules.

The types of modules are:

1. Standard module. This is the most common and easy to implement module. It MUST implement the IHiveModule interface, regardless the language it is written in. A HiveModuleBase helper is available that implements this interface and supports linking of modules. HiveModuleBase is used by most modules.

2. Transport server module. The transport server module receives messages from clients and can send messages to the client, depending on the protocol use. These modules MUST implement the ITransportServer interface. In most cases they are accompanied with a matching client module for the protocol used. A TransportServerBase implements this interface and can be used to store and retrieve incoming connections.

3. Transport client module. The transport client implements a client side protocol for passing messages to a server. These modules MUST implement the IConnection interface.
