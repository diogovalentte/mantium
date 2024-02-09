package mangadex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
func (c *Client) Request(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header = c.header

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	} else if resp.StatusCode != 200 {
		// Decode to an ErrorResponse struct.
		var er ErrorResponse

		if err = json.NewDecoder(resp.Body).Decode(&er); err != nil {
			return nil, err
		}
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)
		return nil, fmt.Errorf("non-200 status code -> (%d) %s", resp.StatusCode, er.GetErrors())
	}

	return resp, nil
}
