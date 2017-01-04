Client proof of IP ownership
==============================

This document explores options by which a client program can prove to a remote
server that it is running at a specific Internet IP address at a given time
without establishing a direct connection to the server. Such a protocol can be
motivated by a couple use cases.

Motivation
----------

Consider a client in an adversarial network which actively attempts to block
protocols using SP^3. Communication with a known SP^3 server would allow the
network to prevent connection establishment by blocking the addresses of known
SP^3 servers.

Consider also a client with a limited vantage-point on its network.  This
could be due to a corporate NAT where outbound connections originate from a
pool that the client can't fix, an iridium-like tunnel, or a custom proxy
protocol using SP^3.

Requirements
------------

These mechanisms are designed to prove to a server that a client is running and
has the potential to receive packets that are sent to it. The conditions SP^3
aims to enforce in all such protocols are:

* Active engagement. The client should be running at the time of transfer.
  It should not be possible to launch replay attacks, and traffic should not
  be able to continue once the client has disconnected.

* DDOS resistance. The server should not be usable to overwhelm the remote
  network.

* Locality. The Client must be able to prove it can receive traffic from the
  destination address.

Mechanisms
----------

* Listening Socket
* Web Notary
* Path Reflection

Listening Socket
----------------

For clients behind a NAT, a spoofed packet can be sent to validate connectivity.
This is not limited to symmetric-cone NATs, since the authorization packet can
be sent to appear to come from a connection already established by the client.

The client should first determine an externally connectible address and port,
either by having a publicly visible address, or through a STUN lookup to
establish a UDP NAT port binding.

It then relays these parameters to the sender, which requests authorization
from SP^3.

The SP^3 server will generate a new STUN response to the presumed client, with
an authorization challenge in the transaction ID of the STUN message.

The client then passes this challenge to the sender, who may use it to complete
authorization.

Web Notary
----------

A web notary is an HTTP web server which will respond to HTTP requests with
a signed acknowledgement of the proof. A web-notary protocol is specified
separately. It is worth noting that the notary service can be provided through
several cloud-fronting services to prevent easy identification of the service,
since many cloud providers will pass the original client IP address back to
the customer service within their cloud.

Request Reflection
------------------

A variety of HTTP servers can be used to prove a client's IP address without
direct communication between the client and an SP^3 server. In this scheme,
the client establishes a TCP connection with an HTTP server, and then leaks
the full state of that connection to the SP^3 server via the relay. The
server will then spoof a packet on the active connection making an HTTP
request, and the client can learn the nonce by extracting it from the the
response it receives from the web server.
