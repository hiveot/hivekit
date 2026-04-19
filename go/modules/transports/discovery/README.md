# discovery

The discovery module offers ways to publish and discover WoT devices and directory of devices following the WoT discovery specification.

Objectives:

1. Serve Thing or directory TDs using WoT mDNS compatible protocols
1. Discover directory TDs using WoT mDNS compatible protocols .
1. Support discovery of protocol endpoints for RC (reverse connections).

## Status

This module is alpha. It is functional but basic. Breaking changes might still happen.

While care has been taken to be compliant with the WoT discovery specification, this has not been testes with 3rd party discovery clients or servers.

## Summary

The discovery module provides both a client and server for device or directory discovery.

The server publishes a DNS-SD discovery record following the [WoT discovery specification](https://w3c.github.io/wot-discovery/#introduction-dns-sd-sec). This record contains the URL of the device or directory TD for accessing the directory as described in this specification. The provided http server serves this URL for downloading the TD of the device or directory.

The client provides the capability to discover this record and download the TD of the thing or directory.
