package main

type AuthenticationMethod int
const (
  WEBSOCKET AuthenticationMethod = 1 + iota
  STUNINJECTION
)

type Status int
const (
  OKAY Status = 1 + iota
  UNAUTHORIZED     // Sender isn't authorized to send to that destination
  INVALID          // Server failed to parse the message
)

type ClientHello struct {
  DestinationAddress string
  AuthenticationMethod AuthenticationMethod
}

type ServerHello struct {
  Challenge string
  Status Status
}

type ClientMessage struct {
  Packet []byte
}

type ServerMessage struct {
  Status Status
}
