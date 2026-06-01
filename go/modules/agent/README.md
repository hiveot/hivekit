# Agent Module

Agent is a module intended for facilitating developing IoT devices and services.

In can be embedded into an application that overrides HandleRequest, or used as a standalone module where an application hooks into using SetAppRequestHook.

## Status

This module is in alpha. It is functional but breaking changes can still happen.


## Usage

in short:  
agent := NewAgent(applicationID)
agent.SetAppRequestHook(func(req,replyTo)error{
    application request handler code
})
