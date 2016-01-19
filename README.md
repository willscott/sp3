(SP)^3: A Simple Practical & Safe Packet Spoofing Protocol
======

Install an SP^3 Server: `go get github.com/willscott/sp3`

SP3 provides a mechanism through which a server which has the capability to
spoof packets can offer that capability in a limited capacity. In particular,
the protocol supports spoofing packets as long as the destination `client`
consents in advance to receive those communications.

Why?
-----

There are several uses of SP^3 we've thought of, and we're sure there are many
more.

* NAT hole-punching facilitation.
    Currently, NAT holepunching only works for UDP, partially because even
    when the clients are controlled, it generally requires root permissions
    to send packets with a specific sequence number.  Having a source of
    packet injection can provide a mechanism to synchronize sequence numbers
    and create TCP connections between two NAT'ed machines.

* Firewall characterization.
    It's often difficult to test how your network will respond to packets sent
    from black-holed or unadvertised prefixes. A source of packets can allow you
    to validate firewall rules and routing policy.

* Circumvention.
    The ability to send packets from arbitrary sources can help to mask traffic
    by adding a layer of cover trafic and IP diversity that makes surveilance
    much more difficult.

Design
-----

There are three participants in SP3: the `server`, `client`, and `sender`.
The server is the host which can send spoofed packets. It acts as a relay,
accepting encapsulated IP packets from the sender and sending them to the client,
even when their source address is spoofed.  The `client` is the destination that
receives the packets. The `sender` is the host that generates the packets.

One issue with packet spoofing is the number of attack vectors it opens. In order
to provide a service that makes a reasonable tradeoff of enabling valid use
cases while not opening itself up to abuse and attacks, the `server` enforces
a policy on packets it is willing to send.  The primary property the `server`
attempts to guarantee is that the `client` consents to receiving spoofed packets.

The server provides a number of mechanisms by which the client can provide this
consent. The simplest is that the client establishes a connection to the server,
and directly tells the server it is wiling to receive the spoofed packets.  This
can be done either through a direct TCP connection, or with a web-socket connection
when the `client` is code running in a web browser. Other methods for when the
client cannot or is unwilling to establish a direct connection to the server are
more complex and explained in the documentation.

Server
------

Build:
```bash
cd server
go build
```

Run
```bash
sudo ./server [--port 8080]
```

Sender
------

Client
------

A web based client is included in the `client` directory.
