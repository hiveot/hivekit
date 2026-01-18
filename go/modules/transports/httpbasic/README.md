# WoT HTTP-Basic Transport Module

The HTTP-Basic Transport Module implements the [W3C WoT HTTP-Basic Profile](https://www.w3.org/TR/wot-profile/#http-basic-profile). It is intended to support access to WoT IoT devices using http requests.

Http-basic can be used to publish notifications from systems that support web-hooks.

## Status

This module is in alpha. It is functional but breaking changes can be expected.

## Public and Protected Routes

The HTTP-Basic server uses both public and protected routes as provided by the HTTP server.

The public routes serve ping request and client login. The protected routes are used for sending requests, responses and notifications.

## Dependencies

This module depends on IHttpServer interface, which can be provided by any compatible http server implementation such as the 'httpserver' module. This interface only has 3 methods of which two are routers, so it is easy to use an alternative http server if needed.

## Configuration

The module does not currently use any configurations. The provided server must be configured with the desired port and server certificate.

## Usage

This module adds endpoints for posting [WoT operations](https://www.w3.org/TR/wot-thing-description11/#form) using a standard URL to describe operation, thingID, and affordance name.

This module includes an API to add forms to a TD to access devices via this transport protocol.

All routes are protected. Sending requests over http-basic requires a valid bearer token in the authorization header. Without a valid token an 'unauthorized' response is returned. This is handled by the middleware of the provided http server and not part of this transport module.

See the test cases for example on how to use this module in the code.

### Ping (public route)

> GET /ping

### Sending A Request

> POST /things/{operation}/{thingID}/{name}
> Where: {operation} is one of:

    "cancelaction" | "invokeaction" | "queryaction" |
    "readproperty" | "readmultipleproperties" | "readallproperties" |
    "writeproperty"| "writemultipleproperties" |

For example, to invoke an action: POST /things/invokeaction/{thingID}/{name}
For example, to read a property value: GET /things/readproperty/{thingID}/{name}

This mapping enables all regular Thing interaction.

Devices that use this module can also add their own paths for custom usage.
