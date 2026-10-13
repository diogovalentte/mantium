package util

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/diogovalentte/mantium/api/src/errordefs"
)

func TestIsPublicAddr(t *testing.T) {
	for addr, want := range map[string]bool{
		"8.8.8.8":                true,
		"2606:4700:4700::1111":   true,
		"127.0.0.1":              false,
		"10.1.2.3":               false,
		"172.16.0.1":             false,
		"192.168.0.118":          false,
		"169.254.169.254":        false,
		"100.70.203.122":         false,
		"0.0.0.0":                false,
		"0.1.2.3":                false,
		"224.0.0.1":              false,
		"::1":                    false,
		"fd00::1":                false,
		"fe80::1":                false,
		"::ffff:127.0.0.1":       false,
		"::ffff:192.168.0.1":     false,
		"::ffff:169.254.169.254": false,
	} {
		if got := isPublicAddr(netip.MustParseAddr(addr)); got != want {
			t.Errorf("isPublicAddr(%s) = %v, want %v", addr, got, want)
		}
	}
}

func TestSafeHTTPClientBlocksRedirectToPrivateAddress(t *testing.T) {
	// The test servers listen on loopback addresses, so treat only 127.0.0.1 as public
	isPublicAddrBackup := isPublicAddr
	isPublicAddr = func(addr netip.Addr) bool { return addr.Unmap() == netip.MustParseAddr("127.0.0.1") }
	defer func() { isPublicAddr = isPublicAddrBackup }()

	listener, err := net.Listen("tcp", "127.0.0.2:0")
	if err != nil {
		t.Skipf("can't listen on 127.0.0.2: %s", err)
	}
	private := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("secret"))
	}))
	private.Listener = listener
	private.Start()
	defer private.Close()

	public := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, private.URL, http.StatusFound)
			return
		}
		w.Write([]byte("ok"))
	}))
	defer public.Close()

	client := SafeHTTPClient(0)

	resp, err := client.Get(public.URL)
	if err != nil {
		t.Fatalf("request to the public address failed: %s", err)
	}
	resp.Body.Close()

	for _, url := range []string{private.URL, public.URL + "/redirect"} {
		_, err = client.Get(url)
		var userErr *errordefs.UserError
		if !errors.As(err, &userErr) {
			t.Fatalf("%s: expected a UserError, got: %v", url, err)
		}
		want := "Blocked access to a private address (127.0.0.2). Set ALLOW_PRIVATE_URLS=true in the API to allow it"
		if userErr.Message != want {
			t.Fatalf("%s: got message %q, want %q", url, userErr.Message, want)
		}
	}
}
