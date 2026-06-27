# Exposed Thing Module

The exposed Thing module is intended for developing IoT devices and services.

In can be embedded into an application that overrides HandleRequest, or used as a standalone module where an application hooks into using SetAppRequestHook.

Exposed Things can be nested. For example a 1-wire gateway is a Thing device with 1-wire devices.


## Status

This module is in alpha. It is functional but breaking changes can still happen.


## Usage

in short:  
m := NewEThing(applicationID)
m.SetAppRequestHook(func(req,replyTo)error{
    application request handler code
})
