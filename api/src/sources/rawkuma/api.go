package rawkuma

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"github.com/diogovalentte/mantium/api/src/util"
)

// Client is a client for the Rawkuma API
type Client struct {
	client *http.Client
}

// newAPIClient creates a new Rawkuma API client
func newAPIClient() *Client {
	client := http.Client{}

	kuma := &Client{
		client: &client,
	}

	return kuma
}

// Request is a helper function to make a request to the Rawkuma API
func (c *Client) Request(method, url string, reqBody io.Reader, retBody any, contentType string) (*http.Response, error) {
	errorContext := fmt.Sprintf("error while making '%s' request", method)

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	header := http.Header{}
	header.Set("Content-Type", contentType)
	req.Header = header

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	} else if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, util.AddErrorContext(errorContext, fmt.Errorf("non-200 status code -> (%d). Body: %s", resp.StatusCode, string(body)))
	}

	if retBody != nil {
		body, _ := io.ReadAll(resp.Body)
		if err = json.NewDecoder(bytes.NewReader(body)).Decode(retBody); err != nil {
			return nil, util.AddErrorContext(errorContext, fmt.Errorf("error decoding request body response into '%s'. Body: %s", reflect.TypeOf(retBody).Name(), string(body)))
		}
	}

	return resp, nil
}
