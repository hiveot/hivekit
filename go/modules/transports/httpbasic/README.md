# WoT HTTP-Basic Transport Module

The HTTP-Basic Transport Module implements the WoT HTTP-Basic protocol. It allows access to WoT IoT devices using http requests and emits these requests using RRN (Request-Response-Notification) messages for further processing.

## Public and Protected Routes

The HTTP-Basic server uses both public and protected routes as provided by the HTTP server.

The public routes serve ping request and client login. The protected routes are used for sending requests, responses and notifications.

## Dependencies

This module depends on IHttpServer interface, which can be provided by any compatible http server implementation such as the 'httpserver' module. This interface only has 3 methods of which two are routers, so it is easy to use an alternative http server if needed.

## Configuration

The module does not currently use any configurations. The provided server must be configured with the desired port and certificate configuration.

## Usage

This module adds endpoints for passing RRN type messages. It has an API to add forms to a TD to access devices via this protocols.

For the TM, see (tentative): https://github.com/hiveot/hivekit/go/modules/transports/httpbasic/tm/httpbasic.json

All routes are protected except for the login and ping routes.
Specific operations:

### Login (public route)

Path: /authn/login
Input: JSON object

```json
{
    "username": name,
    "password": pass
}
```

This returns a bearer token that must be used in the http authorization header when accessing protected routes.

### Logout

> POST /authn/logout

### Ping (public route)

> GET /ping

### Refresh Token

> POST /authn/refresh

### Send Request

> POST /things/{operation}/{thingID}/{name}
> Where: {operation} is one of:

    "cancelaction" | "invokeaction" | "queryaction" |
    "readproperty" | "readmultipleproperties" | "readallproperties" |
    "writeproperty"| "writemultipleproperties" |

Invoke Action: POST /things/invokeaction/{thingID}/{name}

This mapping enables all regular Thing interaction. Devices that have their own paths.
Devices can add paths to the module routes for serving special purposes.

etc.
