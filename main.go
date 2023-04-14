package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"time"

	"kioken/pkg/server"

	"github.com/gorilla/websocket"
)

type serverStats struct {
	NumConnPerSec int `json:"numConnPerSec"` // number of connection made per sec
	NumActiveConn int `json:"numActiveConn"` // number of active conn
	NumIpPerSec   int `json:"numIpPerSec"`   // number of unique IP per sec
	NumTotalConn  int `json:"numTotalConn"`  // total conn ever made
}

var (
	numListner = flag.Int("listners", 5, "Num of conection acceptor loops")
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	flag.Parse()
	// Create a new TCP server and start it
	tcpServer := server.NewTcpServer(":1234", *numListner)
	if err := tcpServer.Start(); err != nil {
		log.Fatalf("Failed to start TCP server: %s", err)
	}

	// Create a new WebSocket server
	http.HandleFunc("/ws", handleWebSocket(tcpServer))

	// Serve the web page on port 80
	http.Handle("/", http.FileServer(http.Dir("web")))
	log.Fatal(http.ListenAndServe(":80", nil))
}

func handleWebSocket(tcpServer *server.TcpServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Upgrade the HTTP connection to WebSocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Failed to upgrade HTTP connection to WebSocket: %s", err)
			return
		}

		// Start a goroutine to send the TCP server stats to the client every second
		go func() {
			ticker := time.NewTicker(time.Second)
			for range ticker.C {
				stats := serverStats{
					NumConnPerSec: tcpServer.GetNumConnPerSec(),
					NumActiveConn: tcpServer.GetNumActiveConn(),
					NumTotalConn:  tcpServer.GetNumTotalConn(),
					NumIpPerSec:   tcpServer.GetIpPerSec(),
				}

				statsJSON, err := json.Marshal(stats)
				if err != nil {
					log.Printf("Failed to marshal server stats to JSON: %s", err)
					continue
				}

				// Send the stats to the client
				if err := conn.WriteMessage(websocket.TextMessage, statsJSON); err != nil {
					log.Printf("Failed to send server stats to client: %s", err)
					return
				}
			}
		}()
	}
}
