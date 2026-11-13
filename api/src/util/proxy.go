package util

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync/atomic"
)

// outboundProxy is the HTTP proxy used for the domains in PROXY_DOMAINS.
// Set by the config package from PROXY_ADDRESS and PROXY_DOMAINS.
type outboundProxy struct {
	url *url.URL
	// dialAddress is the proxy's host:port, as http.Transport dials it.
	dialAddress string
	domains     []string
}

var proxy atomic.Pointer[outboundProxy]

// SetProxy sets the HTTP proxy used to reach domains and their subdomains.
// The address can have a username and password, like http://user:pass@gluetun:8888.
// An empty address disables it.
func SetProxy(address string, domains []string) error {
	if address == "" {
		proxy.Store(nil)
		return nil
	}

	parsed, err := url.Parse(address)
	if err != nil {
		// The address isn't echoed, as it can carry the proxy's password.
		return fmt.Errorf("invalid proxy address: %s", strings.ReplaceAll(err.Error(), address, "<PROXY_ADDRESS>"))
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("invalid proxy address '%s': the scheme must be http or https", parsed.Redacted())
	}
	if parsed.Hostname() == "" {
		return fmt.Errorf("invalid proxy address '%s': missing host", parsed.Redacted())
	}

	var cleanDomains []string
	for _, domain := range domains {
		domain = strings.Trim(strings.ToLower(strings.TrimSpace(domain)), ".")
		if domain != "" {
			cleanDomains = append(cleanDomains, domain)
		}
	}
	if len(cleanDomains) == 0 {
		return fmt.Errorf("a proxy address is set, but no domain to use it with")
	}

	port := parsed.Port()
	if port == "" {
		port = map[string]string{"http": "80", "https": "443"}[parsed.Scheme]
	}

	proxy.Store(&outboundProxy{
		url:         parsed,
		dialAddress: net.JoinHostPort(parsed.Hostname(), port),
		domains:     cleanDomains,
	})
	return nil
}

// ProxyFor returns the proxy URL to reach host, or nil if host should be reached directly.
func ProxyFor(host string) *url.URL {
	p := proxy.Load()
	if p == nil {
		return nil
	}
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	for _, domain := range p.domains {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return p.url
		}
	}
	return nil
}

// ProxyEnabled tells if a proxy is configured.
func ProxyEnabled() bool {
	return proxy.Load() != nil
}

// isProxyDialAddress tells if address is the configured proxy's, which can be private.
func isProxyDialAddress(address string) bool {
	p := proxy.Load()
	return p != nil && p.dialAddress == address
}
