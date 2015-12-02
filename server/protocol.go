package main

type AuthenticationMethod int
const (
  WEBSOCKET AuthenticationMethod = iota
  STUNINJECTION
)

type Status int
const (
  OKAY Status = iota
  UNAUTHORIZED     // Sender isn't authorized to send to that destination
  UNSUPPORTED      // Server doesn't support the requested AuthenticationMethod
  INVALID          // Server failed to parse the message
)

type State int
const (
  CLIENTHELLO State = iota // Waiting for client Hello message.
  HELLORECEIVED
  AUTHORIZED               // Acceptable ClientAuhtorization received.
)

type ClientHello struct {
  DestinationAddress string
  AuthenticationMethod AuthenticationMethod
}

type ServerMessage struct {
  Status Status
  Challenge string
  Sent []byte
}

type ClientAuthorization struct {
  DestinationAddress string
  Challenge string
}

type ClientMessage struct {
  Packet []byte
}
