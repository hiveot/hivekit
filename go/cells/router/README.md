# router - HiveOT Request Router

The objective of the request router is to deliver request messages to Things and pass the response to the replyTo handler. 

This is intended for use by a consumer that sends messages to multiple devices, or for a gateway that forward incoming requests to stand-alone devices.

## Status

This service is in alpha. It is functional and able to forward requests to devices using websockets, sse-sc, http-basic and grpc protocols.


## Summary

This service aims to deliver request messages to IoT devices or services identified by the ThingID in the request, the operation and name in the request are used to determine the Thing form needed for deliving the request. 

This router can be used both client side and gateway server side:

The router determines the connection needed to send a request to a Thing using its TD. If no connection can be established then the request is forwarded to its sink. 

If the sink is a gateway client then the request can be forwarded to the gateway. 

The router needs a 'getTD' callback that provides the TD for a given thing-ID. This callback can be supplied by directory service, client or a discovery client. A directory client needs a directory server TDD which can be provided through discovery. A discovery client locates devices on a local network. In theory both can be used. If a directory client can't resolve a TD then the discovery client can look for it on the local network. 

The router manages a credentials store for connecting to stand-alone devices. For devices supporting client certificate authentication, the router can be provided this certificate.

If the router cannot deliver a request duel to lack of connection information it forwards the request to its sink. In a gateway, this sink can be a 'rc-router' to add support for routing to reverse connected devices. 


### Use Of TD Forms

To deliver requests, the router needs a Thing TD. Without a TD requests cannot be delivered. 

The ThingID is used to lookup the TD (Thing Description) document of the Thing to address. The TD is typically retrieved from a Thing Directory which provides a RetrieveThing method. The Thing Directory can be discovered using the WoT discovery process, or be uploaded locally by an administrator in case of out-of-band provisioning. Finding the directory is out of scope for this service therefore a GetTD method must be provided during instantiation, which is supported by the Directory service.

Determining the request destination:
With the TD known using the ThingID, the service looks up the form of the operation affordance. Combining the TD 'base' attribute and the href value of the operation's 'form' the full endpoint is known. This is described in the [WoT TD specification](https://www.w3.org/TR/wot-thing-description11/#form)

Connecting to stand-alone IoT devices:
With the address known, the router first determines if a connecting to the endpoint already exists. If so, it is reused. 
If a connection doesn't exist, the router will attempt to establish one. When successful, the router passes the request and stores the active connection for later re-use.

### Authentication

When the router handles a request for a stand-alone Thing, it first establishes the connection needed to reach it. Connecting to a Thing device often requires credentials. These can be set using SetCredentials providing the TD of the thing to connect to. 

Why the TD and not the ThingID?

Devices can serve multiple Things using the same connection. Gateways and Hubs especially can serve many Things over a single connection. Setting the same credentials for every single Thing reachable via the same connection is a waste of resources and counter productive. 

Credentials are therefore needed for connections, not individual Things. The router stores credentials by connection URL, not ThingID. Credentials only need to be set for one Thing reachable via the gateway for it to apply for all Things reachable via that gateway. Note that a stand-alone device serving two things is also considered a gateway for these two things.

What is this connection URL? 
This depends on the protocol used. 

Connections using connection based protocols such as websockets, UDS and mqtt are identified by the full URL. If the TD defines this in the 'base' field then this is used. If 'base' is empty then the first Thing level form is used to determe the URL for the preferred protocol. 

The HTTP-Basic protocol can use a different URL per operation. If a Base field is provided it is used. If base is empty then the origin of the first thing level form is used. Eg: https://host:port/. If a proxy server hosts multiple devices on different URLs then it is assumed that these can all be reached using the same  proxy server connection.

As for preferred protocol. The default preferred order is: UDS, websockets, mqtt and http-basic last.


### Reconnecting Client Connections 

Client connections are used when connecting to stand-alone IoT devices or services. The router can enable a reconnect capability to automatically reconnect to devices whose connection has dropped. 

### Multiple Consumers

The router can be linked to by one or more consumers, requests from all consumers will be forwarded and answered while notifications are passed back to the consumers. 

When using a lot of consumers, like ConsumedThings, this is less efficient as each consumer in the chain will receive all notifications. To alleviate this, each ConsumedThing can be registered as a notification sink using the thingID they represent. Note that only a single notification sink per ThingID can be used.

> router.SetNotificationSink(consumedThing, thingID)


## Usage

To create an instance of this service, a directory client, service or discovery client is required for looking up the requested TD. 

The router is best used in a chain of cells, as provided in the consumer or gateway recipes.

