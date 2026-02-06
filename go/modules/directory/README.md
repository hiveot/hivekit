# Directory Module

This directory module provides the means to store and retrieve TD (Thing Description) documents.

The primary objective is to let consumers discover what Things are available and obtain the information on how to access them.

This directly follows the WoT discovery specification https://w3c.github.io/wot-discovery/#exploration-directory.

This module is not a full blow stand-alone application but simply offers the directory capabilities applications or services that want to include a directory. It must be linked to a server module to receive the requests.

## Status

This module is in alpha. It is functional but breaking changes might still happen.

There are two notable issues for which there is no standardization:
1: For security reasons, a TD should only be writable by the owning agent. How to determine who this agent is?
2: How to prevent thingID collisions? There is no mechanism to guarantee uniquenes between devices.

Solution: HiveOT requires that thingIDs contain the agentID prefix separated by a colon. The format for thingID is: "{agentID}:{deviceID}", where {deviceID} is the ID of the device unique within the scope of the agent publishing the TD.

If the TD is to be published in an internet based directory, the agentID must be globally unique and the forms must be updated to externally reachable addresses. In HiveOT this is not a concern of devices. Instead a gateway module must handle external exposure and security.

## Summary

The WoT discovery specification defines the directory service API for storing and retrieving TD information. This module exports a TM that matches the description provided in the specification.

The directory package contains a service and a client module. Both can be used as any other module, and operate client side or server side. The directory server module can be linked to a transport module to receive requests and publish notifications. Similarly the directory client module can be used by applications to query the TDs of the available Things.

The directory should be updated by IoT devices or their agent. In HiveOT, the convention is that Thing agents update the directory with one or more TD's of the Things it manages.

Alternatively, an administrator can update the directory manually with a JSON document using the provided CLI. The CLI is a simple example commandline interface that uses the directory client to read and write the directory.

To write their TD to the directory storage, IoT device agents need to discover the location of the directory and invoke the createThing action, providing the TD JSON document as the payload.

## Backends

This module internally uses a Key-Value bucket store for persisting TD documents. At this point there is no use-case for a custom store so this remains internal.

## Usage

There are two ways to create an instance of the directory.

1. Use the hivekit pipeline factory. This factory accepts a pipeline configuration and automatically creates instances of the neccesary modules.

2. Manually
   1. If the HTTP API is enabled, create an instance of the http server module. Most likely there already is one for use with one of the transport protocols.

   2. Create an instance of the module using NewDirectoryModule() and provide it with the storage location of the embedded database, and the directory router.

   3. Call Start(). This will initialize or create the store and register the HTTP endpoints with the HTTP server module.

   4. To use the RRN message API, link it as a sink to a server module pipeline. Any directory requests will be handled by the module.

   5. Before shutdown call Stop() to ensure the datastore is properly closed.
