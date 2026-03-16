# vcache - Value Cache

The vcache module is a simple cache with property and event notifications that have passed through the module. This retains the notification messages of a Thing affordance.

Requests for reading properties and events are answered by obtaining the notification from the cache without accessing the device directly. This does require that a subscription exists to receive notifications for the devices used.

This can be used for:

1. Performance improvements in clients to obtain values from the cache instead of querying the device itself.
2. A gateway or hub to respond to queries without querying the device.
3. Improved security, masking the actual device from the rest of the network by using connection reversal while serving requests from the cache. Especially effective in combination with the 'twin' module.

## Status

This module is in alpha. It is functional but basic and breaking changes can be expected.

## Summary

The notification cache module stores the latest notifications that pass through the module. Requests for querying properties and events can be answered directly by the vcache without querying the actual device.

The vcache is populated as notifications pass through. In order to receive notifications the module has to be placed in the path of the notifications. This placement is dependent on the pipeline configuration and use-case.

1. Consumer placement. When the cache is placed in the pipeline of a consumer, right before the client connection module. All received notifications passed to the consumer are cached. When the client send requests to read Thing properties, events or action status, the cached value can be returned for the Things that the consumer subscribed to.

When the consumer queryies properties or events of Things that the consumer has not subscribed to, the query is forwarded to the device.

2. Gateway placement. When used inside of a gateway the cache will not receive any subscriptions from consumers as subscriptions are handled by the server connection. There are two possible ways to address this:

A. The gateway forwards subscriptions to the device.
Multiple consumers each subscribing would each add a subscription, something the device has to support. A notification would be sent to the gateway for each subscription, multiplying the number of notifications between device and gateway.

B. The gateway subscribes to all properties/events of connected devices.
Notifications from a device are received once then forwarded by the gateway to each consumer that has a subscription. Since all notifications are received by the gateway, its cache would be up to date as long as the devices send notifications for all properties. Some properties, like counters, are too chatty for this so these should not be observable.

Limitations:

In case of a gateway, the gateway has to subscribe to all properties and events from all connected devices to ensure it will receive and can forward notifications to subscribers. This setup can involve reverse connections of devices, or the gateway subscribing to devices it acts as a consumer of. In both cases it is important to place the cache in the pipeline where all received notifications from devices pass through so the value can be cached.

As the cache doesn't have a Thing's TD, it does not know if it has received notifications for all properties. As a result it cannot return a query for all properties so it should forward these queries to the device. In this case the cache is not effective.

If a query for one or multiple properties is made and it has received values for the requested properties it can provide these directly. In this case the cache is effective.

If more properties are requested than cached it has to forward the query request to the device. In this case performance might suffer slightly as the check has to be made for each property. This probably is insignificant compare to the time the request travels over the network to the device when it is forwarded. Still, in this case the cache is not effective.

If the device itself is not reachable because it is offline or sleeping, the forwarded requests will fail, even though some values might be in the cache.

Digital Twin:

If continuous access to a device is important for an application, the use of a digital twin could be considered. The digital twin caches and subscribes to all available values of a device and publishes a new TD under a different Thing ID. Consumers talk directly to the digital twin, which is updated when the actual device is reachable. This also opens the door for injecting simulation data for integration testing purposes. See the 'twin' module for more details.

## Usage

Client placement in the module pipeline:

```
Requests:       [consumer] -> [vcache] -> [client] -> [server] -> [device]
Notifications:  [consumer] <- [vcache] <- [client] <- [server] <- [device]
```

Two ways to access cached data:

1. Invoke an action to read properties of a Thing by publishing a request.
2. Use the IVCache module interface to read the latest values for properties and events.

When issuing a request to read properties this returns the property value or map of property name-values.
When issuing a request to read an even this returns the notification itself as the timing of the event is important.
