package util

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/diogovalentte/mantium/api/src/errordefs"
)

func encodePNG(t *testing.T, width, height int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewGray(image.Rect(0, 0, width, height))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestValidateImage(t *testing.T) {
	tests := []struct {
		name        string
		img         []byte
		userMessage string
	}{
		{name: "valid", img: encodePNG(t, 10, 10)},
		{name: "not an image", img: []byte("hello"), userMessage: "The cover image is not a valid PNG, JPEG or WebP image"},
		{name: "too many pixels", img: encodePNG(t, 8000, 6000), userMessage: "The cover image is too large (8000x6000 pixels), the maximum is 40 megapixels"},
		{name: "too many bytes", img: make([]byte, MaxImageBytes+1), userMessage: "The cover image is larger than 10 MB"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateImage(test.img)
			if test.userMessage == "" {
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				return
			}
			var userErr *errordefs.UserError
			if !errors.As(err, &userErr) || userErr.Message != test.userMessage {
				t.Fatalf("got %v, want user message %q", err, test.userMessage)
			}
		})
	}
}

func TestGetImageFromURLTooLarge(t *testing.T) {
	SetAllowPrivateURLs(true)
	defer SetAllowPrivateURLs(false)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(bytes.Repeat([]byte("a"), MaxImageBytes+10))
	}))
	defer server.Close()

	_, _, err := GetImageFromURL(server.URL, 1, time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "The cover image is larger than 10 MB") {
		t.Fatalf("expected too large error, got %v", err)
	}
}
