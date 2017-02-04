package main

/**
 * a simple client that establishes a connection to the server, and confirms
 * that packets can be injected back to it on a listening UDP port.
 */

import (
	"flag"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/willscott/goturn/client"
	"github.com/willscott/sp3"
	"log"
	"net"
	"net/url"
	"os"
	"os/signal"
)

var server = flag.String("server", "localhost:80", "SP3 Server")

func main() {
	flag.Parse()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Learn external IP address.
	stun, err := net.Dial("udp", "stun.l.google.com:19302")
	if err != nil {
		panic(err)
	}
	defer stun.Close()
	stunclient := client.StunClient{Conn: stun}
	myPublicAddress, err := stunclient.Bind()
	if err != nil {
		panic(err)
	}
	udpAddr, err := net.ResolveUDPAddr(myPublicAddress.Network(), myPublicAddress.String())
	if err != nil {
		panic(err)
	}

	base := url.URL{}
	u, _ := base.Parse(*server)
	log.Printf("Connecting to SP3 at: %v", u)

	// Create a connection to the server
	conn, err := sp3.Dial(*u, udpAddr.IP, sp3.DirectAuth{}, nil)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// Make a packet.
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
	}
	ip := &layers.IPv4{
		Version:  4,
		IHL:      5,
		TTL:      64,
		Protocol: 17,
		SrcIP:    net.IP{8, 8, 8, 8},
		DstIP:    udpAddr.IP,
	}
	udp := &layers.UDP{
		SrcPort: layers.UDPPort(53),
		DstPort: layers.UDPPort(udpAddr.Port),
	}
	udp.SetNetworkLayerForChecksum(ip)
	request := "Hello World!"
	ip.Length = 20 + 8 + uint16(len(request))
	udp.Length = 8 + uint16(len(request))
	payload := gopacket.Payload([]byte(request))
	if err = gopacket.SerializeLayers(buf, opts, ip, udp, payload); err != nil {
		panic(err)
	}

	// Send it.
	_, err = conn.WriteTo(buf.Bytes(), myPublicAddress)
	if err != nil {
		panic(err)
	}

	// Listen for it.
	pkt := make([]byte, 2048)
	n, err := stun.Read(pkt)
	if err != nil {
		panic(err)
	}
	panic("Got spoofed packet: " + string(pkt[0:n]))
}
