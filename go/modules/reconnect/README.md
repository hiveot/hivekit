# Reconnect Module

The objective of the Reconnect module is to automatically reconnect and restore subscriptions when the connection of the given client unexpectedly drops.

## Status

This module is in alpha. It is functional but breaking changes can still happen.

## Summary

This module controls connecting the provided transport client. The provided client must already have been setup with credentials to authenticate its connection. It registers the connect callback of the client so it can ask the client to re-connect. If the connection fails a new connection is requested after a backoff period. The backup period increases after each failed attempt until a limit is reached.

This module stores event subscription and property observe requests and replays these after the connection is restored.



## Usage

Place this module behind a consumer and provide it with a transport client.

> consumer -> Reconnect -> [wss|grpc|...]client

For this to work the client module must support disconnect and connect notifications (which all hiveot clients do)
