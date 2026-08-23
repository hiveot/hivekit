# AddForms

AddFormsService intercepts directory updates and create actions and modifies the TD payload with base, security, and form information from all configured transports.

This is intended for use in a gateway or hub where Things are reachable through the gateway, and should therefore contain forms for gateway.

## Status

This transport service is in development

## Summary

When a device publishes its TD for discovery or writing it to a directory, the TD must contain the base URL, security scheme, and forms that describe how the thing affordances can be reached.

Since this information belongs to the transport domain it is not readily available in the cells themselves. The purpose of this cell is to enable this updated of published TD's by simply inserting this cell in the request chain.

For this to work cells must write their TD using 'CreateThing' or 'UpdateThing' as described in the directory specification. The cell intercepts this invokeaction request and modifies the published TM/TD with this information before forwarding the modified TD.

Both the discovery service and the directory service handle this request and publish the received TD.

## Usage

The intended use is to include this cell in the request chain between the publisher of the TD and the discovery and directory cells.

This cell can also be used manually through its exposed API.
