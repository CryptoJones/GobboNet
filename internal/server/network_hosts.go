package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/ElodineOfficial/GobboNet/internal/config"
	"github.com/ElodineOfficial/GobboNet/internal/httpx"
)

func (s *Server) hostAllowed(host string) bool {
	s.hostsMu.RLock()
	defer s.hostsMu.RUnlock()
	return (&config.Config{AllowedHosts: s.allowedHosts}).HostAllowed(host)
}

// This route sits behind both the Host and session gates. A hostname must be
// approved through an already trusted address, never through the blocked name.
func (s *Server) handleNetworkHosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodPost {
		httpx.Error(w, r, http.StatusMethodNotAllowed, "GET or POST only")
		return
	}
	if r.Method == http.MethodPost {
		// Require a same-origin browser operation even when auth was disabled by
		// the owner. CORS alone does not prevent cross-origin writes. TLS may end
		// at a reverse proxy, so compare the explicit HTTP(S) origin's authority
		// to Host rather than trusting any forwarded headers.
		origin, err := url.Parse(r.Header.Get("Origin"))
		if err != nil || (origin.Scheme != "http" && origin.Scheme != "https") ||
			!strings.EqualFold(origin.Host, r.Host) || origin.User != nil ||
			origin.Path != "" || origin.RawQuery != "" || origin.Fragment != "" {
			httpx.Error(w, r, http.StatusForbidden, "Save network names from GobboNet's CONFIG panel on this address.")
			return
		}
		raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 16<<10))
		var in struct {
			Hosts *[]string `json:"allowed_hosts"`
		}
		if err != nil || json.Unmarshal(raw, &in) != nil || in.Hosts == nil {
			httpx.Error(w, r, http.StatusBadRequest, "allowed_hosts must be a JSON list of hostnames")
			return
		}
		hosts, err := config.NormalizeAllowedHosts(*in.Hosts)
		if err != nil {
			httpx.Error(w, r, http.StatusBadRequest, err.Error())
			return
		}
		if !(&config.Config{AllowedHosts: hosts}).HostAllowed(r.Host) {
			httpx.Error(w, r, http.StatusBadRequest, "Keep the hostname you are using, or connect by IP address before removing it.")
			return
		}
		// Persist before publishing: a failed save must not silently change access.
		s.hostsMu.Lock()
		if s.cfg.Path == "" {
			s.hostsMu.Unlock()
			httpx.Error(w, r, http.StatusConflict, "No config file is available; network names were not changed.")
			return
		}
		err = config.Set(s.cfg.Path, "allowed_hosts", strings.Join(hosts, " "))
		if err == nil {
			s.allowedHosts = hosts
		}
		s.hostsMu.Unlock()
		if err != nil {
			httpx.Error(w, r, http.StatusInternalServerError, "Could not save config.toml; network names were not changed. Check that the file is writable.")
			return
		}
	}
	s.hostsMu.RLock()
	hosts := append([]string{}, s.allowedHosts...)
	s.hostsMu.RUnlock()
	httpx.WriteJSON(w, r, http.StatusOK, map[string]any{"allowed_hosts": hosts})
}
