package config

import (
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

// Roadmap 1.7.7 #7: the CONFIG panel can choose the address and port phones
// connect to. These are the pieces of that which belong to the config file.

// Ports the panel may save.
//
// On Windows the bounds are launch.bat's and setup-lan.bat's, which both
// quietly replace anything outside them with 9066: below 1024 is reserved, and
// above 32767 is the range Windows hands out to outbound connections (see the
// listen_port comment in DefaultTOML). A port the Windows launcher would
// silently discard is a port the panel must not offer, or saving it would look
// like a feature that does not work.
//
// Elsewhere the Linux setup wizard's bounds apply (installer-linux/
// gobbonet-setup.py): 1024-65535. A Linux user who chose 40000 there must not
// find the panel refusing to save anything until they move off it.
const (
	MinPanelPort          = 1024
	MaxPanelPort          = 32767 // Windows
	MaxPanelPortElsewhere = 65535
)

// MaxPanelPortFor is the highest port the panel offers on goos.
func MaxPanelPortFor(goos string) int {
	if goos == "windows" {
		return MaxPanelPort
	}
	return MaxPanelPortElsewhere
}

// EnginePorts are the ports the panel will not hand to the web server: the
// defaults for llama.cpp (11437) and the embedding server (11436), which the
// Linux setup wizard also refuses, plus whatever ports llm_url and embed_url
// name on this machine. Taking one would leave the web server and the engine
// fighting over it on the next start.
func (c *Config) EnginePorts() map[int]bool {
	ports := map[int]bool{11436: true, 11437: true}
	for _, raw := range []string{c.LLMURL, c.EmbedURL} {
		u, err := url.Parse(raw)
		if err != nil || u.Port() == "" {
			continue
		}
		h := strings.ToLower(u.Hostname())
		if h != "127.0.0.1" && h != "localhost" && h != "::1" {
			continue
		}
		if n, err := strconv.Atoi(u.Port()); err == nil {
			ports[n] = true
		}
	}
	return ports
}

// SavedListen is what the config file says, as opposed to what is running.
// The two differ after a save (it applies on the next start) and when an
// environment variable or a --host/--port flag overrides the file.
type SavedListen struct {
	Host string
	Port int
}

// ReadListen reads listen_host and listen_port from the file at path alone:
// no environment, no flags. Absent keys read as the defaults, as Load does.
func ReadListen(path string) (SavedListen, error) {
	v := struct {
		Host string `toml:"listen_host"`
		Port int    `toml:"listen_port"`
	}{DefaultListenHost, DefaultListenPort}
	// Decoded into a struct holding only these two keys: every other key in
	// the file is undecoded, and undecoded is fine here -- Load is what
	// polices unknown keys.
	if _, err := toml.DecodeFile(path, &v); err != nil {
		return SavedListen{}, err
	}
	return SavedListen{Host: v.Host, Port: v.Port}, nil
}

// ListenEnvOverrides names the environment variables that are overriding the
// file's listen_host or listen_port right now, in the order applyEnv reads
// them. A save to the file does nothing while one of these is set, and the
// panel says so rather than reporting a save that cannot take effect.
func ListenEnvOverrides() []string {
	var out []string
	for _, key := range []string{"LISTEN_HOST", "LISTEN_PORT"} {
		for _, prefix := range []string{"GOBBONET_", "GEMMA_"} {
			if os.Getenv(prefix+key) != "" {
				out = append(out, prefix+key)
				break
			}
		}
	}
	return out
}
