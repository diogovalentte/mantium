package mangahub

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"github.com/google/uuid"

	"github.com/diogovalentte/mantium/api/src/util"
)

// Client is a client for the MangaHub API
type Client struct {
	client *http.Client
	header http.Header
}

// NewMangaHubClient creates a new MangaHub API client
func NewMangaHubClient() *Client {
	client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MaxVersion: tls.VersionTLS12,
			},
		},
	}

	header := http.Header{}
	header.Set("Content-Type", "application/json")
	header.Set("Accept", "application/json")
	header.Set("Origin", baseSiteURL)
	header.Set("User-Agent", userAgent)

	hub := &Client{
		client: &client,
		header: header,
	}

	return hub
}

// Request is a helper function to make a request to the MangaHub API
func (c *Client) Request(method, url string, reqBody io.Reader, retBody any) (*http.Response, error) {
	errorContext := fmt.Sprintf("error while making '%s' request", method)

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	req.Header = c.header
	req.Header.Set("x-mhub-access", uuid.New().String())

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	} else if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, util.AddErrorContext(errorContext, fmt.Errorf("non-200 status code -> (%d). Body: %s", resp.StatusCode, string(body)))
	}

	if retBody != nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if err = json.NewDecoder(bytes.NewReader(body)).Decode(retBody); err != nil {
			return nil, util.AddErrorContext(errorContext, fmt.Errorf("error decoding request body response into '%s'. Body: %s", reflect.TypeOf(retBody).Name(), string(body)))
		}
	}

	return resp, nil
}
