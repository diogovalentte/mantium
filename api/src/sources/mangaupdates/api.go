package mangaupdates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/diogovalentte/mantium/api/src/util"
)

// Client is a client for the MangaUpdates API
type Client struct {
	client *http.Client
	header http.Header
}

// NewMangaUpdatesClient creates a new MangaUpdates API client
func NewMangaUpdatesClient() *Client {
	client := http.Client{}

	header := http.Header{}
	header.Set("Content-Type", "application/json")

	c := &Client{
		client: &client,
		header: header,
	}

	return c
}

// Request is a helper function to make a request to the MangaUpdates API
func (c *Client) Request(method, url string, reqBody io.Reader, retBody any) (*http.Response, error) {
	errorContext := fmt.Sprintf("error while making '%s' request", method)

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	req.Header = c.header

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	} else if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, util.AddErrorContext(errorContext, fmt.Errorf("non-200 status code -> (%d). Body: %s", resp.StatusCode, strings.ReplaceAll(string(body), "\n", "")))
	}

	if retBody != nil {
		body, _ := io.ReadAll(resp.Body)
		if err = json.NewDecoder(bytes.NewReader(body)).Decode(retBody); err != nil {
			return nil, util.AddErrorContext(errorContext, fmt.Errorf("error decoding request body response into '%s'. Body: %s", reflect.TypeOf(retBody).Name(), string(body)))
		}
	}

	return resp, nil
}
