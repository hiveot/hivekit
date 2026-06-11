# Consumer Module

Consumer is a HiveOT module representing a WoT consumer. 

This provides functions to read, write and observe properties, read and subscribe to
events, and invoke and query actions.

In can be embedded into an application that overrides HandleNotification, or used as a standalone module where an application hooks into using SetNotificationHook.

## Status

This module is in alpha. It is functional but breaking changes can still happen.


TODO: should consumer include directory api support? 
 -> how does it know where it is?  
   a. require discovery first
   b. just try the same server as the device
   c. provide the tdd
TODO: should consumer include 'ConsumedThing' types? - currently only used in Hiveoview
 -> if there is a spec maybe
 -> can it replace a consumer? no
    consumers intended for accessing multiple things - directory interface makes sense
    problem with a consumer is that it isnt a client connection
    alt   client -> consumed thing  == client side of a thing   
                    directory

    should consumer have a transportclient?
    a. no, its a chain so not really
    b. yes, 'connected' status?

    maybe a consumer factory recipe with consumer, directory, discovery, authn client?
    ? how to access the apis?   




## Usage

in short:  
agent := NewAgent(applicationID)
agent.SetAppRequestHook(func(req,replyTo)error{
    application request handler code
})
