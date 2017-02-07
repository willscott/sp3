package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pborman/uuid"
	"github.com/willscott/sp3"
)

type Server struct {
	sync.Mutex
	upgrader     websocket.Upgrader
	webServer    http.Server
	config       Config
	destinations map[string]*websocket.Conn
	clientHosts  map[string]*websocket.Conn
}

type Config struct {
	Port               int
	Device             string
	Src                string
	Dst                string
	PathReflectionFile string
}

func (s Server) Authorize(hello sp3.SenderHello) (challenge string, err error) {
	if hello.AuthenticationMethod == sp3.PATHREFLECTION {
		state := &PathReflectionState{}
		if err = json.Unmarshal(hello.AuthenticationOptions, state); err != nil {
			return "", err
		}
		if !PathReflectionServerTrusted(s.config, state) {
			return "", errors.New("Untrusted Server")
		}
		return SendPathReflectionChallenge(s.config, state)
	} else if hello.AuthenticationMethod == sp3.WEBSOCKET {
		if val, ok := s.clientHosts[hello.DestinationAddress]; ok {
			resp := sp3.ServerMessage{
				Status:    sp3.OKAY,
				Challenge: uuid.New(),
			}
			dat, _ := json.Marshal(resp)
			if err = val.WriteMessage(websocket.TextMessage, dat); err != nil {
				return "", err
			}
			return resp.Challenge, nil
		}

		return "", errors.New("No active connection from requested destination.")
	} else {
		return "", errors.New("UNSUPPORTED")
	}
}

func (s Server) Cleanup(remoteAddr string) {
	log.Printf("Closed connection from %s.", remoteAddr)
	if conn, ok := s.destinations[remoteAddr]; ok {
		conn.Close()
		delete(s.destinations, remoteAddr)
		if addrHost, _, err := net.SplitHostPort(remoteAddr); err == nil {
			if cc, ok := s.clientHosts[addrHost]; ok && cc == conn {
				delete(s.clientHosts, addrHost)

				// See if there's another connection from the same address to promote.
				for addr, otherConn := range s.destinations {
					if destAddr, _, err := net.SplitHostPort(addr); err == nil && destAddr == addrHost {
						s.clientHosts[destAddr] = otherConn
						break
					}
				}
			}
		}
	}
}

func IPHandler(server *Server) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		addrHost, _, _ := net.SplitHostPort(r.RemoteAddr)
		fmt.Fprintf(w, "externalip({ip:\"%s\"})", addrHost)
	})
}

func SocketHandler(server *Server) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := server.upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		addrHost, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return
		}

		server.destinations[r.RemoteAddr] = c
		if _, ok := server.clientHosts[addrHost]; !ok {
			server.clientHosts[addrHost] = c
		}
		senderState := sp3.SENDERHELLO
		var sendStream chan<- []byte
		challenge := ""

		defer server.Cleanup(r.RemoteAddr)
		for {
			msgType, msg, err := c.ReadMessage()
			if err != nil {
				log.Println("read err:", err)
				break
			}
			if senderState == sp3.SENDERHELLO && msgType == websocket.TextMessage {
				hello := sp3.SenderHello{}
				err := json.Unmarshal(msg, &hello)
				if err != nil {
					log.Println("Hello err:", err)
					break
				}

				chal, err := server.Authorize(hello)
				if err != nil {
					log.Println("Authorize err:", err)
					resp := sp3.ServerMessage{
						Status: sp3.UNAUTHORIZED,
					}
					dat, _ := json.Marshal(resp)
					c.WriteMessage(websocket.TextMessage, dat)
					break
				}
				challenge = chal
				senderState = sp3.HELLORECEIVED
				continue
			} else if senderState == sp3.HELLORECEIVED && msgType == websocket.TextMessage {
				auth := sp3.SenderAuthorization{}
				err := json.Unmarshal(msg, &auth)
				if err != nil {
					log.Println("Auth err:", err)
					break
				}
				if challenge != "" && challenge == auth.Challenge {
					senderState = sp3.AUTHORIZED
					// Further messages should now be considered as binary packets.
					sendStream = CreateSpoofedStream(addrHost, auth.DestinationAddress)
					defer close(sendStream)

					resp := sp3.ServerMessage{
						Status: sp3.OKAY,
					}
					dat, _ := json.Marshal(resp)
					if err = c.WriteMessage(websocket.TextMessage, dat); err != nil {
						break
					}
					log.Printf("Authorized %v to send to %v.", r.RemoteAddr, auth.DestinationAddress)
				} else {
					log.Println("Bad Challenge from", r.RemoteAddr, " expected ", auth.Challenge, " but got ", challenge)
					resp := sp3.ServerMessage{
						Status: sp3.UNAUTHORIZED,
					}
					dat, _ := json.Marshal(resp)
					c.WriteMessage(websocket.TextMessage, dat)
					break
				}
				continue
			} else if senderState == sp3.AUTHORIZED && msgType == websocket.BinaryMessage {
				// Main forwarding loop.
				sendStream <- msg
				continue
			}
			// Else - unexpected message
			log.Println("Unexpected message", msg)
			break
		}
	})
}

func NewServer(conf Config) *Server {
	server := &Server{
		config:       conf,
		destinations: make(map[string]*websocket.Conn),
		clientHosts:  make(map[string]*websocket.Conn),
	}

	addr := fmt.Sprintf("0.0.0.0:%d", conf.Port)
	mux := http.NewServeMux()
	mux.Handle("/sp3", SocketHandler(server))
	// By default serve a demo site.
	mux.Handle("/client/", http.StripPrefix("/client/", http.FileServer(http.Dir("../demo"))))
	mux.Handle("/ip.js", IPHandler(server))
	mux.Handle("/pathreflection.json", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, conf.PathReflectionFile)
	}))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/client/", 301)
	}))

	webServer := &http.Server{Addr: addr, Handler: mux}

	server.webServer = *webServer
	return server
}

func (s *Server) Serve() error {
	return s.webServer.ListenAndServe()
}
