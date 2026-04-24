# directory - Things Directory Module

This directory module provides the means to store and retrieve TD (Thing Description) documents.

The primary objective is to let consumers discover what Things are available and obtain the information on how to access them.

This module follows the WoT discovery specification https://w3c.github.io/wot-discovery/#exploration-directory and a subset of the TM described at https://w3c.github.io/wot-discovery/#directory-api-spec.

This module is not a full blow stand-alone application but simply offers the directory capabilities to applications or services that want to include a directory. It must be linked to a server transport module to receive the requests.

## Status

This module is in alpha. It is functional but breaking changes might still happen.

There are two notable issues for which there is no standardization:
1: For security reasons, a device TD should only be updatable by the owning agent of a device. How to determine who this agent is?
2: How to prevent thingID collisions? There is no mechanism to guarantee uniquenes between devices. One option is to use UUIDs. Another is to use namespaces in the ID.

Current solution: HiveOT uses the convention that thingIDs contain the agentID prefix separated by a colon. The format for thingID is: "{agentID}:{deviceID}", where {deviceID} is the ID of the device unique within the scope of the agent publishing the TD.

If the TD is to be published in an internet based directory, the agentID must be globally unique and the forms must be updated to externally reachable addresses. In HiveOT this is not a concern of devices. Instead a gateway module must handle external exposure and security.

## Summary

The WoT discovery specification defines the directory service API for storing and retrieving TD information. This module exports a TM that matches the description provided in the specification.

The directory package contains these modules: the directory service, its http API server, a messaging client, and an HTTP client. These can be used as any other module, and operate client side or server side. Typically, the directory server is linked to a transport server to receive requests and publish notifications. Similarly the directory client module can be used by applications to query the TDs of the available Things.

The directory should be updated by IoT devices or their agent. In HiveOT, the convention is that Thing agents update the directory with one or more TD's of the Things it manages.

Alternatively, an administrator can update the directory manually with a JSON document using the provided CLI. The CLI is a simple example commandline interface that uses the directory client to read and write the directory.

To write their TD to the directory storage, IoT device agents need to discover the location of the directory and invoke the createThing action, providing the TD JSON document as the payload.

## Backends

This module internally uses a Key-Value bucket store for persisting TD documents. When read, TDs are cached in memory for fast access by consumers.

TBD: maybe use a filesystem based backend where TDs are stored? It would make importing TDs out-of-band easier.

## Usage

### Creating a Directory

Examples of creating an instance of the directory.

1. Use the HiveKit module factory. This factory provides the application environment and automatically creates instances of the neccesary modules.

2. Manually without the HTTP API: **[server]->[directory]**
   1. Create an instance of the module using directorypkg.NewDirectoryService() and provide it with the storage location of the embedded database.

   2. Call Start on the service. This will initialize and create the store.

   3. Link it as a sink of a server module chain. Any directory requests will be handled by the module. Modules can be chained and it doesn't matter where in the chain the directory module reside.

   4. Before shutdown call Stop() to ensure the datastore is properly closed.

3. Manually with the HTTP API: **[server]->[http-api]->[directory]**
   1. Create an instance of the directory http API server using directorypkg.NewDirectoryHttpHandler()

   2. Set the HTTP API module as the sink of the server module chain.

   3. Create an instance of the directory service using directorypkg.NewDirectoryService() and provide it with the storage location of the embedded database, and the http api. The http api is used to set the base URL, security and forms of the directory TD.

   4. Set the service module as the sink of the http api module. Request received via the directory http API is now handled by the server.

   5. Call Start on both the http api and the service(). This will initialize or create the store and register the HTTP endpoints with the HTTP server module.

   6. Before shutdown call Stop() to ensure the datastore is properly closed.

### Updating the directory

There are several use-cases for updating the directory with TD from Things. At this moment it isn't clear if there is a preferred way. HiveOT is leaning towards option 3 and 4 as HiveOT devices do not run servers.

1. Stand-alone devices can use discovery to publish their TD or TDD in case there are multiple. Someone need to get the TD and add it to the directory. Who?
2. A stand-alone device discovers a directory and write its TD to it.
3. Devices with reverse connection to a hub or gateway can write the TDs they manage to the its directory.
4. An administrator can manually upload TDs to the directory import location. This is not yet supported (but seems like a good idea)
