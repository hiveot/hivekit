# Agent Module

Agent is a module intended for facilitating developing IoT devices and services.

In can be embedded into an application that overrides HandleRequest, or used as a standalone module where an application hooks into using SetAppRequestHook.


---
FIXME: clarify differences between agent and exposed things
- agents connect to or run a server
- things expose device/service info
- are agents also things?
- are agents merely clients or servers?
- difference between connection clientID vs agentID?
---




## Status

This module is in alpha. It is functional but breaking changes can still happen.


## Usage

in short:  
agent := NewAgent(applicationID)
agent.SetAppRequestHook(func(req,replyTo)error{
    application request handler code
})
