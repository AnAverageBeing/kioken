package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"kioken/pkg/server"

	"github.com/gorilla/websocket"
)

type serverStats struct {
	NumConnPerSec string `json:"numConnPerSec"` // number of connection made per sec
	NumActiveConn string `json:"numActiveConn"` // number of active conn
	NumTotalConn  string `json:"numTotalConn"`  // total conn ever made
	InboundMBps   string `json:"inboundMBps"`   // inbound in MBps
}

var (
	numListener = flag.Int("ln", 5, "Num of connection acceptor loops")
	poolSize    = flag.Int("ps", 50000, "Num of workers in pool")
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	flag.Parse()
	// Create a new TCP server and start it
	tcpServer, err := server.NewServer(":1234", *poolSize)
	if err != nil {
		log.Fatalln(err)
	}
	tcpServer.Start(*numListener)

	// Create a new WebSocket server
	http.HandleFunc("/ws", handleWebSocket(tcpServer))

	// Serve the web page on port 80
	http.Handle("/", http.FileServer(http.Dir("web")))
	log.Fatal(http.ListenAndServe(":80", nil))
}

func handleWebSocket(tcpServer *server.Server) http.HandlerFunc {
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
					NumConnPerSec: fmt.Sprintf("%d", tcpServer.GetNumConnRate()),
					NumActiveConn: fmt.Sprintf("%d", tcpServer.GetNumActiveConn()),
					NumTotalConn:  fmt.Sprintf("%d", tcpServer.GetNumConnCount()),
					InboundMBps:   fmt.Sprintf("%f", tcpServer.GetInDataRate()),
				}

				statsJSON, err := json.Marshal(stats)
				if err != nil {
					log.Printf("Failed to marshal server stats to JSON: %s", err)
					continue
				}

				// Send the stats to the client
				if err := conn.WriteMessage(websocket.TextMessage, statsJSON); err != nil {
					// log.Printf("Failed to send server stats to client: %s", err)
					return
				}
			}
		}()
	}
}
