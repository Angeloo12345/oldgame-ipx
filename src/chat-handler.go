package main

// CHAT (oldgame): chat při hře v místnosti — okýnko v play.php.
// play.php pošle GET /msg?room=R&nick=N&text=T a polluje GET /msgs?room=R&since=SEQ.
// Ring buffer posledních 50 zpráv per místnost (sanitizeNick je v presence-handler.go).

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type chatMsg struct {
	seq  int
	nick string
	text string
}
type chatRoom struct {
	mu   sync.Mutex
	seq  int
	msgs []chatMsg
}

var chatRooms sync.Map // room -> *chatRoom

func sanitizeText(s string) string {
	if len(s) > 240 {
		s = s[:240]
	}
	var b strings.Builder
	for _, r := range s {
		if r >= 32 && r != '"' && r != '\\' {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

// /msg?room=R&nick=N&text=T — pošle zprávu do chatu místnosti
func chatPostHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	q := r.URL.Query()
	room, nick, text := q.Get("room"), sanitizeNick(q.Get("nick")), sanitizeText(q.Get("text"))
	if room == "" || text == "" {
		w.Write([]byte("0"))
		return
	}
	crI, _ := chatRooms.LoadOrStore(room, &chatRoom{})
	cr := crI.(*chatRoom)
	cr.mu.Lock()
	cr.seq++
	cr.msgs = append(cr.msgs, chatMsg{cr.seq, nick, text})
	if len(cr.msgs) > 50 {
		cr.msgs = cr.msgs[len(cr.msgs)-50:]
	}
	cr.mu.Unlock()
	w.Write([]byte("ok"))
}

// /msgs?room=R&since=S — JSON [{"seq":..,"nick":..,"text":..}] zpráv se seq>since
func chatGetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	q := r.URL.Query()
	room := q.Get("room")
	since, _ := strconv.Atoi(q.Get("since"))
	var sb strings.Builder
	sb.WriteString("[")
	if crI, ok := chatRooms.Load(room); ok {
		cr := crI.(*chatRoom)
		cr.mu.Lock()
		first := true
		for _, m := range cr.msgs {
			if m.seq <= since {
				continue
			}
			if !first {
				sb.WriteString(",")
			}
			first = false
			sb.WriteString(fmt.Sprintf("{\"seq\":%d,\"nick\":%q,\"text\":%q}", m.seq, m.nick, m.text))
		}
		cr.mu.Unlock()
	}
	sb.WriteString("]")
	w.Write([]byte(sb.String()))
}
