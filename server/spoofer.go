package main

import (
  "encoding/hex"
  "errors"
  "fmt"

  "github.com/google/gopacket"
  "github.com/google/gopacket/layers"
  "github.com/google/gopacket/pcap"

  "log"
  "net"
  "time"
)

var (
  handle *pcap.Handle
  ipv4Layer layers.IPv4
  ipv4Parser *gopacket.DecodingLayerParser
  linkHeader []byte
)

func CreateSpoofedStream(config Config, source string, destination string) chan []byte {
  if handle == nil {
    setupSpoofingSocket(config)
  }

  dest := net.ParseIP(destination)
  src := net.ParseIP(source)
  flow := make(chan []byte)
  go handleSpoofedStream(src, dest, flow)
  return flow
}

func handleSpoofedStream(src net.IP, dest net.IP, que chan []byte) error {
  if p4 := dest.To4(); len(p4) == net.IPv4len {
    for {
      req := <-que
      if err := spoofIPv4Message(req, src, dest); err != nil {
        return err
      }
    }
  } else {
    return errors.New("UNSUPPORTED")
  }
}

func setupSpoofingSocket(config Config) {
  var err error
  ipv4Parser = gopacket.NewDecodingLayerParser(layers.LayerTypeIPv4, &ipv4Layer)

  handle, err = pcap.OpenLive(config.Device, 1024, false, 30 * time.Second)
  if err != nil {
    panic(err)
  }

  srcBytes, _ := hex.DecodeString(config.Src)
  dstBytes, _ := hex.DecodeString(config.Dst)
  linkHeader = append(dstBytes, srcBytes...)
  linkHeader = append(linkHeader, 0x08, 0) // IPv4 EtherType

//  var ipv6Layer layers.ipv6
//  ipv6Parser := gopacket.NewDecodingLayerParser(layers.LayerTypeIPv6, &ipv6Layer)
}

func spoofIPv4Message(packet []byte, realSrc net.IP, dest net.IP) error {
  // Make sure destination is okay
  decoded := []gopacket.LayerType{}
  if err := ipv4Parser.DecodeLayers(packet, &decoded); len(decoded) != 1 {
    return err
  }
  if !dest.Equal(ipv4Layer.DstIP) {
    log.Println("Intended packet was to", ipv4Layer.DstIP, "not the authorized", dest)
    return errors.New("INVALID DESTINATION")
  }

  if err := handle.WritePacketData(append(linkHeader, packet...)); err != nil {
    log.Println("Couldn't send packet", err)
    return err
  }
  log.Println(fmt.Sprintf("%d bytes sent to %v as %v from %v", len(packet), dest, ipv4Layer.SrcIP, realSrc))
  return nil
}
