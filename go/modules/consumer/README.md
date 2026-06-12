# Consumer Module

Consumer is a HiveOT module representing a WoT consumer. 

This provides functions to read, write and observe thing properties, read and subscribe to
events, and invoke and query actions.

It is intended to be used in the beginning of a hivekit module chain that connects to a device or gateway. Multiple consumers can be chained together, each handling an aspect of the application.

The ConsumedThing module is a local representation of a remote Thing. It contains its state, latest event and can be used to send actions to the Thing.

## Status

The Consumer module is in alpha. It is functional but breaking changes can still happen.

ConsumedThing is under development.



## Usage

in short:  

```
consumer := NewConsumer(nil)
consumer.SetNotificationHook(func(notif *msg.NotificationMessage){
     notification processing
})
```

ConsumedThing is intended for applications that connect to one or more things. It should be connected to the device or gateway using the TD obtained for this device. It needs to be linked to a module that knows how to connect to the device, like the router module, which is linked to a directory holding a copy of Thing TDs. Some connection options: 

*  <=> means a linking modules for both notification and requests
*  => means linking the request handling to the next module in the chain
*  <= means linking the module as the notification sink of the next module in the chain
*  <->  means linking both notification and requests for a single specific thingID. This is only supported by the router.

Option 1: a single consumed thing connects to a device or gateway using a transport client for a known protocol
> ConsumedThing <=> Transport Client

Option 2: a single consumed thing connects using the Router who uses the TD forms to establish a connection to device or gateway
> ConsumedThing <=> Router <=> Transport Client

Option 3: multiple consumed things 
> ConsumedThing 1..* <-> Router <=> 1..* Transport Client

The standard HiveModule linking only supports a single chain. To use multiple ConsumedThings, multiple notification handlers need to be linked to the module. Instead of SetNotificationLink, use AddNotificationLink(thingID) on the router. Only notifications for the specific thingID are received over that link.

