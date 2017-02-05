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
	"encoding/json"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"io/ioutil"
	"log"
	"net"
)

type PathReflectionState struct {
	serverIP       net.IP
	serverPort     uint16
	clientIP       net.IP
	clientPort     uint16
	sequenceNumber uint32
}

func getPathReflectionServers() map[string]string {
	data, err := ioutil.ReadFile("pathreflection.json")
	if err != nil {
		log.Fatalf("Couldn't read path reflection config: %s", err)
		return nil
	}

	var config map[string]string
	if err := json.Unmarshal(data, &config); err != nil {
		log.Fatalf("Couldn't parse path reflection config: %s", err)
		return nil
	}
	return config
}

func genToken() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	encoded := base64.StdEncoding.EncodeToString(b)
	return encoded, nil
}

func PathReflectionServerTrusted(state *PathReflectionState) bool {
	_, ok := getPathReflectionServers()[state.serverIP.String()]
	return ok
}

func SendPathReflectionChallenge(state *PathReflectionState) (string, error) {
	token, err := genToken()
	if err != nil {
		return "", err
	}
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
	}
	ip := &layers.IPv4{
		Version:  4,
		IHL:      5,
		TTL:      64,
		Protocol: 6,
		SrcIP:    state.clientIP,
		DstIP:    state.serverIP,
	}
	tcp := &layers.TCP{
		SrcPort: layers.TCPPort(state.clientPort),
		DstPort: layers.TCPPort(state.serverPort),
		Window:  4380,
		Seq:     state.sequenceNumber,
	}
	tcp.SetNetworkLayerForChecksum(ip)
	host := getPathReflectionServers()[state.serverIP.String()]
	request := "GET /sp3." + token + "/ HTTP/1.0\r\nHost: " + host + "\r\n\r\n"
	ip.Length = 20 + 20 + uint16(len(request))
	payload := gopacket.Payload([]byte(request))
	if err = gopacket.SerializeLayers(buf, opts, ip, tcp, payload); err != nil {
		return "", err
	}

	//send.
	if err = SpoofIPv4Message(buf.Bytes(), state.clientIP, state.serverIP); err != nil {
		return "", err
	}

	return token, nil
}
