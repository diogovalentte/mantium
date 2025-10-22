package mangaplus

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"google.golang.org/protobuf/proto"

	"github.com/mendoncart/mantium/api/src/util"
)

// Client is a client for the Manga Plus API
type Client struct {
	client *http.Client
	header http.Header
}

// NewMangaPlusClient creates a new Manga Plus API client
func NewMangaPlusClient() *Client {
	client := http.Client{}
	header := http.Header{
		"Accept":     []string{"*/*"},
		"User-Agent": []string{"okhttp/4.9.0"},
	}
	dex := &Client{
		client: &client,
		header: header,
	}

	return dex
}

// Request is a helper function to make a request to the Manga Plus API
func (c *Client) Request(url string) (*http.Response, *Response, error) {
	errorContext := "error while making request"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, util.AddErrorContext(errorContext, err)
	}

	req.Header = c.header

	reqResp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, util.AddErrorContext(errorContext, err)
	} else if reqResp.StatusCode != http.StatusOK {
		defer reqResp.Body.Close()
		body, _ := io.ReadAll(reqResp.Body)
		return nil, nil, util.AddErrorContext(errorContext, fmt.Errorf("non-200 status code -> (%d). Body: %s", reqResp.StatusCode, string(body)))
	}

	defer reqResp.Body.Close()
	body, err := io.ReadAll(reqResp.Body)
	if err != nil {
		return nil, nil, util.AddErrorContext(errorContext, err)
	}
	var response Response
	if err = proto.Unmarshal(body, &response); err != nil {
		return nil, nil, util.AddErrorContext(errorContext, fmt.Errorf("error decoding request body response. Body: %s", string(body)))
	}

	if response.Success == nil {
		code := strings.ReplaceAll(response.GetError().GetDefault().GetCode(), " ", "")
		message := response.GetError().GetDefault().GetMessage()

		return nil, nil, util.AddErrorContext(errorContext, fmt.Errorf("error response from Manga Plus API: %s - %s", code, message))
	}

	return reqResp, &response, nil
}
