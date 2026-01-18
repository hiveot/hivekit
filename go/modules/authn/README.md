# Authn - Authentication Module

The authentication module provides a method to obtain authentication tokens using password based login.

Existing tokens can be refreshed and tokens can be cancelled.

## Status

This module is in alpha. It is functional but breaking changes can be expected.

## todo

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

Logout requires a valid authentication token.

> POST /authn/logout

### Ping (public route)

> GET /ping

### Refresh Token

Token refresh requires an authenticated connection.

> POST /authn/refresh

See the test cases for example on how to use this module in the code.
