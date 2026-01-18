# Logging Module

Logging captures the message flow and writes a filtered copy to the designated output.

The purpose is to track specific messages for review later.

## Status

## Summary

This module should be a installed in between a source and a sink of another module. All messages received by the module will be forwarded to the sink unchanged.

Logging is implemented using slog, the golang structured logging framework. The supported output formats are text and/or json. Logging can be configured through a yaml configuration file that defines the filters and outputs. ANSI colorized output is also supported. Custom handlers as described [here](https://betterstack.com/community/guides/logging/logging-in-go/#customizing-slog-handlers) including tint, slog-sampling and slog-multi are also supported.

While slog provides a unified logging frontend, it supports different backends.

This is work in progress.
