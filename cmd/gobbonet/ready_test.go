package main

import (
	"bytes"
	"errors"
	"github.com/ElodineOfficial/GobboNet/internal/config"
	"github.com/ElodineOfficial/GobboNet/internal/server"
	"strings"
	"testing"
)

func TestReadyPanelAddressesReflectActualListener(t *testing.T) {
	addrs := []server.LANAddr{
		{IP: "10.8.0.2", Iface: "VPN", Kind: server.LANTunnel},
		{IP: "192.168.1.20", Iface: "Wi-Fi", Kind: server.LANPhysical},
		{IP: "192.168.2.20", Iface: "Ethernet", Kind: server.LANPhysical},
	}
	var b bytes.Buffer
	printReadyPanel(&b, config.Config{}, &server.Bind{Host: "0.0.0.0", Port: 9067}, addrs)
	out := b.String()
	for _, want := range []string{"http://127.0.0.1:9067/", "Wi-Fi\n      http://192.168.1.20:9067/", "Ethernet\n      http://192.168.2.20:9067/", "http://10.8.0.2:9067/"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %s", want, out)
		}
	}
	if strings.Index(out, "192.168.1.20") > strings.Index(out, "10.8.0.2") {
		t.Fatal("VPN appeared ahead of phone addresses")
	}
	b.Reset()
	printReadyPanel(&b, config.Config{}, &server.Bind{Host: "127.0.0.1", Port: 9067, FellBack: true, WideErr: errors.New("denied")}, addrs)
	if !strings.Contains(b.String(), "LAN ACCESS IS OFF") || strings.Contains(b.String(), "192.168.1.20") {
		t.Fatal("fallback advertised an unreachable LAN address")
	}
	b.Reset()
	printReadyPanel(&b, config.Config{}, &server.Bind{Host: "192.168.1.20", Port: 9067}, addrs[1:2])
	if strings.Contains(b.String(), "127.0.0.1") {
		t.Fatal("specific interface bind advertised unbound loopback")
	}
	// 1.7.7 #7: a pinned address keeps loopback beside it, and this PC is
	// then told to use loopback, as it is everywhere else.
	b.Reset()
	printReadyPanel(&b, config.Config{}, &server.Bind{Host: "192.168.1.20", Port: 9067, Loopback: "127.0.0.1"}, addrs[1:2])
	if !strings.Contains(b.String(), "ON THIS PC\n    http://127.0.0.1:9067/") {
		t.Fatalf("pinned bind with loopback served did not offer it on this PC:\n%s", b.String())
	}
}

// panelFor renders the ready panel as a given platform and console would.
func panelFor(t *testing.T, goos string, onTerminal bool, bind *server.Bind) string {
	t.Helper()
	oldGOOS, oldTerm := panelGOOS, panelOnTerminal
	panelGOOS, panelOnTerminal = goos, func() bool { return onTerminal }
	defer func() { panelGOOS, panelOnTerminal = oldGOOS, oldTerm }()
	var b bytes.Buffer
	cfg := config.Default()
	cfg.Path = "/home/me/.config/gobbonet/config.toml"
	printReadyPanel(&b, cfg, bind, []server.LANAddr{{IP: "192.168.1.20", Iface: "wlan0", Kind: server.LANPhysical}})
	return b.String()
}

// Issue #63 and its LAN follow-up: the panel was the Windows text on every
// platform. Linux was told to run setup-lan.bat, and a server with no terminal
// (desktop launcher, login start) to keep its window open and press Ctrl+C.
func TestReadyPanelIsPerPlatform(t *testing.T) {
	lan := &server.Bind{Host: "0.0.0.0", Port: 9066}
	fell := &server.Bind{Host: "127.0.0.1", Port: 9066, FellBack: true, WideErr: errors.New("denied")}

	win := panelFor(t, "windows", true, lan) + panelFor(t, "windows", false, fell)
	for _, want := range []string{"setup-lan.bat", "Keep this window open", "Ctrl+C", "excludedportrange"} {
		if !strings.Contains(win, want) {
			t.Errorf("Windows panel lost %q:\n%s", want, win)
		}
	}
	if strings.Contains(win, "pkill") {
		t.Errorf("Windows panel told to use pkill:\n%s", win)
	}

	for _, onTerminal := range []bool{true, false} {
		linux := panelFor(t, "linux", onTerminal, lan) + panelFor(t, "linux", onTerminal, fell)
		for _, bad := range []string{"setup-lan.bat", "netsh", "excludedportrange", "Administrator"} {
			if strings.Contains(linux, bad) {
				t.Errorf("Linux panel (terminal=%v) mentions %q:\n%s", onTerminal, bad, linux)
			}
		}
		if !strings.Contains(linux, "gobbonet doctor") {
			t.Errorf("Linux panel (terminal=%v) does not point at gobbonet doctor:\n%s", onTerminal, linux)
		}
	}
	term := panelFor(t, "linux", true, lan)
	if !strings.Contains(term, "Ctrl+C") || strings.Contains(term, "pkill") {
		t.Errorf("Linux in a terminal should keep the Ctrl+C advice:\n%s", term)
	}
	bg := panelFor(t, "linux", false, lan)
	if strings.Contains(bg, "Keep this window open") || strings.Contains(bg, "Ctrl+C") ||
		!strings.Contains(bg, "no terminal attached") || !strings.Contains(bg, "pkill -x gobbonet") {
		t.Errorf("Linux with no terminal should say how to stop it instead:\n%s", bg)
	}
}

// The phone address must be typed with its scheme: without it iOS Safari
// tries https, which this server cannot answer.
func TestReadyPanelSaysToTypeHTTP(t *testing.T) {
	for _, goos := range []string{"windows", "linux"} {
		out := panelFor(t, goos, true, &server.Bind{Host: "0.0.0.0", Port: 9066})
		if !strings.Contains(out, "http:// included") || !strings.Contains(out, "http://192.168.1.20:9066/") {
			t.Errorf("%s panel does not tell the user to type http://:\n%s", goos, out)
		}
	}
}
