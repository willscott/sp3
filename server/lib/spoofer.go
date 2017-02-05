package server

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"

	"log"
	"net"
)

var (
	// If TestChannel is set, spoofed packets will be sent to it, rather than to pcap.
	TestSpoofChannel chan []byte
	handle           *pcap.Handle
	ipv4Layer        layers.IPv4
	linkHeader       []byte
)
var ipv4Parser *gopacket.DecodingLayerParser = gopacket.NewDecodingLayerParser(layers.LayerTypeIPv4, &ipv4Layer)

func CreateSpoofedStream(source string, destination string) chan []byte {
	dest := net.ParseIP(destination)
	src := net.ParseIP(source)
	flow := make(chan []byte)
	go handleSpoofedStream(src, dest, flow)
	return flow
}

func handleSpoofedStream(src net.IP, dest net.IP, que chan []byte) error {
	if p4 := dest.To4(); len(p4) == net.IPv4len {
		for req := range que {
			if err := SpoofIPv4Message(req, src, dest); err != nil {
				log.Printf("Could not spoof message [%v->%v]: %v", src, dest, err)
				close(que)
				return err
			}
		}
		return nil
	} else {
		return errors.New("UNSUPPORTED")
	}
}

func SetupSpoofingSockets(config Config) error {
	var err error

	handle, err = pcap.OpenLive(config.Device, 2048, false, pcap.BlockForever)
	if err != nil {
		return err
	}
	// make sure the handle doesn't queue up packets and start blocking / dying
	handle.SetBPFFilter("ip.len > 5000")

	srcBytes, _ := hex.DecodeString(config.Src)
	dstBytes, _ := hex.DecodeString(config.Dst)
	linkHeader = append(dstBytes, srcBytes...)
	linkHeader = append(linkHeader, 0x08, 0) // IPv4 EtherType

	//  var ipv6Layer layers.ipv6
	//  ipv6Parser := gopacket.NewDecodingLayerParser(layers.LayerTypeIPv6, &ipv6Layer)
	return nil
}

func SpoofIPv4Message(packet []byte, realSrc net.IP, dest net.IP) error {
	// Make sure destination is okay
	decoded := []gopacket.LayerType{}
	if err := ipv4Parser.DecodeLayers(packet, &decoded); len(decoded) != 1 {
		return err
	}
	if !dest.Equal(ipv4Layer.DstIP) {
		log.Println("Intended packet was to", ipv4Layer.DstIP, "not the authorized", dest)
		return errors.New("INVALID DESTINATION")
	}

	if TestSpoofChannel != nil {
		TestSpoofChannel <- packet
		return nil
	}

	if err := handle.WritePacketData(append(linkHeader, packet...)); err != nil {
		log.Println("Couldn't send packet", err)
		return err
	}
	log.Println(fmt.Sprintf("%d bytes sent to %v as %v from %v", len(packet), dest, ipv4Layer.SrcIP, realSrc))
	return nil
}
