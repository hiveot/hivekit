# rcrouter - router for reverse connected devices

The objective of the rc-router is to route requests to reverse- connected devices. 

This is intended to be used inside a gateway that supports RC devices and works together with a directory that provides the thing TD.

## Status

This service is in alpha. It is functional but breaking changes should be expected.

## Summary

The rc-router supports reverse connections by IoT devices. Even though this is not defined by any WoT specification, it is possible using the existing WoT TD and transport specifications to make this work. 

RC devices are regular IoT devices that do not run a server but instead connect to a gateway, using a configured account and provided credentials. RC device credentials consist of a client certificate that is self-signed by the gateway CA. Once connected, RC device receive request from the gateway and send event and property notifications to the gateway just like stand-alone devices. 

RC devices can also write their TD and the TD's of the Things they manage to the directory of the gateway. The gateway stores the sender's clientID with the TD for use by the router.

Since RC devices don't run servers, the TD's they write to the directory do not contain forms, security information and base URL that the stand-alone devices include in their TD. 

When the rc-router receives a request, it looks up the TD of the device. If the TD does not contain any forms then this identifies it as a RC device. The rc-router uses the 'senderID' field in the TD to determine the client connection to forward the request to. The senderID field is added to the TD by the directory when a TD is written. 

Consumers don't need to see any of this and don't need to know how the Thing is connected to the gateway. Consumers simply pass all Thing requests to the gateway. 

### Authentication

The rc-router does not establish new connections, so no authentication is needed. 




## Usage

To create an instance of this service, a list of transport servers and a directory instance is required. 

The rc-router is best used in a chain of cells, as provided in the gateway recipe. It is typically placed in a chain before the stand-alone device router.

