// Package telegram provides functionality to send messages via Telegram Bot API
package telegram

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/mendoncart/mantium/api/src/config"
	"github.com/mendoncart/mantium/api/src/util"
)

type Bot struct {
	APIToken string
	ChatIDs  []string
}

// GetTelegramBot returns a new Telegram Bot instance
func GetTelegramBot() (*Bot, error) {
	configs := config.GlobalConfigs.Telegram
	
	if configs.APIToken == "" {
		return nil, util.AddErrorContext("telegram bot token not configured", nil)
	}

	if len(configs.ChatIDs) == 0 {
		return nil, util.AddErrorContext("no telegram chat IDs configured", nil)
	}

	return &Bot{
		APIToken: configs.APIToken,
		ChatIDs:  configs.ChatIDs,
	}, nil
}

// SendMessage sends a message to all configured Telegram chats
func (b *Bot) SendMessage(ctx context.Context, messageText string, buttonLabel string, buttonURL string) error {
	baseURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", b.APIToken)
	
	// Create inline keyboard with URL button if provided
	var inlineKeyboard string
	if buttonLabel != "" && buttonURL != "" {
		inlineKeyboard = fmt.Sprintf(`,"reply_markup":{"inline_keyboard":[[{"text":"%s","url":"%s"}]]}`, buttonLabel, buttonURL)
	}

	for _, chatID := range b.ChatIDs {
		// Construct the request body
		requestBody := fmt.Sprintf(`{"chat_id":"%s","text":"%s","parse_mode":"HTML"%s}`, 
			chatID, messageText, inlineKeyboard)

		// Create request
		req, err := http.NewRequestWithContext(ctx, "POST", baseURL, 
			strings.NewReader(requestBody))
		if err != nil {
			return util.AddErrorContext("could not create telegram request", err)
		}
		req.Header.Set("Content-Type", "application/json")

		// Send request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return util.AddErrorContext("could not send telegram message", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return util.AddErrorContext(
				fmt.Sprintf("telegram API returned non-200 status code: %d", resp.StatusCode), 
				nil)
		}
	}

	return nil
}
