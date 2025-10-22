package mangadex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"github.com/diogovalentte/mantium/api/src/util"
)

// Client is a client for the Mangadex API
type Client struct {
	client *http.Client
	header http.Header
}

// NewMangadexClient creates a new Mangadex API client
func NewMangadexClient() *Client {
	client := http.Client{}

	header := http.Header{}
	header.Set("Content-Type", "application/json")

	dex := &Client{
		client: &client,
		header: header,
	}

	return dex
}

// Request is a helper function to make a request to the Mangadex API
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
		errorContext = util.AddErrorContext(errorContext, fmt.Errorf("non-200 status code -> (%d)", resp.StatusCode)).Error()

		// Decode to an ErrorResponse struct
		var er ErrorResponse
		defer resp.Body.Close()
		if err = json.NewDecoder(resp.Body).Decode(&er); err != nil {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			return nil, util.AddErrorContext(errorContext, fmt.Errorf("error while decoding API error response into ErrorResponse. Body: %s", string(body)))
		}
		return nil, util.AddErrorContext(errorContext, fmt.Errorf(er.GetErrors()))
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
