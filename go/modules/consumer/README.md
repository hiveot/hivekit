# Consumer Module

Consumer is a HiveOT module representing a WoT consumer. 

This provides functions to read, write and observe properties, read and subscribe to
events, and invoke and query actions.

In can be embedded into an application that overrides HandleNotification, or used as a standalone module where an application hooks into using SetNotificationHook.

## Status

This module is in alpha. It is functional but breaking changes can still happen.


## Usage

in short:  
agent := NewAgent(applicationID)
agent.SetAppRequestHook(func(req,replyTo)error{
    application request handler code
})
