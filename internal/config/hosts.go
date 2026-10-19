package config

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// NormalizeAllowedHosts accepts exact DNS names or pasted HTTP(S) addresses.
// It never resolves DNS or trusts an entire provider's domain: resolving to a
// local IP is exactly what a DNS-rebinding domain can do, too.
func NormalizeAllowedHosts(values []string) ([]string, error) {
	if len(values) > 64 {
		return nil, fmt.Errorf("use at most 64 network names")
	}
	hosts := []string{}
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		address := value
		if !strings.Contains(address, "://") {
			address = "http://" + address
		}
		u, err := url.Parse(address)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil ||
			(u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
			return nil, fmt.Errorf("enter an exact hostname or HTTP(S) address without a path, login, query or fragment")
		}
		if port := u.Port(); port != "" {
			n, err := strconv.Atoi(port)
			if err != nil || n < 1 || n > 65535 {
				return nil, fmt.Errorf("invalid port in network address")
			}
		}
		name := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
		if name == "" || len(name) > 253 {
			return nil, fmt.Errorf("enter a hostname, such as my-goblin.duckdns.org")
		}
		if net.ParseIP(name) == nil {
			for _, label := range strings.Split(name, ".") {
				if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
					return nil, fmt.Errorf("invalid hostname %q; use exact names, without wildcards", name)
				}
				for _, c := range label {
					if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
						return nil, fmt.Errorf("invalid hostname %q; use exact names, without wildcards", name)
					}
				}
			}
		}
		if !seen[name] {
			hosts = append(hosts, name)
			seen[name] = true
		}
	}
	return hosts, nil
}
