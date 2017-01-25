/**
 * This file is part of the SP^3 server, implementing the "Path reflection" form
 * of IP verification - namely the injection of a spoofed HTTP request from the
 * SP^3 server to a remote web server that then causes a token to be sent back
 * to the client.
 */

package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"net"
)

type PathReflectionState struct {
	serverIP       net.IP
	serverPort     uint16
	clientIP       net.IP
	clientPort     uint16
	sequenceNumber uint32
}

//TODO: load from json file / make dynamic
const PathReflectionServers = map[net.IP]string{
	net.IPv4(198, 35, 26, 96): "wikimedia.org",
}

func genToken() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	encoded := base64.StdEncoding.EncodeToString(b)
	return encoded, nil
}

func PathReflectionServerTrusted(state PathReflectionState) bool {
	_, ok := PathReflectionServers[state.serverIP]
	return ok
}

func SendPathReflectionChallenge(state PathReflectionState) (string, error) {
	token, err := genToken()
	if err != nil {
		return nil, err
	}
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
	}
	ip := &layers.IPv4{
		SrcIP: state.clientIP,
		DstIP: state.serverIP,
	}
	tcp := &layers.TCP{
		SrcPort: state.clientPort,
		DstPort: state.serverPort,
		Seq:     state.sequenceNumber,
	}
	host := PathReflectionServers[state.serverIP]
	request := "GET /sp3." + token + "/ HTTP/1.0\r\nHost: " + host + "\r\n\r\n"
	payload := gopacket.Payload([]byte(request))
	if err = gopacket.SerializeLayers(buf, opts, ip, tcp, payload); err != nil {
		return nil, err
	}

	//send.
	if err = SpoofIPv4Message(buf.Bytes(), state.clientIP, state.serverIP); err != nil {
		return nil, err
	}

	return token, nil
}
