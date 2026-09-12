# router - HiveOT Request Router

The objective of the router is to deliver request messages to Things that are addressed and pass the response to the replyTo handler. This is intended to be used by a consumer that can sent messages to multiple devices.

## Status

This service is in alpha. Both routing to connected RC devices and stand-alone Thing devices is supported based on the TD.

Authentication with client devices works for bearer tokens. Additional security schemes should be implemented in the 'Authenticate' method of the client implementations. 


## Summary

This service aims to deliver request messages to IoT devices or services identified by the ThingID in the request, the operation and name in the request are used to determine the Thing form needed for deliving the request. The service can create outgoing connections to stand-alone Things or use reverse-connections from the device when running on a gateway.

### Use Of TD Forms

The ThingID is used to lookup the TD (Thing Description) document of the Thing to address. The TD is typically retrieved from a Thing Directory which provides a RetrieveThing method. The Thing Directory can be discovered using the WoT discovery process, or be uploaded locally by an administrator in case of out-of-band provisioning. Finding the directory is out of scope for this service therefore a GetTD method must be provided during instantiation, which is supported by the Directory service.

Determining the request destination:
With the TD known using the ThingID, the service looks up the form of the operation affordance. Combining the TD 'base' attribute and the href value of the operation's 'form' the full endpoint is known. This is described in the [WoT TD specification](https://www.w3.org/TR/wot-thing-description11/#form)

Connecting to IoT devices:
With the address known, the router first determines if a connecting to the endpoint already exists. If so, it is reused with the path from the discovered form.
If a connection doesn't exist, the router will attempt to establish one. When successful, the router passes the request and stores the active connection for later re-use.

Connecting to IoT RC devices:
HiveOT also supports reverse connections by IoT devices. This is intended for use with gateways or hubs that provide a bridge between consumers and IoT devices. Device Things can manage nested Things and use a separate account ID to connect to the gateway. Device Things self-provision by writing the TD's of the Things they manage to the directory. When writing a TD, the directory stores the sender clientID - the device Thing account ID - so the RC client ID is known for each Thing.

Since RC devices don't run servers, the TD they write to the directory does not need to contain forms to connect with. Instead, the router uses the directory to lookup the  device account ID associated with the Thing and forwards requests to the server to whom the device is connected. If successful the request is forwarded to the device Thing which will handle the request or forward it to the nested Thing that is addressed in the request.

If no destination can be found, the requests fail and the sender will receive an error.

### Authentication

When the router handles a request for a Thing it first establishes the connection needed to reach that Thing. Connecting to a Thing often requires credentials. These can be set using SetCredentials providing the TD of the thing to connect to.

Why the TD and not the ThingID?

Devices can serve multiple Things using the same connection. Gateways and Hubs especially can serve many Things over a single connection. Setting the same credentials for every single Thing reachable via the same gateway is a waste of resources and counter productive. 

Credentials are therefore needed for connections, not individual Things. The router stores credentials by connection URL, not ThingID. Credentials only need to be set for one Thing reachable via the gateway for it to apply for all Things reachable via that gateway. Note that a stand-alone device serving two things is also considered a gateway for these two things.

What is this connection URL? This depends on the protocol used. 

Connection based protocols such as websockets, UDS and mqtt require the full URL. If the TD defines this in the 'base' field then this is used. If 'base' is empty then the first Thing level form is used to determe the URL for the preferred protocol. 

The HTTP-Basic protocol can use a different URL per operation. If a Base field is provided it is used. If base is empty then the origin of the first thing level form is used. Eg: https://host:port/. If a proxy server hosts multiple devices on different URLs then it is assumed that the proxy server handles the authentication.

This approach has the limitation that it assumes that all operations on a Thing can use the same connection URL. This is considered 'good enough' until a valid use-case shows it isn't. 

As for preferred protocol. The default order is: UDS, websockets, mqtt and http-basic last.


### Reconnecting Client Connections 

Client connections are used when connecting to stand-alone IoT devices or services. The router can enable a reconnect capability to automatically reconnect to devices whose connection has dropped. 

### Multiple Consumers

The router can be linked to by a chain of one or more consumers, requests from all consumers will be forwarded and answered while notifications are passed back to the consumer chain. When using a lot of consumers, like ConsumedThings, this is less efficient as each consumer in the chain will receive all notifications. 

To alleviate this, each ConsumedThing can be registered as a notification sink using the thingID they represent. Note that only a single notification sink per ThingID can be used.

> router.SetNotificationSink(consumedThing, thingID)


## Usage

To create an instance of this service, a directory instance is required. If reverse-connections by devices is used then one or more transport servers should be provided as well so the router can forward the requests.
