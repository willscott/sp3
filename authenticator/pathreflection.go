package authenticator

import (
	"encoding/json"
	"errors"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/willscott/sp3"
	"golang.org/x/net/proxy"

	"log"
	"math/rand"
	"net"
	"net/http"

	"strings"
)

type PathReflectionAuth struct {
	Dialer   proxy.Dialer
	servers  map[string]string
	clientIP net.IP
	conn     net.Conn
	done     chan<- string
}

//This should be kept in-sync with server/lib/pathreflection
type pathReflectionState struct {
	ServerIP              net.IP
	ServerPort            uint16
	ClientIP              net.IP
	ClientPort            uint16
	SequenceNumber        uint32
	AcknowledgementNumber uint32
}

// The default local port range for debian jessie
const IP_LOCAL_PORT_LOW = 32768
const IP_LOCAL_PORT_HIGH = 60999

func CreatePathReflectionAuthFromURL(sp3Url string, clientIP net.IP) (*PathReflectionAuth, error) {
	resp, err := http.Get(sp3Url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	servers := map[string]interface{}{}
	err = json.NewDecoder(resp.Body).Decode(&servers)
	if err != nil {
		return nil, err
	}
	typedServers := make(map[string]string)
	for k := range servers {
		typedServers[k] = k
	}
	return CreatePathReflectionAuth(typedServers, clientIP), nil
}

func CreatePathReflectionAuth(servers map[string]string, clientIP net.IP) *PathReflectionAuth {
	pra := new(PathReflectionAuth)
	pra.clientIP = clientIP
	pra.servers = servers
	return pra
}

func (p *PathReflectionAuth) Authenticate(done chan<- string) (sp3.AuthenticationMethod, []byte, error) {
	p.done = done

	// Choose an allowed host/ip
	var addr string
	var err error
	if len(p.servers) == 0 {
		return sp3.PATHREFLECTION, nil, errors.New("No servers configured for reflection.")
	} else {
		pos := rand.Int() % len(p.servers)
		for k := range p.servers {
			if pos == 0 {
				addr = k
				break
			}
			pos -= 1
		}
	}
	log.Printf("Connection will be to %s", addr)
	// Connect
	if p.Dialer == nil {
		p.Dialer = &net.Dialer{}
	}
	p.conn, err = p.Dialer.Dial("ip4:tcp", addr)
	if err != nil {
		return sp3.PATHREFLECTION, nil, err
	}

	// TCP Handshake
	state := pathReflectionState{
		p.clientIP,
		uint16(IP_LOCAL_PORT_LOW + rand.Int()%(IP_LOCAL_PORT_HIGH-IP_LOCAL_PORT_LOW)),
		net.ParseIP(addr),
		uint16(80),
		uint32(rand.Int()),
		0,
	}

	iplayer := &layers.IPv4{
		Version:  4,
		IHL:      5,
		TTL:      64,
		Protocol: 6,
		SrcIP:    state.ClientIP,
		DstIP:    state.ServerIP,
	}

	tcplayer := &layers.TCP{
		SrcPort: layers.TCPPort(state.ClientPort),
		DstPort: layers.TCPPort(state.ServerPort),
		Window:  4380,
		Seq:     state.SequenceNumber,
		SYN:     true,
	}

	buf := gopacket.NewSerializeBuffer()
	tcplayer.SetNetworkLayerForChecksum(iplayer)
	err = gopacket.SerializeLayers(buf, gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}, tcplayer)
	if err != nil {
		return sp3.PATHREFLECTION, nil, err
	}
	log.Printf("About to write SYN")
	if _, err = p.conn.Write(buf.Bytes()); err != nil {
		return sp3.PATHREFLECTION, nil, err
	}

	// Wait for a syn-ack to learn server's squence number.
	log.Printf("Waiting for SYN-ACK")
	synackbytes := make([]byte, 2048)
	respn, err := p.conn.Read(synackbytes)
	if err != nil {
		return sp3.PATHREFLECTION, nil, err
	}
	rpkt := gopacket.NewPacket(synackbytes[0:respn], layers.LayerTypeTCP, gopacket.Lazy)
	if tcpLayer := rpkt.Layer(layers.LayerTypeTCP); tcpLayer != nil {
		tcp, _ := tcpLayer.(*layers.TCP)
		state.AcknowledgementNumber = tcp.Seq + 1
	} else {
		return sp3.PATHREFLECTION, nil, errors.New("SYNACK not understood.")
	}

	// Set up the listener for server response to injected query.
	go p.listen()

	// Leak State
	data, err := json.Marshal(state)
	if err != nil {
		return sp3.PATHREFLECTION, nil, err
	}
	return sp3.PATHREFLECTION, data, nil
}

func (p *PathReflectionAuth) listen() {
	bufbytes := make([]byte, 2048)
	respn, err := p.conn.Read(bufbytes)
	log.Printf("Path Reflection got an incoming packet.")
	if err != nil {
		log.Printf("Couldn't read path reflection packet: %v", err)
		p.done <- ""
		return
	}
	rpkt := gopacket.NewPacket(bufbytes[0:respn], layers.LayerTypeTCP, gopacket.Default)
	if payload := rpkt.ApplicationLayer(); payload != nil {
		strpayload := string(payload.Payload())
		idx := strings.Index(strpayload, "sp3.")
		if idx == -1 {
			p.done <- ""
			return
		}
		idx += 4
		end := strings.IndexFunc(strpayload[idx:], isbase64)
		if end == -1 {
			p.done <- ""
			return
		}
		p.done <- strpayload[idx : idx+end]
	} else {
		err := rpkt.ErrorLayer()
		log.Printf("Couldn't parse packet!", err)
		p.done <- ""
	}
}

func isbase64(c rune) bool {
	val := (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
	return !val
}
