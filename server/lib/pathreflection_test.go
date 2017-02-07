package server

import (
	"bytes"
	"net"
	"testing"
)

func TestGenPacket(t *testing.T) {
	TestSpoofChannel = make(chan []byte, 1)

	state := &PathReflectionState{
		net.IP{127, 0, 0, 1},
		uint16(8080),
		net.IP{127, 0, 0, 1},
		uint16(8081),
		0,
		0,
	}
	conf := Config{0, "", "", "", "../pathreflection.json"}
	challenge, err := SendPathReflectionChallenge(conf, state)
	if err != nil {
		t.Fatal("Error sending challenge", err)
	}

	// Compare
	sentPkt := <-TestSpoofChannel
	if !bytes.Contains(sentPkt, []byte(challenge)) {
		t.Fatal("Challenge not in spoofed packet")
	}
}
