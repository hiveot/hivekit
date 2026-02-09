# Authn - Authentication Module

The authentication module provides a method to manage clients and obtain authentication tokens using password based login.

Existing tokens can be refreshed by clients and client tokens can be cancelled.

## Status

This module is in alpha. It is functional but breaking changes can be expected.

## Usage

To create an instance of the module a http server must be provided that will serve the http endpoints.

The http server is optional and used to make http endpoints available for logging in, logging out and token refresh. The AuthnHttpClient is a simple wrapper to simplify its usage.

In order to login and create auth tokens, an account must be created for the client first. The module api can be used to manage clients. The module TM also describes which actions are available for user management through RRN messages.

### HTTP API

This module supports the [Things API](https://w3c.github.io/wot-discovery/#exploration-directory-api-things) of the discovery specification.

The TD published by this module provides the actual endpoints for the various operations. The endpoints described below are examples that are based on the defaults.

#### Login (public route)

Path: /authn/login
Input: JSON object

```json
{
  "username": "name",
  "password": "pass"
}
```

This returns a bearer token that must be placed in the http authorization header:

> "authorization": "bearer {token}"

#### Logout

Logout requires a valid authentication token.

> POST /authn/logout

#### Refresh Token

Token refresh requires an authenticated connection.

> POST /authn/refresh

See the test cases for example on how to use this module in the code.
