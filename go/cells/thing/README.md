# Exposed Thing 

The exposed Thing is intended for developing IoT devices and services.

In can be embedded into an application that overrides HandleRequest, or used as a standalone cell where an application hooks into using SetAppRequestHook.

Exposed Things can be nested. For example a 1-wire gateway is a Thing device with 1-wire devices.

HiveKit has the convention that exposed things have a 'td' property that holds the current active TD for that thing. On Start it publishes a property notification to subscribers. The Hivekit directory will update itself if it sees this notification. It also allows consumers to update the cached device TD by reading the 'td' property. The WoT discovery specification does not specify who updates a directory with TD's, just defines the write directory action. This approach uses property notifications instead.



## Status

This cell is in alpha. It is functional but breaking changes can still happen.


## Usage

The exposed thing can be used 'as-is' by registering an app request hook that handles the requests sent to the device, or it can be embedded into a service.

When using the request hook:  
```go
m := thing.NewExposedThing(applicationID)
m.SetAppRequestHook(func(req,replyTo)error{
    //application request handler code
})
```

See [ExposedThing.go](ExposedThing.go) for details.


