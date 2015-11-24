package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pborman/uuid"
)

type Server struct {
	sync.Mutex
	upgrader websocket.Upgrader
	webServer http.Server
	config Config
	destinations map[string]*websocket.Conn

	// the 10 minute timer to clean sending grants
	lastSweepTime     time.Time
}

func (s Server) Authorize(hello ClientHello) (challenge string, err error) {
	if (hello.AuthenticationMethod != WEBSOCKET) {
		return "", errors.New("UNSUPPORTED")
	}
	addrParts := strings.Split(hello.DestinationAddress, ":")
	if (len(addrParts) < 1) {
		return "", errors.New("UNSUPPORTED")
	}
	for addr,conn := range s.destinations {
		if (strings.HasPrefix(addr, addrParts[0])) {
			// Found the connection
			challenge := uuid.New()
			if err = conn.WriteMessage(websocket.TextMessage, []byte(challenge)); err != nil {
				return "", err
	    }
			return challenge, nil;
			break
		}
	}

	return "", nil
}

func SocketHandler(server *Server) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := server.upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		server.destinations[r.RemoteAddr] = c
		senderState := CLIENTHELLO
		challenge := ""

		defer c.Close()
		defer delete(server.destinations, r.RemoteAddr)
		for {
			msgType, msg, err := c.ReadMessage()
			if err != nil {
				log.Println("read err:", err)
				break
			}
			if (senderState == CLIENTHELLO && msgType == websocket.TextMessage) {
				hello := ClientHello{}
				err := json.Unmarshal(msg, &hello)
				if err != nil {
					log.Println("Hello err:", err)
					break
				}
				chal, err := server.Authorize(hello);
				if err != nil {
					log.Println("Authorize err:", err)
					break
				}
				challenge = chal
				senderState = HELLORECEIVED
			} else if (senderState == HELLORECEIVED && msgType == websocket.TextMessage) {
				auth := ClientAuthorization{}
				err := json.Unmarshal(msg, &auth)
				if err != nil {
					log.Println("Auth err:", err)
					break
				}
				if challenge != "" && challenge == auth.Challenge {
					senderState = AUTHORIZED
				} else {
					break
				}
			} else if (senderState == AUTHORIZED && msgType == websocket.BinaryMessage) {
				// Main forwarding loop.
			}

			break
		}
	})
}

func NewServer(conf Config) *Server {
	server := &Server{
		config:     conf,
		destinations: make(map[string]*websocket.Conn),
	}

	addr := fmt.Sprintf("0.0.0.0:%d", conf.port)
	mux := http.NewServeMux()
	mux.Handle("/", SocketHandler(server))
	webServer := &http.Server{Addr: addr, Handler: mux}
	webServer.ListenAndServe()

	server.webServer = *webServer
	return server
}
