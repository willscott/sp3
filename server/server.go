package main

import (
	//"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Server struct {
	sync.Mutex
	webServer http.Server
	config Config

	// the 10 minute timer to clean sending grants
	lastSweepTime     time.Time
}

var upgrader = websocket.Upgrader{}
var destinations = make(map[string]*websocket.Conn)

func SocketHandler(server *Server) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		destinations[r.RemoteAddr] = c
		defer c.Close()
		defer delete(destinations, r.RemoteAddr)
		for {
			_, _, err := c.ReadMessage()
			if err != nil {
				log.Println("read err:", err)
				break
			}
			break
		}
	})
}

func NewServer(conf Config) *Server {
	server := &Server{
		config:     conf,
	}

	addr := fmt.Sprintf("0.0.0.0:%d", conf.port)
	mux := http.NewServeMux()
	mux.Handle("/", SocketHandler(server))
	webServer := &http.Server{Addr: addr, Handler: mux}
	webServer.ListenAndServe()

	server.webServer = *webServer
	return server
}
