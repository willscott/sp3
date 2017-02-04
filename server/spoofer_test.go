package main

import (
	"bytes"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"net"
	"testing"
	"time"
)

func TestCreateSpoofedStream(t *testing.T) {
	// Send packets to channel, rather than socket.
	TestSpoofChannel = make(chan []byte, 5)

	to := "127.0.0.1"
	from := "127.0.0.1"

	outbound := CreateSpoofedStream(from, to)
	if outbound == nil {
		t.Fatal("Creation of spoofed stream failed.")
	}

	// Send legit packet.
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
	}
	ip := &layers.IPv4{
		Version:  4,
		IHL:      5,
		TTL:      64,
		Protocol: 6,
		SrcIP:    net.IPv4(127, 0, 0, 1),
		DstIP:    net.IPv4(127, 0, 0, 1),
	}
	data := "This is a test packet..."
	ip.Length = 20 + uint16(len(data))
	payload := gopacket.Payload([]byte(data))
	if err := gopacket.SerializeLayers(buf, opts, ip, payload); err != nil {
		t.Fatal("Couldn't construct packet")
	}

	outbound <- buf.Bytes()
	sentPkt := <-TestSpoofChannel
	if !bytes.Contains(sentPkt, []byte(payload)) {
		t.Fatal("Valid packet not spoofed")
	}

	// Send bad packet.
	ip.DstIP = net.IPv4(10, 0, 0, 1)
	if err := gopacket.SerializeLayers(buf, opts, ip, payload); err != nil {
		t.Fatal("Couldn't construct packet")
	}
	outbound <- buf.Bytes()

	select {
	case <-TestSpoofChannel:
		t.Fatal("Bad packet delivered")
	case <-time.After(time.Second):
		t.Log("Bad Packet lost")
	}
}
