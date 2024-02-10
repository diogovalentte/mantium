package comick

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
)

// Client is a client for the Mangadex API
type Client struct {
	client *http.Client
	header http.Header
}

var userAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:30.0) Gecko/20100101 Firefox/30.0"

// NewComickClient creates a new Mangadex API client
func NewComickClient() *Client {
	client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MaxVersion: tls.VersionTLS12,
			},
		},
	}

	header := http.Header{}
	header.Set("Content-Type", "application/json")
	header.Set("User-Agent", userAgent)

	dex := &Client{
		client: &client,
		header: header,
	}

	return dex
}

// Request is a helper function to make a request to the Mangadex API
func (c *Client) Request(method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header = c.header

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	} else if resp.StatusCode != 200 {
		return nil, fmt.Errorf("non-200 status code -> (%d)", resp.StatusCode)
	}

	return resp, nil
}
