/**
 * This file is part of the SP^3 server, implementing the "Path reflection" form
 * of IP verification - namely the injection of a spoofed HTTP request from the
 * SP^3 server to a remote web server that then causes a token to be sent back
 * to the client.
 */

package server

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
	ServerIP       net.IP
	ServerPort     uint16
	ClientIP       net.IP
	ClientPort     uint16
	SequenceNumber uint32
	AcknowledgementNumber uint32
}

func getPathReflectionServers(path string) map[string]string {
	data, err := ioutil.ReadFile(path)
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
	// remove non alphanumeric characters
	output := ""
	for _, code := range encoded {
		if code == '+' || code == '/' || code == '=' {
			continue
		}
		output = output + string(code)
	}
	return output, nil
}

func PathReflectionServerTrusted(conf Config, state *PathReflectionState) bool {
	_, ok := getPathReflectionServers(conf.PathReflectionFile)[state.ServerIP.String()]
	return ok
}

func SendPathReflectionChallenge(conf Config, state *PathReflectionState) (string, error) {
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
		SrcIP:    state.ClientIP,
		DstIP:    state.ServerIP,
	}
	tcp := &layers.TCP{
		SrcPort: layers.TCPPort(state.ClientPort),
		DstPort: layers.TCPPort(state.ServerPort),
		Window:  4380,
		Seq:     state.SequenceNumber,
		Ack:     state.AcknowledgementNumber,
		ACK:     true,
		DataOffset: 5,
	}
	tcp.SetNetworkLayerForChecksum(ip)
	host := getPathReflectionServers(conf.PathReflectionFile)[state.ServerIP.String()]
	request := "GET /sp3." + token + "/ HTTP/1.0\r\nHost: " + host + "\r\n\r\n"
	ip.Length = 20 + 20 + uint16(len(request))
	payload := gopacket.Payload([]byte(request))
	if err = gopacket.SerializeLayers(buf, opts, ip, tcp, payload); err != nil {
		return "", err
	}

	//send.
	if err = SpoofIPv4Message(buf.Bytes(), state.ClientIP, state.ServerIP); err != nil {
		return "", err
	}

	return token, nil
}
