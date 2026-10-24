package server

import (
	"crypto/sha256"
	"log"
	"net/http"
	"sync"

	"github.com/ElodineOfficial/GobboNet/internal/auth"
)

// rejectLog remembers which refused sessions it has already reported, so the
// server's log says once, per token and reason, why a browser was asked to
// sign in again -- the answer a "signed in, then asked again" report needs and
// had no way to give. Only a hash of the token is kept, never the token, and
// nothing is sent anywhere: this is the console window's own log.
type rejectLog struct {
	mu    sync.Mutex
	seen  map[[32]byte]struct{}
	order [][32]byte
}

const rejectLogSize = 256

// first reports whether this token and reason are new to the log.
func (l *rejectLog) first(token, why string) bool {
	key := sha256.Sum256([]byte(why + "\x00" + token))
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.seen == nil {
		l.seen = make(map[[32]byte]struct{})
	}
	if _, dup := l.seen[key]; dup {
		return false
	}
	l.seen[key] = struct{}{}
	l.order = append(l.order, key)
	if len(l.order) > rejectLogSize {
		delete(l.seen, l.order[0])
		l.order = l.order[1:]
	}
	return true
}

// noteRejected logs why a presented session was refused, the first time.
func (s *Server) noteRejected(r *http.Request, token, why string) {
	if why == auth.RejectNone || why == auth.RejectNoToken || !s.rejected.first(token, why) {
		return
	}
	switch why {
	case auth.RejectUnknown:
		log.Printf("[auth] %s %s: a session this server did not issue was refused, so that tab is asked to sign in. "+
			"Usual causes: it is from before GobboNet restarted (sessions live in memory), or another GobboNet "+
			"on this address but a different port signed this browser in -- browsers share cookies across ports.",
			r.Method, r.URL.Path)
	case auth.RejectExpired:
		log.Printf("[auth] %s %s: a session reached its time limit (session_ttl_hours = %.0f) and has to sign in again.",
			r.Method, r.URL.Path, s.sessions.TTL().Hours())
	case auth.RejectOtherClient:
		log.Printf("[auth] %s %s: a signed-in session arrived from a different network address or browser than it "+
			"signed in from, so it has to sign in again. Sessions from the network are bound to both; from this PC "+
			"(127.0.0.1, ::1) they are not.", r.Method, r.URL.Path)
	}
}
