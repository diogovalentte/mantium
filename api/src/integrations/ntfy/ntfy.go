// Package ntfy provides a wrapper around gotfy.Publisher to send messages to the Ntfy server
package ntfy

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/AnthonyHewins/gotfy"

	"github.com/diogovalentte/mantium/api/src/config"
	"github.com/diogovalentte/mantium/api/src/util"
)

// GetNtfyPublisher returns a new NtfyPublisher
func GetNtfyPublisher() (*Publisher, error) {
	contextError := "could not get Ntfy publisher"

	configs := config.GlobalConfigs.Ntfy

	server, err := url.Parse(configs.Address)
	if err != nil {
		return nil, util.AddErrorContext(contextError, err)
	}

	customClient := &http.Client{
		Transport: &customNtfyTransport{
			ntfyToken: configs.Token,
		},
	}
	publisher, err := gotfy.NewPublisher(server, customClient)
	if err != nil {
		return nil, util.AddErrorContext(contextError, err)
	}

	return &Publisher{
		Publisher: publisher,
		Topic:     configs.Topic,
		Token:     configs.Token,
	}, nil
}

type customNtfyTransport struct {
	ntfyToken string
}

func (t *customNtfyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.ntfyToken))

	return http.DefaultTransport.RoundTrip(req)
}

// Publisher is a wrapper around gotfy.Publisher
type Publisher struct {
	Publisher *gotfy.Publisher
	Topic     string
	Token     string
}

// SendMessage sends a message to the Ntfy server
func (t *Publisher) SendMessage(ctx context.Context, message *gotfy.Message) error {
	_, err := t.Publisher.SendMessage(ctx, message)
	if err != nil {
		return util.AddErrorContext("could not send message to Ntfy", err)
	}

	return nil
}
