package util

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"github.com/diogovalentte/mantium/api/src/errordefs"
)

func TestSetProxy(t *testing.T) {
	defer SetProxy("", nil)

	for _, tc := range []struct {
		address string
		domains []string
		wantErr bool
	}{
		{"", nil, false},
		{"http://gluetun:8888", []string{"cmoa.jp"}, false},
		{"https://proxy.example.com", []string{" Cmoa.JP. ", ""}, false},
		{"http://gluetun:8888", nil, true},
		{"http://gluetun:8888", []string{" ", "."}, true},
		{"socks5://gluetun:1080", []string{"cmoa.jp"}, true},
		{"gluetun:8888", []string{"cmoa.jp"}, true},
		{"http://", []string{"cmoa.jp"}, true},
	} {
		err := SetProxy(tc.address, tc.domains)
		if (err != nil) != tc.wantErr {
			t.Errorf("SetProxy(%q, %q) error = %v, wantErr %v", tc.address, tc.domains, err, tc.wantErr)
		}
	}
}

func TestProxyFor(t *testing.T) {
	defer SetProxy("", nil)

	if ProxyFor("www.cmoa.jp") != nil || ProxyEnabled() {
		t.Fatal("no proxy should be used when it isn't configured")
	}

	if err := SetProxy("http://gluetun:8888", []string{"cmoa.jp", " BigComics.jp "}); err != nil {
		t.Fatal(err)
	}
	for host, want := range map[string]bool{
		"cmoa.jp":          true,
		"www.cmoa.jp":      true,
		"WWW.CMOA.JP.":     true,
		"bigcomics.jp":     true,
		"img.bigcomics.jp": true,
		"notcmoa.jp":       false,
		"cmoa.jp.evil.com": false,
		"mangadex.org":     false,
	} {
		if got := ProxyFor(host) != nil; got != want {
			t.Errorf("ProxyFor(%q) used = %v, want %v", host, got, want)
		}
	}
	if got := ProxyFor("cmoa.jp").String(); got != "http://gluetun:8888" {
		t.Errorf("ProxyFor returned %q", got)
	}

	if !isProxyDialAddress("gluetun:8888") || isProxyDialAddress("gluetun:80") {
		t.Error("isProxyDialAddress should match only the proxy host:port")
	}
	if err := SetProxy("http://gluetun", []string{"cmoa.jp"}); err != nil {
		t.Fatal(err)
	}
	if !isProxyDialAddress("gluetun:80") {
		t.Error("the proxy's default port should be 80 for http")
	}
}

func TestSafeTransportUsesProxyOnlyForItsDomains(t *testing.T) {
	// The test servers listen on loopback addresses, so treat only 127.0.0.1 as public
	isPublicAddrBackup := isPublicAddr
	isPublicAddr = func(addr netip.Addr) bool { return addr.Unmap() == netip.MustParseAddr("127.0.0.1") }
	defer func() { isPublicAddr = isPublicAddrBackup }()
	defer SetProxy("", nil)

	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("direct"))
	}))
	defer site.Close()

	// The proxy is on a private address, like a container in the same Docker network
	listener, err := net.Listen("tcp", "127.0.0.2:0")
	if err != nil {
		t.Skipf("can't listen on 127.0.0.2: %s", err)
	}
	var proxied []string
	proxyServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxied = append(proxied, r.URL.String())
		w.Write([]byte("proxied"))
	}))
	proxyServer.Listener = listener
	proxyServer.Start()
	defer proxyServer.Close()

	client := SafeHTTPClient(0)
	get := func(url string) (string, error) {
		resp, err := client.Get(url)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		return string(body), err
	}

	// The site isn't in the proxy domains: direct
	if err := SetProxy(proxyServer.URL, []string{"cmoa.jp"}); err != nil {
		t.Fatal(err)
	}
	if body, err := get(site.URL); err != nil || body != "direct" {
		t.Fatalf("got %q, %v; want the direct response", body, err)
	}

	// The site is in the proxy domains: through the proxy, even though the proxy is private
	if err := SetProxy(proxyServer.URL, []string{"127.0.0.1"}); err != nil {
		t.Fatal(err)
	}
	if body, err := get(site.URL + "/page"); err != nil || body != "proxied" {
		t.Fatalf("got %q, %v; want the proxied response", body, err)
	}
	if len(proxied) != 1 || proxied[0] != site.URL+"/page" {
		t.Fatalf("proxy received %v", proxied)
	}

	// A private destination is still blocked when it would go through the proxy
	if err := SetProxy(proxyServer.URL, []string{"127.0.0.2"}); err != nil {
		t.Fatal(err)
	}
	_, err = get("http://127.0.0.2:1/secret")
	var userErr *errordefs.UserError
	if !errors.As(err, &userErr) {
		t.Fatalf("expected a UserError for a private destination behind the proxy, got: %v", err)
	}
	if len(proxied) != 1 {
		t.Fatalf("the private destination reached the proxy: %v", proxied)
	}
}

func TestSafeTransportRefusesTheProxyAsADestination(t *testing.T) {
	isPublicAddrBackup := isPublicAddr
	isPublicAddr = func(addr netip.Addr) bool { return addr.Unmap() == netip.MustParseAddr("127.0.0.1") }
	defer func() { isPublicAddr = isPublicAddrBackup }()
	defer SetProxy("", nil)

	listener, err := net.Listen("tcp", "127.0.0.2:0")
	if err != nil {
		t.Skipf("can't listen on 127.0.0.2: %s", err)
	}
	proxyServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("proxy itself"))
	}))
	proxyServer.Listener = listener
	proxyServer.Start()
	defer proxyServer.Close()

	if err := SetProxy(proxyServer.URL, []string{"cmoa.jp"}); err != nil {
		t.Fatal(err)
	}
	_, err = SafeHTTPClient(0).Get(proxyServer.URL)
	var userErr *errordefs.UserError
	if !errors.As(err, &userErr) {
		t.Fatalf("a manga URL pointing at the private proxy should be blocked, got: %v", err)
	}
}

func TestSetProxyErrorsDontShowThePassword(t *testing.T) {
	defer SetProxy("", nil)
	for _, address := range []string{"socks5://user:hunter2@gluetun:1080", "http://user:hunter2@:8888", "http://user:hunter2@gluetun:port"} {
		err := SetProxy(address, []string{"cmoa.jp"})
		if err == nil {
			t.Fatalf("SetProxy(%q) should fail", address)
		}
		if strings.Contains(err.Error(), "hunter2") {
			t.Errorf("error shows the password: %s", err)
		}
	}
}

func TestSafeTransportSendsTheProxyCredentials(t *testing.T) {
	isPublicAddrBackup := isPublicAddr
	isPublicAddr = func(addr netip.Addr) bool { return addr.Unmap() == netip.MustParseAddr("127.0.0.1") }
	defer func() { isPublicAddr = isPublicAddrBackup }()
	defer SetProxy("", nil)

	var auth string
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Proxy-Authorization")
		w.Write([]byte("proxied"))
	}))
	defer proxyServer.Close()

	proxyURL := strings.Replace(proxyServer.URL, "http://", "http://mantium:s3cret@", 1)
	if err := SetProxy(proxyURL, []string{"127.0.0.1"}); err != nil {
		t.Fatal(err)
	}
	resp, err := SafeHTTPClient(0).Get("http://127.0.0.1:1/page")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	// base64("mantium:s3cret")
	if auth != "Basic bWFudGl1bTpzM2NyZXQ=" {
		t.Fatalf("Proxy-Authorization = %q", auth)
	}
}
