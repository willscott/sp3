package authenticator

import (
	"encoding/json"
	"github.com/willscott/sp3"
	"github.com/willscott/sp3/server/lib"
	"net"
	"testing"
)

type MockDialer struct {
	net.Conn
}

func (m *MockDialer) Dial(network, address string) (net.Conn, error) {
	return m.Conn, nil
}

func TestAuthenticate(t *testing.T) {
	servconf := server.Config{8888, "eth0", "0", "0", "../server/pathreflection.json"}
	serv := server.NewServer(servconf)
	if serv == nil {
		t.Fatal("Could not start server.")
	}
	go serv.Serve()

	// Create path reflection authenticator.
	auth, err := CreatePathReflectionAuthFromURL("http://localhost:8888/pathreflection.json", net.IP{0, 0, 0, 0})
	if err != nil {
		t.Fatal("Could not create reflection auth from local server.", err)
	}

	// Make a dummy dialer.
	authClientConn, authConnServer := net.Pipe()
	auth.Dialer = &MockDialer{authClientConn}

	// Receive initial syn which authenticator should send out.
	connBuf := make([]byte, 2048)
	go func() {
		n, err := authConnServer.Read(connBuf)
		if err != nil {
			t.Fatal("Authenticator failed to establish connection.")
		}
		connBuf = connBuf[0:n]
		// Send the syn-ack.
		authConnServer.Write(connBuf)
	}()

	// Authenticate
	done := make(chan string, 1)
	mode, opts, err := auth.Authenticate(done)
	if err != nil || mode != sp3.PATHREFLECTION || opts == nil || len(connBuf) == 0 {
		t.Fatal("Could not begin auth process", err)
	}

	// Send the injected packet.
	server.TestSpoofChannel = make(chan []byte, 1)
	decodeopts := &server.PathReflectionState{}
	if err = json.Unmarshal(opts, decodeopts); err != nil {
		t.Fatal("Could not understand auth opts", err)
	}
	challenge, err := server.SendPathReflectionChallenge(servconf, decodeopts)
	if err != nil {
		t.Fatal("Could not generate auth pkg", err)
	}

	// Strip off the ip header.
	injectPkt := <-server.TestSpoofChannel
	authConnServer.Write(injectPkt[20:])

	// auth should now be done
	token := <-done
	if token != challenge {
		t.Fatal("Authentication extracted wrong token from response.")
	}
}
