package comick

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"github.com/diogovalentte/mantium/api/src/util"
)

// Client is a client for the Comick API
type Client struct {
	client *http.Client
	header http.Header
}

var userAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:30.0) Gecko/20100101 Firefox/30.0"

// NewComickClient creates a new Comick API client
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

// Request is a helper function to make a request to the Comick API
func (c *Client) Request(method, url string, reqBody io.Reader, retBody interface{}) (*http.Response, error) {
	errorContext := "error while making '%s' request"

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, method), err)
	}

	req.Header = c.header

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, method), err)
	} else if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, method), fmt.Errorf("non-200 status code -> (%d). Body: %s", resp.StatusCode, string(body)))
	}

	if retBody != nil {
		body, _ := io.ReadAll(resp.Body)
		if err = json.NewDecoder(bytes.NewReader(body)).Decode(retBody); err != nil {
			return nil, util.AddErrorContext(fmt.Sprintf(errorContext, method), fmt.Errorf("error decoding request body response into '%s'. Body: %s", reflect.TypeOf(retBody).Name(), string(body)))
		}
	}

	return resp, nil
}
