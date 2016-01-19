# SP^3 Protocol

## Protocol Overview

    Client                  Server                    Destination
    | --- Client Hello --------> |
    |                            |                          |
    |                            | --- Challenge ---------> |
    |                            |                          |
    | <------ Challenge Relayed to client ----------------  |
    |                            |                          |
    | -- Client Authorization -> |
    |                            |
    | <-- Acknowledgement ------ |
    |                            |
    | ----- Packets -----------> |

The protocol begins with a Client Hello message. This is a JSON string where
the client specifies the destination address is wants to send packets to, and
the mechanism that it wants to use to get access to. The server will then relay
a challenge to the destination address. The sender then needs to retrieve
that challenge from the destination and use it to complete authorization from
the server.  Once authorization is granted, the client can send packets to the
destination via the server.

## Protocol Messages

### Client Hello

This is a text string sent as a websocket frame which is the JSON stringification
of the following object:

```javascript
{
  "DestinationAddress": "<destination IP>"
  "AuthenticationMethod": 0
}
```
The list of AuthenticationMethod's is in protocol.go. The simplest is method 0,
where the challenge will be sent through an active websocket connection to the
destination, if one exists.  Other authentication methods are more complex, but
allow for delegation of authority to allow a destination to prove that a client
desiring to receive packets from SP^3 is running at a given IP address without
the need for direct communication between the destination and an SP^3 server.

### Challenge

The challenge for websocket Authentication is a JSON stringified message
with the following fields:

```javascript
{
  "Status": 0,
  "Challenge": "string"
}
```

The challenge is an opaque string, which should be provided to the server by
the sender in a ClientAuthorization message.

### Client Authorization

The client authorizes through a JSON stringified message with the following
fields:

```javascript
{
  "DestinationAddress": "string",
  "Challenge": "string"
}
```

The destination address is the same as before, and is used to support
the ability to authorize multiple destinations. The challenge is the one
provided by the server in its challenge.

### Acknowledgement

If the authorization is successful, the server will respond with a message
indicating that ``` {"Status": 0} ```. Error status codes are documented in
`protocol.go`.

### Packets

Once the client has been authorized, it should send packets to be forwarded
as binary frames over the websocket connection is has open to the SP^3 server.
These messages are Layer 3 packets - beginning with the IP header. The server
will ensure that there is a parseable IPv4 (IPv6 soon!) header, and that the
destination IP address is the one authenticated by the client. Note that the
network sending may place additional restrictions on packets that can be
sent by the server. For instance, the router will not handle source-routed or
improperly check-summed packets.
