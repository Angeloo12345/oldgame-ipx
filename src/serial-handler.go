package main

// SERIAL relay (oldgame): raw byte-pipe pro hry bez IPX, co jedou přes null-modem
// (Wacky Wheels, IndyCar…). Emulátor (patchnutý SDLnetTCP → WS most) otevře
// ws://relay/serial/<room>; tady jen přeposíláme bajty druhému peerovi v místnosti.
// Žádné IPX parsování — čistý transparent pipe. /ipx/ se NEdotýká.

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

type SerialHandler struct {
	rooms sync.Map
}

type SerialRoom struct {
	clients *sync.Map
}

var serialHandler = &SerialHandler{}

func serialGetRoom(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) < 3 || parts[1] != "serial" {
		return ""
	}
	return parts[2]
}

func (h *SerialHandler) OnConnect(conn *websocket.Conn, room string) {
	r, _ := h.rooms.LoadOrStore(room, &SerialRoom{clients: &sync.Map{}})
	clients := r.(*SerialRoom).clients
	prev, loaded := clients.Swap(conn.RemoteAddr().String(), conn)
	if loaded {
		prev.(*websocket.Conn).Close()
	}
}

func (h *SerialHandler) OnMessage(conn *websocket.Conn, room string, data []byte) {
	r, ok := h.rooms.Load(room)
	if !ok {
		return
	}
	clients := r.(*SerialRoom).clients
	// přepošli bajty VŠEM ostatním v místnosti (pro 1v1 = druhému peerovi)
	clients.Range(func(_, dest interface{}) bool {
		if conn != dest {
			dest.(*websocket.Conn).WriteMessage(websocket.BinaryMessage, data)
		}
		return true
	})
}

func (h *SerialHandler) OnClose(conn *websocket.Conn, room string) {
	r, ok := h.rooms.Load(room)
	if !ok {
		return
	}
	clients := r.(*SerialRoom).clients
	clients.Delete(conn.RemoteAddr().String())
	empty := true
	clients.Range(func(_, _ interface{}) bool { empty = false; return false })
	if empty {
		h.rooms.Delete(room)
	}
}

// Stats: JSON {room: pocet} pro /serial-rooms (počty hráčů u sériových her).
func (h *SerialHandler) Stats() string {
	var sb strings.Builder
	sb.WriteString("{")
	first := true
	h.rooms.Range(func(k, v interface{}) bool {
		n := 0
		v.(*SerialRoom).clients.Range(func(_, _ interface{}) bool { n++; return true })
		if !first {
			sb.WriteString(",")
		}
		first = false
		sb.WriteString(fmt.Sprintf("%q:%d", k.(string), n))
		return true
	})
	sb.WriteString("}")
	return sb.String()
}

func serialWebSocket(w http.ResponseWriter, r *http.Request) {
	room := serialGetRoom(r.URL.Path)
	if len(room) == 0 {
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	serialHandler.OnConnect(conn, room)
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}
		serialHandler.OnMessage(conn, room, data)
	}
	serialHandler.OnClose(conn, room)
	conn.Close()
}
