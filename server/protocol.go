package main

type AuthenticationMethod int

const (
	WEBSOCKET AuthenticationMethod = iota
	STUNINJECTION
	PATHREFLECTION
)

type Status int

const (
	OKAY         Status = iota
	UNAUTHORIZED        // Sender isn't authorized to send to that destination
	UNSUPPORTED         // Server doesn't support the requested AuthenticationMethod
	INVALID             // Server failed to parse the message
)

type State int

const (
	SENDERHELLO State = iota // Waiting for client Hello message.
	HELLORECEIVED
	AUTHORIZED // Acceptable ClientAuhtorization received.
)

type SenderHello struct {
	DestinationAddress    string
	AuthenticationMethod  AuthenticationMethod
	AuthenticationOptions []byte
}

type ServerMessage struct {
	Status    Status
	Challenge string
	Sent      []byte
}

type SenderAuthorization struct {
	DestinationAddress string
	Challenge          string
}

type SenderMessage struct {
	Packet []byte
}
