package config

import (
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestNormalizeAllowedHosts(t *testing.T) {
	got, err := NormalizeAllowedHosts([]string{" https://My-Goblin.DuckDNS.org:9066/ ", "my-goblin.duckdns.org", "Desktop.Tail123.ts.net.", "desktop"})
	want := []string{"my-goblin.duckdns.org", "desktop.tail123.ts.net", "desktop"}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, %v; want %v", got, err, want)
	}
	cfg := Config{AllowedHosts: got}
	for _, h := range []string{"my-goblin.duckdns.org:9066", "MY-GOBLIN.DUCKDNS.ORG.:9066", "desktop.tail123.ts.net", "desktop:9066"} {
		if !cfg.HostAllowed(h) {
			t.Errorf("rejected configured name %s", h)
		}
	}
	for _, h := range []string{"other.duckdns.org", "other.tail123.ts.net", "my-goblin.duckdns.org.evil.test"} {
		if cfg.HostAllowed(h) {
			t.Errorf("trusted unrelated name %s", h)
		}
	}
	for _, value := range []string{"", "*", "*.duckdns.org", "https://name.test/path", "https://user:pass@name.test", "name.test?x=y", "name.test#fragment", "ftp://name.test", "bad name.test", "bad_name.test", "a..test", "-a.test", "a-.test", "name.test:0", "name.test:65536", "name.test:no", strings.Repeat("a", 64) + ".test"} {
		if _, err := NormalizeAllowedHosts([]string{value}); err == nil {
			t.Errorf("accepted invalid value %q", value)
		}
	}
	if _, err := NormalizeAllowedHosts(make([]string, 65)); err == nil {
		t.Error("accepted excessive list")
	}
}

func TestHostAllowedConfiguredTrailingDot(t *testing.T) {
	cfg := Config{AllowedHosts: []string{"My-Goblin.DuckDNS.org."}}
	if !cfg.HostAllowed("my-goblin.duckdns.org:9066") {
		t.Fatal("configured fully qualified name was rejected")
	}
}

func TestNetworkHostsConcurrentConfigSaves(t *testing.T) {
	path := writeConfig(t, "allowed_hosts = []\nidle_standdown_minutes = 5\n[ui]\ntheme = \"green\"\n")
	for i := 0; i < 20; i++ {
		var wg sync.WaitGroup
		for key, value := range map[string]string{"allowed_hosts": "my-goblin.duckdns.org desktop.tail123.ts.net", "idle_standdown_minutes": "12"} {
			wg.Add(1)
			go func(key, value string) {
				defer wg.Done()
				if err := Set(path, key, value); err != nil {
					t.Error(err)
				}
			}(key, value)
		}
		wg.Wait()
		cfg, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		if len(cfg.AllowedHosts) != 2 || cfg.IdleStandDownMinutes != 12 {
			t.Fatal("concurrent settings save lost a value")
		}
		if err := Set(path, "allowed_hosts", ""); err != nil {
			t.Fatal(err)
		}
		if err := Set(path, "idle_standdown_minutes", "5"); err != nil {
			t.Fatal(err)
		}
	}
}
