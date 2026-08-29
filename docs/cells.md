# HiveKit Cell Basics

HiveKit cells are building blocks for building devices and applications. Cells follow the separation of concerns paradigm where each cell is performs a single task. Applications are build by linking cells. 

The standard cell has a simple interface: A handler for request messages with a replyTo callback, and a handler for notification messages. Cells are linked by setting a request sink to the next cell in the chain. Similarly a notification sink is set to the upstream cell.

[![cell](hivekit-cell.png)](#hivekit-cells)

Cell interaction takes place using RRN (request-response-notification) messages. These contain a WoT operation, Thing-ID, affordance name and optional input and output payloads.

Each cell has an cell instance ID that can be used as a thing-ID. Where applicable, their capabilities can be described in a WoT TD (Thing Description) document that describes its properties, events and actions. 

HiveKit cells interact using _RRN_ Request-Response and publish-subscribe Notification messages. HiveKit combines the strengths of these two messaging patterns into a simple and easy to use messaging system for connecting cells. RRN messages define an envelope that describes a WoT operation, the Thing to address, the name of the message, and its payload, as described in the [W3C WoT Thing Description](https://www.w3.org/TR/wot-thing-description11/).

## Cell Types

The following cell categories can be distinguished:

1. Service cells are producers that offer a service, such as authentication, logging and routing. Service cells can be configured through properties and queried using actions. Services also act as a consumer when they collect and process information.  

The ExposedThing cell helps writing devices and services. It provides methods for publishing notifications, tracking property state and handle requests to read properties.

2. Middleware cells are a class of cells whose purpose is to analyze, filter and route messages. For example, logging, authorizing, routing are middleware tasks. 

3. Transport cells role is to link cells over the network. A transport client cell connects to a corresponding transport server cell. Request, response and notification messages are send between client and server cells as defined by the transport protocol. Client-Server cell pairs are available for multiple protocols such as http-basic, websockets, gRPC and others. Server cells track event subscriptions and subscriptions to observe properties made via the client.

4. Consumer cells collect information from producers. Consumers publish requests for information and receive responses and notifications. 
   
The 'Consumer' cell implementation helps writing consumers by providing methods for publishing requests and subscribing to event and property notifications.

## Linking Cells

A core capability of cells is the ability to chain them together. Chains offer application level functionality. A chain can operate on a single computer system or include cells across multiple computer systems linked by transport cells. This allows for creating a powerful distributed IoT solution with small lightweight cells that require few resources and are simple to maintain.

Creating a cell chain can be done manually by programatically linking cells, or dynamically by providing a recipe to the factory service. 

## Cell Factory

Cells in HiveKit are not applications themselves but intended to construct an application. The [factory cell](go/cells/factory/README.md) facilitates building applications by chaining cells defined in a recipe. This chaining aggregates functionality provided by each cell. 

Application specific logic can easily be incorporated using the hooks provided by the exposed-thing cell, or by providing application logic as a cell itself and adding this cell to the recipe.

![cell](docs/cell-chain.png)


## Adding Cells

One of the goals of HiveKit is to make it easy to add compatible cells.

To develop a cell implement its IHiveCell interface. The provided CellBase implements the little boilerplate that is needed. The HandleRequest method is the most important method to implement. Exposing a TM is recommended for IoT devices.

To use a cell connect it as the sink of the previous cell in the chain. In case of an IoT device the previous cell can be one of the messaging server cells. The server passes requests to the HandleRequest method which the cell must implement, and responses are returned to the sender. Notifications emitted by the cell are passed to the registered notification handler which is the server cell.
