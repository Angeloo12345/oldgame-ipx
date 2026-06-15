package main

// PRESENCE (oldgame): jména hráčů v místnosti — nezávislé na WS handlerech.
// play.php každých pár sekund pošle GET /presence?room=R&id=ID&nick=N (heartbeat),
// a GET /players?room=R vrátí [{id,nick}] (v TTL). Tak hra ukáže „Soupeř: <jméno>".

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type presEntry struct {
	nick string
	seen int64
}

var presence sync.Map // room(string) -> *sync.Map(id -> *presEntry)

const presenceTTL = 12 // s — po této době bez heartbeatu hráč zmizí

func sanitizeNick(s string) string {
	if len(s) > 16 {
		s = s[:16]
	}
	var b strings.Builder
	for _, r := range s {
		if r >= 32 && r != '"' && r != '\\' && r != '<' && r != '>' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// /presence?room=R&id=ID&nick=N — registrace/heartbeat hráče
func presenceHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	q := r.URL.Query()
	room, id, nick := q.Get("room"), q.Get("id"), sanitizeNick(q.Get("nick"))
	if room == "" || id == "" {
		w.Write([]byte("0"))
		return
	}
	rm, _ := presence.LoadOrStore(room, &sync.Map{})
	rm.(*sync.Map).Store(id, &presEntry{nick: nick, seen: time.Now().Unix()})
	w.Write([]byte("ok"))
}

// /players?room=R — JSON [{"id":..,"nick":..}] živých hráčů (v TTL)
func playersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	room := r.URL.Query().Get("room")
	var sb strings.Builder
	sb.WriteString("[")
	if rmI, ok := presence.Load(room); ok {
		rm := rmI.(*sync.Map)
		now := time.Now().Unix()
		first := true
		rm.Range(func(k, v interface{}) bool {
			e := v.(*presEntry)
			if now-e.seen > presenceTTL {
				rm.Delete(k) // úklid starých
				return true
			}
			if !first {
				sb.WriteString(",")
			}
			first = false
			sb.WriteString(fmt.Sprintf("{\"id\":%q,\"nick\":%q}", k.(string), e.nick))
			return true
		})
	}
	sb.WriteString("]")
	w.Write([]byte(sb.String()))
}
