package main

import (
  "errors"

  "github.com/google/gopacket"
  "github.com/google/gopacket/layers"

  "golang.org/x/net/ipv4"
//  "golang.org/x/net/ipv6"

  "net"
)

func CreateStream(destination string) chan []byte {
  if packet4Conn == nil {
    SetupSockets()
  }

  dest := net.ParseIP(destination)
  flow := make(chan []byte)
  go HandleStream(dest, flow)
  return flow
}

func HandleStream(dest net.IP, que chan []byte) {
  for {
    req := <-que
    ConditionalForward(req, dest)
  }
}

var (
  packet4Conn net.PacketConn
  raw4Conn *ipv4.RawConn
  ipv4Layer layers.IPv4
  ipv4Parser *gopacket.DecodingLayerParser
)

func SetupSockets() {
  var err error
  ipv4Parser = gopacket.NewDecodingLayerParser(layers.LayerTypeIPv4, &ipv4Layer)

  packet4Conn, err = net.ListenPacket("ip4", "127.0.0.1")
  if err != nil {
    panic(err)
  }
  raw4Conn, err = ipv4.NewRawConn(packet4Conn)
  if err != nil {
    panic(err)
  }

//  var ipv6Layer layers.ipv6
//  ipv6Parser := gopacket.NewDecodingLayerParser(layers.LayerTypeIPv6, &ipv6Layer)
}

func ConditionalForward(packet []byte, dest net.IP) error {
  if p4 := dest.To4(); len(p4) == net.IPv4len {
    return ConditionalForward4(packet, dest)
  } else {
    return ConditionalForward6(packet, dest)
  }
}

func ConditionalForward4(packet []byte, dest net.IP) error {
  // Make sure destination is okay
  decoded := []gopacket.LayerType{}
  if err := ipv4Parser.DecodeLayers(packet, &decoded); err != nil {
    return err
  }
  if dest.Equal(ipv4Layer.DstIP) {
    return errors.New("INVALID DESTINATION")
  }

  if _, err := raw4Conn.Write(packet); err != nil {
    return err
  }
  return nil
}

func ConditionalForward6(packet []byte, dest net.IP) error {
  return errors.New("UNSUPPORTED")
}
