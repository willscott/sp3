package main

/**
 * A client that sends itself packets from the IPv4 space to see if there are
 * networks restricted from ingress at an IP level.
 */

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/willscott/goturn/client"
	"github.com/willscott/sp3"
)

var server = flag.String("server", "localhost:80", "SP3 Server")
var mask = flag.Int("netmask", 24, "how coarse to send sources from (24 = send from each /24)")

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

	// Create a connection to the server
	conn, err := sp3.Dial(*u, udpAddr.IP, sp3.DirectAuth{}, nil)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	log.Printf("SP^3 Connection Established.")

	// make the base packet.
	ip := &layers.IPv4{
		Version:  4,
		IHL:      5,
		TTL:      64,
		Protocol: 17,
		SrcIP:    net.IPv4(0, 0, 0, 0),
		DstIP:    udpAddr.IP,
	}
	udp := &layers.UDP{
		SrcPort: layers.UDPPort(443),
		DstPort: layers.UDPPort(udpAddr.Port),
	}
	udp.SetNetworkLayerForChecksum(ip)
	request := "Hello World!"
	ip.Length = 20 + 8 + uint16(len(request))
	udp.Length = 8 + uint16(len(request))
	payload := gopacket.Payload([]byte(request))

	errchan := make(chan error)

	// recevier.
	go func() {
		recvdSrcs := make([]byte, 1<<(29-uint(*mask)))
		pkt := make([]byte, 2048)
		for {
			stun.SetDeadline(time.Now().Add(time.Second))
			n, readerr := stun.Read(pkt)
			if readerr != nil {
				errchan <- err
				break
			}
			rpkt := gopacket.NewPacket(pkt[0:n], layers.LayerTypeIPv4, gopacket.Default)
			if ipLayer := rpkt.Layer(layers.LayerTypeIPv4); ipLayer != nil {
				ipdat, _ := ipLayer.(*layers.IPv4)
				val := binary.LittleEndian.Uint32(ipdat.SrcIP) / (1 << (32 - uint(*mask)))
				recvdSrcs[val/8] |= 1 << (val % 8)
			}
		}
		// print stats.
		run := 0
		var toAdd = uint32(1 << uint(32-*mask))
		ipspot := make([]byte, 4)
		for i := uint64(0); i < uint64(1<<32)/uint64(toAdd); i++ {
			if recvdSrcs[i/8]&(1<<uint(i%8)) == 0 {
				run++
			} else if run > 0 {
				binary.LittleEndian.PutUint32(ipspot, uint32(uint32(i)*toAdd))
				m := *mask
				for run > 1 {
					m--
					run /= 2
				}
				fmt.Printf("%v/%d\n", net.IPv4(ipspot[0], ipspot[1], ipspot[2], ipspot[3]), m)
				run = 0
			}
		}
	}()

	// Send them.
	var toAdd = uint32(1 << uint(32-*mask))
	var current = uint32(0)
	ipbuf := make([]byte, 4)
	for i := uint64(0); i < uint64(1<<32)/uint64(toAdd); i++ {
		select {
		case msg := <-errchan:
			fmt.Fprintf(os.Stderr, "Reading failed: %v\n", msg)
			return
		default:
			current += toAdd
			binary.LittleEndian.PutUint32(ipbuf, current)
			ip.SrcIP = ipbuf
			buf := gopacket.NewSerializeBuffer()
			if err = gopacket.SerializeLayers(buf, gopacket.SerializeOptions{}, ip, udp, payload); err != nil {
				panic(err)
			}
			_, err = conn.WriteTo(buf.Bytes(), myPublicAddress)
			if err != nil {
				panic(err)
			}
		}
	}
}
