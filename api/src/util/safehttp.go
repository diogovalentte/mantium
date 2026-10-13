package util

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/diogovalentte/mantium/api/src/errordefs"
)

// allowPrivateURLs tells if URLs chosen by the user can point to non-public addresses.
// Set by the config package from ALLOW_PRIVATE_URLS.
var allowPrivateURLs atomic.Bool

// SetAllowPrivateURLs sets if URLs chosen by the user can point to non-public addresses.
func SetAllowPrivateURLs(allow bool) {
	allowPrivateURLs.Store(allow)
}

// AllowPrivateURLs tells if URLs chosen by the user can point to non-public addresses.
func AllowPrivateURLs() bool {
	return allowPrivateURLs.Load()
}

var (
	unspecifiedIPv4Network = netip.MustParsePrefix("0.0.0.0/8")
	sharedAddressSpace     = netip.MustParsePrefix("100.64.0.0/10") // CGNAT
)

// isPublicAddr tells if an address is a public unicast address.
// It's a variable so tests can treat their local servers as public.
var isPublicAddr = func(addr netip.Addr) bool {
	addr = addr.Unmap()
	return addr.IsValid() &&
		!addr.IsLoopback() &&
		!addr.IsPrivate() &&
		!addr.IsLinkLocalUnicast() &&
		!addr.IsLinkLocalMulticast() &&
		!addr.IsInterfaceLocalMulticast() &&
		!addr.IsMulticast() &&
		!addr.IsUnspecified() &&
		!unspecifiedIPv4Network.Contains(addr) &&
		!sharedAddressSpace.Contains(addr)
}

// PrivateAddressError returns the error of a request to a non-public address that was blocked.
func PrivateAddressError(host string) error {
	return &errordefs.UserError{Message: fmt.Sprintf("Blocked access to a private address (%s). Set ALLOW_PRIVATE_URLS=true in the API to allow it", host)}
}

var safeDialer = &net.Dialer{
	Timeout:   30 * time.Second,
	KeepAlive: 30 * time.Second,
}

// safeDialContext resolves the host and refuses to connect if any of its addresses isn't public, unless private URLs are allowed.
// It checks the address that is actually used to connect, so it also covers redirects and DNS rebinding.
func safeDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if AllowPrivateURLs() {
		return safeDialer.DialContext(ctx, network, address)
	}

	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	addrs, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		if !isPublicAddr(addr) {
			return nil, PrivateAddressError(host)
		}
	}

	var lastErr error
	for _, addr := range addrs {
		conn, err := safeDialer.DialContext(ctx, network, net.JoinHostPort(addr.Unmap().String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// SafeTransport is the transport for requests to URLs chosen by the user, like custom manga pages
// and cover images. It refuses to connect to non-public addresses unless ALLOW_PRIVATE_URLS is true.
var SafeTransport = func() *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = safeDialContext
	return transport
}()

// SafeHTTPClient returns a client for requests to URLs chosen by the user, see SafeTransport.
func SafeHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: SafeTransport,
		Timeout:   timeout,
	}
}

// CheckPublicURL returns an error if the URL host isn't a public address, unless private URLs are allowed.
func CheckPublicURL(ctx context.Context, rawURL string) error {
	if AllowPrivateURLs() {
		return nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	addrs, err := net.DefaultResolver.LookupNetIP(ctx, "ip", parsed.Hostname())
	if err != nil {
		return err
	}
	for _, addr := range addrs {
		if !isPublicAddr(addr) {
			return PrivateAddressError(parsed.Hostname())
		}
	}
	return nil
}
