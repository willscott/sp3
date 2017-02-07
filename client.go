package sp3

import (
	"errors"
	"github.com/gorilla/websocket"
	"log"
	"net"
	"net/url"
	"time"
)

/**
 * A golang client for connecting to an SP^3 server to send packets.
 */
func Dial(sp3server url.URL, destination net.IP, auth Authenticator, dialer *websocket.Dialer) (*Sp3Conn, error) {
	finished := make(chan string)
	mode, opts, err := auth.Authenticate(finished)

	if err != nil {
		return nil, err
	}

	if dialer == nil {
		dialer = websocket.DefaultDialer
	}

	conn := &Sp3Conn{}
	conn.incomingMessage = make(chan ServerMessage)
	conn.destination = &net.IPAddr{IP: destination}
	conn.Conn, _, err = dialer.Dial(sp3server.String(), nil)
	if err != nil {
		return nil, err
	}

	// Send SenderHello.
	hello := &SenderHello{
		destination.String(),
		mode,
		opts,
	}
	err = conn.Conn.WriteJSON(hello)
	if err != nil {
		conn.Close()
		return nil, err
	}

	go conn.readLoop()

	// Wait for Authenticator to finish challenge.
AuthLoop:
	for {
		select {
		case challenge := <-finished:
			if len(challenge) == 0 {
				conn.Close()
				return nil, errors.New("Authentication failed.")
			}
			// Finish Authentication
			auth := &SenderAuthorization{
				destination.String(),
				challenge,
			}
			err = conn.Conn.WriteJSON(auth)
			if err != nil {
				return nil, err
			}
			break AuthLoop
		case msg := <-conn.incomingMessage:
			if msg.Status != OKAY {
				conn.Close()
				return nil, errors.New("Server closed connection with status: " + string(int(msg.Status)))
			} else if mode == WEBSOCKET && len(msg.Challenge) > 0 {
				// On another thread to prevent blocking.
				go func() {
					finished <- msg.Challenge
				}()
			}
		}
	}

	// Make sure server is okay with auth.
	msg := <-conn.incomingMessage
	if msg.Status != OKAY {
		conn.Close()
		return nil, errors.New("Server rejected authentication (" + string(int(msg.Status)) + ") " + msg.Challenge)
	}

	// Watch for incoming errors.
	go conn.watchLoop()

	return conn, nil
}

/**
 * An SP3 Authenticator is the interface for a specific authentication method.
 * The method is passed a channel to complete the challenge from the server;
 * THe Authenticate method should set up a listener, and then provide the
 * AuthenticationMethod and AuthenticationOptions for the SenderHello message.
 */
type Authenticator interface {
	Authenticate(chan<- string) (AuthenticationMethod, []byte, error)
}

type DirectAuth struct {
	done chan<- string
}

func (d DirectAuth) Authenticate(done chan<- string) (AuthenticationMethod, []byte, error) {
	d.done = done
	return WEBSOCKET, []byte{}, nil
}

type Sp3Conn struct {
	*websocket.Conn
	destination     net.Addr
	incomingMessage chan ServerMessage
	lastError       error
}

func (s *Sp3Conn) readLoop() {
	for {
		msg := new(ServerMessage)
		err := s.Conn.ReadJSON(msg)
		if err != nil {
			close(s.incomingMessage)
			s.lastError = err
			break
		}
		s.incomingMessage <- *msg
	}
}

func (s *Sp3Conn) watchLoop() {
	for {
		msg, ok := <-s.incomingMessage
		if msg.Status != OKAY || !ok {
			if ok {
				s.Close()
				s.lastError = errors.New("Server Closed Connection: " + string(int(msg.Status)))
			} else {
				s.lastError = errors.New("Network Connection Closed")
			}
			return
		}
	}
}

func (s *Sp3Conn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	return 0, nil, errors.New("SP3 Connections do not receive data.")
}

func extractHost(addr net.Addr) string {
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil || len(host) == 0 {
		return addr.String()
	}
	return host
}

func (s *Sp3Conn) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	if extractHost(addr) != extractHost(s.destination) {
		log.Printf("Invalid Destination %v vs %v", extractHost(addr), extractHost(s.destination))
		return 0, errors.New("Invalid Destination")
	}
	if s.lastError != nil {
		return 0, s.lastError
	}
	err = s.Conn.WriteMessage(websocket.BinaryMessage, b)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func (s *Sp3Conn) Close() error {
	if s.lastError != nil {
		return s.lastError
	}
	return s.Conn.Close()
}

func (s *Sp3Conn) LocalAddr() net.Addr {
	return nil
}

func (s *Sp3Conn) SetDeadline(t time.Time) error {
	if s.lastError != nil {
		return s.lastError
	}
	return s.Conn.SetWriteDeadline(t)
}

func (s *Sp3Conn) SetReadDeadline(t time.Time) error {
	return errors.New("Reads are not supported by this form of Connection")
}

func (s *Sp3Conn) SetWriteDeadline(t time.Time) error {
	if s.lastError != nil {
		return s.lastError
	}
	return s.Conn.SetWriteDeadline(t)
}
