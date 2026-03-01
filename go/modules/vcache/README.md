# Vcache Hivekit Module

The vcache module is a value-cache of notifications and requests that have passed through the module. This retains the last request and notification message of a Thing affordance.

Requests for reading properties and events are answered by obtaining the value from the cache without accessing the module directly. This does require that a subscription exists to the devices used.

This can be used for:

1. Performance improvements in clients to obtain values from the cache instead of querying the device itself.
2. A gateway or hub to respond to queries without querying the device.
3. Improved security, masking the actual device from the rest of the network by using connection reversal while serving requests from the cache. Especially effective in combination with a twin module.

## Status

This module is in alpha. It is functional but basic and breaking changes can be expected.

## Summary

The cache module stores the latest notification values that pass through the module. Requests for querying properties, events or actions can be answered directly by the vcache without querying the actual device.

The cache is populated as notifications pass through. In order to receive notifications the module has to be placed in the path of the notifications. This placement is dependent on the pipeline configuration and use-case.

TL&DR, the behavior of the cache is the same. Nothing special needs to be done when used client side or server/gateway side.

1. Consumer placement. When the cache is placed in the pipeline of a consumer, right before the client connection module. All received notifications are passed through the module and are cached. When the client send requests to read Thing properties, events or action status, the cached value can be returned for the Things that the consumer subscribed to.

To support querying properties or events of Things that the consumer has not subscribed to, the cache keeps a list of Thing subscriptions. If a query is directed to an unregistered Thing, the query is forwarded.

If the query is directed to a Thing whose value is subscribed then the currently cached value is returned. If a value is not available then the request is forwarded.

The cache tracks all, a single or multiple subscriptions. If a query request is made for multiple properties or events then it is forwarded unless all requested properties or events have a value available.

2. Device (server) placement. This will do nothing unless the device acts as a consumer and subscribes/queries properties or events of other devices. See the previous point for this setup.

3. Gateway placement. When used inside of a gateway the cache will not receive any subscriptions from consumers, as subscriptions are handled by the server connection. Similar to consumer placement however, all notifications are cached and queries for available properties or events are answered from the cache if available.

This setup can involve reverse connections of devices, or the gateway subscribing to devices it acts as a consumer of. In both cases it is important to place the cache in the pipeline where all received notifications from devices pass through so the value can be cached.

## Usage

Client placement in the module pipeline:

```
Requests:       [consumer] -> [vcache] -> [client] -> [server] -> [device]
Notifications:  [consumer] <- [vcache] <- [client] <- [server] <- [device]
```

Two ways to obtain cached data:

1. Invoke an action to read properties of a Thing by publishing a request.
2. Use the IVCache module interface to read properties, events or action status.
