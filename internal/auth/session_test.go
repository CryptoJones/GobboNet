package auth

import (
	"net/http/httptest"
	"testing"
	"time"
)

func fp(remote, ua string) string {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = remote
	r.Header.Set("User-Agent", ua)
	return ClientFingerprint(r)
}

// One browser on this machine is one client however it arrives. A Go listener
// on 0.0.0.0 is dual-stack, so the same tab can come in as 127.0.0.1 and ::1,
// and the browser string can change under an extension or DevTools. Binding a
// loopback session to either signed people out for nothing: nothing sniffed off
// the network can arrive from loopback.
func TestFingerprintLoopbackIsOneClient(t *testing.T) {
	want := fp("127.0.0.1:5000", "Chrome A")
	for _, c := range []struct{ remote, ua string }{
		{"127.0.0.1:6000", "Chrome A"},
		{"[::1]:7000", "Chrome A"},
		{"127.0.0.1:8000", "Chrome B (UA changed by an extension)"},
		{"[::ffff:127.0.0.1]:9000", ""},
		{"127.0.0.5:9001", "Chrome A"},
	} {
		if got := fp(c.remote, c.ua); got != want {
			t.Errorf("%s %q is this machine but fingerprints as %q, not %q", c.remote, c.ua, got, want)
		}
	}
	if want != LoopbackClient {
		t.Errorf("loopback fingerprint = %q, want %q", want, LoopbackClient)
	}
}

// From the network the binding stays exactly as it was: address AND browser.
func TestFingerprintNetworkStaysBound(t *testing.T) {
	phone := fp("192.168.1.5:5000", "Safari")
	if fp("192.168.1.5:6000", "Safari") != phone {
		t.Error("same device, new connection: must be the same client")
	}
	if fp("[::ffff:192.168.1.5]:6000", "Safari") != phone {
		t.Error("an IPv4-mapped address is the IPv4 address it carries")
	}
	if fp("192.168.1.6:5000", "Safari") == phone {
		t.Error("another device on the LAN must not share the session")
	}
	if fp("192.168.1.5:5000", "Firefox") == phone {
		t.Error("another browser at the same address must not share the session")
	}
	if fp("[fe80::1%eth0]:5000", "Safari") == phone || fp("[2001:db8::5]:5000", "Safari") == phone {
		t.Error("an IPv6 address is a different client from the IPv4 one")
	}
	if fp("192.168.1.5:5000", "Safari") == LoopbackClient {
		t.Error("a LAN address is not loopback")
	}
}

func TestCheckSaysWhy(t *testing.T) {
	s := NewSessionStore(12)
	tok, err := s.Create("client-a")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		token, client string
		ok            bool
		why           string
	}{
		{tok, "client-a", true, RejectNone},
		{"", "client-a", false, RejectNoToken},
		{"never-issued", "client-a", false, RejectUnknown},
		{tok, "client-b", false, RejectOtherClient},
	} {
		ok, why := s.Check(c.token, c.client)
		if ok != c.ok || why != c.why {
			t.Errorf("Check(%q, %q) = %v, %q; want %v, %q", c.token, c.client, ok, why, c.ok, c.why)
		}
		if s.Validate(c.token, c.client) != c.ok {
			t.Errorf("Validate disagrees with Check for %q", c.token)
		}
	}
	s.mu.Lock()
	sess := s.sessions[tok]
	sess.expiry = time.Now().Add(-time.Second)
	s.sessions[tok] = sess
	s.mu.Unlock()
	if ok, why := s.Check(tok, "client-a"); ok || why != RejectExpired {
		t.Errorf("expired: got %v, %q", ok, why)
	}
	if ok, why := s.Check(tok, "client-a"); ok || why != RejectUnknown {
		t.Errorf("an expired session is dropped, so next time it is unknown: got %v, %q", ok, why)
	}
}
