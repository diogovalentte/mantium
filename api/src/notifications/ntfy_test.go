package notifications

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/AnthonyHewins/gotfy"

	"github.com/diogovalentte/mantium/api/src/config"
)

func setup() error {
	err := config.SetConfigs("../../../.env.test")
	if err != nil {
		return err
	}

	return nil
}

func TestMain(m *testing.M) {
	err := setup()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestSendNtfyMessage(t *testing.T) {
	publisher, err := GetNtfyPublisher()
	if err != nil {
		t.Fatalf("error getting ntfy publisher: %v", err)
	}

	link, err := url.Parse("https://www.google.com")
	if err != nil {
		t.Fatalf("error parsing link: %v", err)
	}
	msg := &gotfy.Message{
		Topic:   publisher.Topic,
		Title:   "Mantium test message",
		Message: "Test message",
		Actions: []gotfy.ActionButton{
			&gotfy.ViewAction{
				Label: "Open Link",
				Link:  link,
				Clear: false,
			},
		},
		ClickURL: link,
	}
	ctx := context.Background()
	err = publisher.SendMessage(ctx, msg)
	if err != nil {
		t.Fatalf("error sending message: %v", err)
	}
}
