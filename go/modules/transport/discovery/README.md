# Discovery

The discovery module offers ways to publish and discover WoT Things and directory following the WoT discovery specification.

Objectives:

1. Serve Thing or directory TDs using WoT mDNS compatible protocols
1. Discover Thing and Directory TDs using WoT mDNS compatible protocols .
1. Seamless integration with the directory in a module chain.

## Status

This module is alpha. It is functional and can be used stand alone and in a module chain. Breaking changes might still happen.

While care has been taken to be compliant with the WoT discovery specification, this has not been testes with 3rd party discovery clients or servers.

## Summary

The discovery module provides both a client and server for device and directory discovery. It integrates with the directory in a module chain by emitting notifications of discovered Things. It provides the following capabilities:

### Serving a Directory TDD

The discovery server can publish a directory TD (TDD) for use by consumers. The directory itself does not need to run on the same machine to be served, only its TDD is needed.

The server publishes a DNS-SD discovery record following the [WoT discovery specification](https://w3c.github.io/wot-discovery/#introduction-dns-sd-sec). The discovery record contains a TXT record with the URL to download the TDD.

The discovery server needs a HTTP server to serve the exploration endpoint for TDD download. This is provided in the constructor. When the module factory is used, the http server must be included in the list of registered modules.

Ways to serve a TDD:
1. Directly invoke ServeDirectoryTD on the module.
2. Pass a request to the module request handler with operation invokeaction, action name ServeDirectoryTDAction (defined in the API), and the TDD JSON as the input.

### Serving a Thing TD

Stand-alone Things can publish their TD using the discovery server. In this scenario the discovery server runs on the device itself.

This follows the WoT discovery specification to publish the TD of a device.

The discovery server needs a HTTP server to serve the TD for download. This is provided in the constructor. When the module factory is used, the http server must be included in the list of registered modules.

Ways to publish the device TD:
1. Directly invoke ServeThingTD on the module.
2. Pass a request to the module request handler with operation invokeaction, action name ServeThingTDAction (defined in the API), and the TD JSON as the input.


### Discover Directory TDDs

Consumers can discover a directory using the discovery client module. 

Ways to discover a Directory TD:
1. Instantiate it with the option to discover on Start, or
1. Invoke the module DiscoverFirstDirectory method, or
1. Invoke the module DiscoverDirectories method to find all directories.
1. Invoke the module DiscoverDirectoryRDs to download the TDDs


### Discover Thing TDs

Consumers can discover stand-alone Thing TDs using the discovery client module.
1. Directly invoke the module DiscoverThings method.
2. Directly invoke DiscoverThingTDs to download the TDs


### Discovery and Use Of RC devices

Devices that reverse connect (RC) to a gateway need to discover this gateway in order to connect to it.

Since reverse connections and gateway discovery are not defined in the WoT specification, the solution chosen is to use the server that serves the directory as the gateway connection. This implies that the gateway runs the directory, which is not unreasonable.

1. The RC device uses directory discovery to obtain the TDD. The TDD describes how to connect to the directory using one of the bi-directional protocols (websocket, sse, mqtt, etc). 
   * alternatively the RC device is provided a URL of the directory.
2. The RC device connects to the directory and writes its TD using the WoT directory specification to create a TD.
3. The RC device keeps this connection to receive requests.
4. The gateway intercepts the directory request, updates the RC device TD forms with that of the gateway, and forward requests for the RC device ThingID to the RC device connection. 
   
This use-case is supported by the directory server and router modules. 

Alternatively, AppEnvironment supports a commandline option to provide a server URL to connect to, bypassing discovery altogether.