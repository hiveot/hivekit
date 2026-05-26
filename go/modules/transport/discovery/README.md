# discovery

The discovery module offers ways to publish and discover WoT devices and directory of devices following the WoT discovery specification.

Objectives:

1. Serve Thing or directory TDs using WoT mDNS compatible protocols
1. Discover directory TDs using WoT mDNS compatible protocols .
1. Support discovery of protocol endpoints for RC (reverse connections).

## Status

This module is alpha. It is functional but basic. Breaking changes might still happen.

While care has been taken to be compliant with the WoT discovery specification, this has not been testes with 3rd party discovery clients or servers.

Still to do:

1. Provide an elegant solution for automatically include the Thing or Directory TD when used in a factory chain. Who is responsible for taking a device TD, updating it with the security scheme, forms and passing it on to the discovery module?
2. Define how discovery of RC connected devices should function. There isn't a WoT specification. RC devices don't need forms or security scheme as messaging works via the gateway.

## Summary

The discovery module provides both a client and server for device or directory discovery. It supports discovery of individual devices and discovery of directories.

The server publishes a DNS-SD discovery record following the [WoT discovery specification](https://w3c.github.io/wot-discovery/#introduction-dns-sd-sec). This record contains the URL of the device or directory TD for accessing the TD as described in this specification. A http server is needed serves this URL for downloading the TD of the device or directory.

To publish a Thing TD or Directory TD, invoke the module ServeThingTD or ServerDirectoryTD.

### Discovery of Reverse Connected devices on a Gateway

This discovery server can be part of the module chain on a gateway. A directory running on the gateway can publish its directory TD on this discovery server.

Devices that do not run servers need to discover the gateway, connect to it using reverse connection and update the directory with its TD, if present.
The best approach for this is work in progress:
option 1: discover a directory and assume it is the gateway
option 2: discover a directory, write its TD and lookup a gateway device for a messaging connection
option 3: use the serverURL commandline argument

Since RC devices do not expose an endpoint there is no need for a base URL and forms in their TD. The security scheme is a NoScheme. The TD needs to be updated by the gateway to include the gateway security scheme, the corresponding base URL, and forms for device operations using the gateway protocol.
