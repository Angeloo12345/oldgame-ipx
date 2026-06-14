package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

// port = synthetic IPX "server" node address (NOT the TCP listen port).
// Keep it constant so the IPX registration reply stays identical to upstream.
const port = "1900"

var upgrader = websocket.Upgrader{
	Subprotocols: []string{"binary"},
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var ipxHandler = &IpxHandler{
	serverAddress: "127.0.0.1:" + port,
}

func getRoom(r *http.Request) string {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		return ""
	}
	if parts[1] != "ipx" {
		return ""
	}
	return parts[2]
}

func ipxWebSocket(w http.ResponseWriter, r *http.Request) {
	room := getRoom(r)
	if len(room) == 0 {
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	ipxHandler.OnConnect(conn, room)
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}

		ipxHandler.OnMessage(conn, room, data)
	}

	ipxHandler.OnClose(conn, room)
	conn.Close()
}

var cert string
var key string

func main() {
	flag.StringVar(&cert, "c", "", ".cert file")
	flag.StringVar(&key, "k", "", ".key file")
	flag.Parse()

	// Listen port: PaaS (Render/Koyeb/…) inject $PORT and terminate TLS for us.
	// Fallback to 1900 = original behaviour for a bare VPS / local run.
	listen := os.Getenv("PORT")
	if listen == "" {
		listen = port
	}

	http.HandleFunc("/ipx/", ipxWebSocket)
	// /serial/ = raw byte-pipe pro null-modem hry bez IPX (Wacky Wheels, IndyCar…)
	http.HandleFunc("/serial/", serialWebSocket)
	// /rooms = monitoring: JSON {místnost: počet_klientů} (kolik hráčů je v které místnosti)
	http.HandleFunc("/rooms", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Write([]byte(ipxHandler.Stats()))
	})
	// /serial-rooms = počty hráčů u sériových her
	http.HandleFunc("/serial-rooms", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Write([]byte(serialHandler.Stats()))
	})
	// Root = health check (for the PaaS) + keep-alive ping target (anti-sleep).
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("oldgame ipx relay ok\n"))
	})

	log.Println("oldgame ipx relay listening on :" + listen)
	if len(cert) == 0 || len(key) == 0 {
		log.Println("TLS off (PaaS terminates TLS, or local plain ws)")
		if err := http.ListenAndServe(":"+listen, nil); err != nil {
			log.Fatal(err)
		}
	} else if err := http.ListenAndServeTLS(":"+listen, cert, key, nil); err != nil {
		log.Fatal(err)
	}
}
