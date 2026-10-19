package server

import (
	"errors"
	"net"
	"sync"
)

// joinedListener serves several sockets as one net.Listener, so http.Server
// can Serve a pinned address and its loopback companion together (see
// Bind.Loopback) without the caller managing two servers.
//
// Each socket is accepted on its own goroutine and handed over through one
// channel. Close closes every socket; Accept then reports net.ErrClosed, which
// is what http.Server expects in order to return cleanly.
type joinedListener struct {
	primary net.Listener
	all     []net.Listener

	conns chan acceptResult
	done  chan struct{}
	once  sync.Once
}

type acceptResult struct {
	conn net.Conn
	err  error
}

func joinListeners(primary net.Listener, others ...net.Listener) net.Listener {
	j := &joinedListener{
		primary: primary,
		all:     append([]net.Listener{primary}, others...),
		conns:   make(chan acceptResult),
		done:    make(chan struct{}),
	}
	for _, l := range j.all {
		go j.pump(l)
	}
	return j
}

func (j *joinedListener) pump(l net.Listener) {
	for {
		c, err := l.Accept()
		if err != nil && errors.Is(err, net.ErrClosed) {
			return
		}
		select {
		case j.conns <- acceptResult{c, err}:
		case <-j.done:
			if c != nil {
				c.Close()
			}
			return
		}
		if err != nil {
			// A non-closing error (EMFILE, say) is passed up once for
			// http.Server to back off on, and the loop carries on: it retries
			// temporary errors itself, and only a closed listener ends it.
			var ne net.Error
			if !errors.As(err, &ne) {
				return
			}
		}
	}
}

func (j *joinedListener) Accept() (net.Conn, error) {
	select {
	case r := <-j.conns:
		return r.conn, r.err
	case <-j.done:
		return nil, net.ErrClosed
	}
}

func (j *joinedListener) Close() error {
	var first error
	j.once.Do(func() {
		close(j.done)
		for _, l := range j.all {
			if err := l.Close(); err != nil && first == nil {
				first = err
			}
		}
	})
	return first
}

// Addr is the primary socket's: the address the user chose.
func (j *joinedListener) Addr() net.Addr { return j.primary.Addr() }
