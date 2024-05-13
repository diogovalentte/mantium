package mangadex

import (
	"context"
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
func (c *Client) Request(ctx context.Context, method, url string, reqBody io.Reader, retBody interface{}) (*http.Response, error) {
	errorContext := "Error while making '%s' request"

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, util.AddErrorContext(err, fmt.Sprintf(errorContext, method))
	}

	req.Header = c.header

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, util.AddErrorContext(err, fmt.Sprintf(errorContext, method))
	} else if resp.StatusCode != http.StatusOK {
		errorContext = util.AddErrorContext(fmt.Errorf("Non-200 status code -> (%d) %%s", resp.StatusCode), errorContext).Error()
		// Decode to an ErrorResponse struct.
		var er ErrorResponse

		defer resp.Body.Close()
		if err = json.NewDecoder(resp.Body).Decode(&er); err != nil {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			return nil, util.AddErrorContext(fmt.Errorf("Error while decoding API error response into ErrorResponse. Body: %s", string(body)), fmt.Sprintf(errorContext, method))
		}
		return nil, util.AddErrorContext(fmt.Errorf(er.GetErrors()), fmt.Sprintf(errorContext, method))
	}

	if retBody != nil {
		defer resp.Body.Close()
		if err = json.NewDecoder(resp.Body).Decode(retBody); err != nil {
			body, _ := io.ReadAll(resp.Body)
			return nil, util.AddErrorContext(fmt.Errorf("Error decoding request body response into '%s'. Body: %s", reflect.TypeOf(retBody).Name(), string(body)), fmt.Sprintf(errorContext, method))
		}
	}

	return resp, nil
}
