# router - HiveOT Request Routing Module

The objective of the router module is to deliver request messages to Things that are addressed and pass the response to the replyTo handler. This is intended to be used by a client that can sent messages to multiple devices.

## Status

This module is in early alpha. Routing to connected RC agents is supported.
Routing to stand-alone devices is under development.

## Summary

This module aims to deliver request messages to IoT devices or services identified by the ThingID in the request, the operation and name in the request are used to determine the final href to use for deliving the request. The module can create outgoing connections to Things or use reverse-connections from device agents.

The GetTD Method:
The ThingID is used to lookup the TD (Thing Description) document of the Thing to address. The TD is typically retrieved from a Thing Directory which provides a RetrieveThing method. The Thing Directory can be discovered using the WoT discovery process, or be uploaded locally by an administrator in case of out-of-band provisioning. Finding the directory is out of scope for this module therefore a GetTD method must be provided during instantiation.

Determining the request destination:
With the TD known using the ThingID, the module looks up the form of the operation affordance. Combining the TD 'base' attribute and the href value of the operation's 'form' the full endpoint is known. This is described in the [WoT TD specification](https://www.w3.org/TR/wot-thing-description11/#form)

Connecting to IoT devices:
With the addres known, the router first determines if a connecting to the endpoint already exists. If so, it is reused with the path from the discovered form.
If a connection doesn't exist, the router will attempt to establish one. When successful, the router passes the request and stores the active connection for later re-use.

Connecting to IoT RC agents:
HiveOT also supports reverse connections by IoT devices. This is intended for gateways, hubs and similar devices that bridge between consumers and IoT devices. Device agents can manage multiple devices and have their own account ID. Agents self-provision by writing the TD's of the devices it manages to the directory. When writing a TD, the directory stores the sender clientID in a custom field in the TD so the sender is known for each TD.

Since RC agents don't run servers, the TD they write will not have any href's to connect to. Instead, the router will use the sender clientID in the TD to lookup the agent connection with the provided transport servers. If successful the request is forwarded to the agent which will forward it to the device it manages.

If no destination can be found, the requests fail and the sender will receive an error.

A future feature can be to support request lifespan and cache the request until a connection can be established or the request expires.

## Usage

To create an instance of this module, a directory instance is required. If reverse-connections by agents is supported then one or more transport servers should be provided as well.
