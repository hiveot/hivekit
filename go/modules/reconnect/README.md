# Reconnect Module

Reconnect is a module that automatically re-applies request a reconnect after a client loses its connection and applies event subscriptions and property observations after a connection is restored.



## Status

This module is in alpha. It is functional but breaking changes can still happen.


## Usage

Place this module between a consumer and a connection client module in the chain:

> consumer -> reconnect -> [wss|grpc|...]client

For this to work the client module must support disconnect and connect notifications (which all hiveot clients do)
