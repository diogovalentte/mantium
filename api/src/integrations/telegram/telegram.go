// Package telegram provides functionality to send messages via Telegram Bot API
package telegram

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"encoding/json"
	"log"
	"strconv"
	"bytes"
	"io"
	
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/mendoncart/mantium/api/src/config"
	"github.com/mendoncart/mantium/api/src/util"
)


// Global bot instance
var globalBot *Bot
var botInitialized bool

// InitializeBotIfEnabled initializes the Telegram bot if polling is enabled
// This should be called when the application starts
func InitializeBotIfEnabled() error {
	if !config.GlobalConfigs.Telegram.Valid {
		log.Println("Telegram bot not configured, skipping initialization")
		return nil
	}

	if !config.GlobalConfigs.Telegram.EnablePolling {
		log.Println("Telegram bot polling is disabled")
		return nil
	}

	if botInitialized {
		log.Println("Telegram bot already initialized")
		return nil
	}

	log.Println("Initializing Telegram bot with polling enabled...")
	
	bot, err := GetTelegramBot()
	if err != nil {
		return util.AddErrorContext("failed to initialize telegram bot", err)
	}

	globalBot = bot
	botInitialized = true
	
	log.Println("✅ Telegram bot initialized successfully and polling started")
	return nil
}

// CallbackData represents the structure of callback data for Telegram inline buttons
type CallbackData struct {
	Action    string `json:"a"`           // Action to perform
	MangaID   string `json:"m,omitempty"` // Manga ID
	ChapterID string `json:"c,omitempty"` // Chapter ID
	Status    int    `json:"s,omitempty"` // Status value
	Offset    int    `json:"o,omitempty"` // Offset for pagination
	Command   string `json:"cmd,omitempty"` // Command type (unread/reading)
}

// Bot represents a Telegram Bot
type Bot struct {
    APIToken string
    ChatIDs  []int64 
    api      *tgbotapi.BotAPI
    polling  bool
}

// MangaInfo holds basic manga information needed for operations
type MangaInfo struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Source       string `json:"source"`
	MultiMangaID int    `json:"multiMangaID"`
}

// MangaListItem represents a manga in the list
type MangaListItem struct {
	ID                  int    `json:"id"`
	Name                string `json:"name"`
	Source              string `json:"source"`
	LastReadChapter     *ChapterInfo `json:"lastReadChapter"`
	LastReleasedChapter *ChapterInfo `json:"lastReleasedChapter"`
	Status              int    `json:"status"`
	MultiMangaID        int    `json:"multiMangaID"`
}

// ChapterInfo represents chapter information
type ChapterInfo struct {
	Chapter string `json:"chapter"`
	URL     string `json:"url"`
}

// MangaSearchResult represents a search result
type MangaSearchResult struct {
	Name        string `json:"name"`
	InternalID  string `json:"internalID"`
	URL         string `json:"url"`
	CoverURL    string `json:"coverURL"`
	Description string `json:"description"`
	Source      string `json:"source"`
	LastChapter string `json:"lastChapter"`
}

// Callback actions for inline buttons
const (
	ActionHelp           = "help"
	ActionListUnread     = "list_unread"
	ActionListReading    = "list_reading"
	ActionSearch         = "search"
	ActionSetRead        = "set_read"
	ActionReadChapter    = "read_chapter"
	ActionChangeStatus   = "change_status"
	ActionSetStatus      = "set_status"
	ActionCancel         = "cancel"
	ActionListMore       = "list_more"
)

// UserSession stores temporary session data for users
type UserSession struct {
	WaitingForSearch bool
	LastCommand      string
	LastOffset       int
}

// Global session storage (in production, use a proper session manager)
var userSessions = make(map[int64]*UserSession)

// GetTelegramBot returns a new Telegram Bot instance or the global instance if already initialized
func GetTelegramBot() (*Bot, error) {
	// If bot is already initialized and polling, return the global instance
	if botInitialized && globalBot != nil {
		return globalBot, nil
	}

	configs := config.GlobalConfigs.Telegram
	
	if configs.APIToken == "" {
		return nil, util.AddErrorContext("telegram bot token not configured", nil)
	}

	if len(configs.ChatIDs) == 0 {
		return nil, util.AddErrorContext("no telegram chat IDs configured", nil)
	}

	// Convert string chat IDs to int64
	chatIDs := make([]int64, len(configs.ChatIDs))
	for i, idStr := range configs.ChatIDs {
		var id int64
		_, err := fmt.Sscanf(idStr, "%d", &id)
		if err != nil {
			return nil, util.AddErrorContext(fmt.Sprintf("invalid chat ID: %s", idStr), err)
		}
		chatIDs[i] = id
	}

	api, err := tgbotapi.NewBotAPI(configs.APIToken)
	if err != nil {
		return nil, util.AddErrorContext("failed to create telegram bot", err)
	}

	bot := &Bot{
		APIToken: configs.APIToken,
		ChatIDs:  chatIDs,
		api:      api,
		polling:  configs.EnablePolling,
	}

	// Start polling if enabled
	if configs.EnablePolling {
		go bot.startPolling()
		botInitialized = true
		globalBot = bot
	}

	return bot, nil
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
		requestBody := fmt.Sprintf(`{"chat_id":%d,"text":"%s","parse_mode":"HTML"%s}`, 
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

// SendChapterUpdate sends a notification about a new chapter using the standardized manga card format
// This ensures consistent button layout across all notifications
func (b *Bot) SendChapterUpdate(ctx context.Context, messageText string, mangaID, chapterID string, coverURL string, chapterURL string) error {
	// If polling is not enabled, fall back to simple message
	if !b.polling {
		return b.SendMessage(ctx, messageText, "", "")
	}

	// Fetch the complete manga data to use the standardized card format
	manga, err := b.getMangaByID(mangaID)
	if err != nil {
		log.Printf("Error fetching manga %s for chapter update: %v", mangaID, err)
		// Fallback to simple message if we can't fetch manga data
		return b.SendMessage(ctx, messageText, "", "")
	}

	// Send the manga card to all configured chat IDs with custom caption
	for _, chatID := range b.ChatIDs {
		b.sendMangaCard(chatID, *manga, messageText)
	}

	return nil
}

// startPolling starts the bot polling loop to receive updates
// This runs in a separate goroutine and handles incoming callback queries and messages
func (b *Bot) startPolling() {
	log.Println("Starting Telegram bot polling...")
	
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		// Handle callback queries (button clicks)
		if update.CallbackQuery != nil {
			b.handleCallback(update.CallbackQuery)
		}
		
		// Handle regular messages and commands
		if update.Message != nil {
			// Check if user is in a session (e.g., waiting for search input)
			session, hasSession := userSessions[update.Message.Chat.ID]
			
			if hasSession && session.WaitingForSearch && !update.Message.IsCommand() {
				// User is replying with search query
				query := update.Message.Text
				log.Printf("Received search query from user: %s", query)
				
				// Clear session
				delete(userSessions, update.Message.Chat.ID)
				
				// Perform search
				b.performSearch(update.Message.Chat.ID, query)
				continue
			}
			
			// Check if it's a command
			if update.Message.IsCommand() {
				// Clear any active session when a new command is issued
				delete(userSessions, update.Message.Chat.ID)
				b.handleCommand(update.Message)
			} else {
				// Regular message (not a command, not in session)
				log.Printf("Received message from user %s: %s", 
					update.Message.From.UserName, update.Message.Text)
			}
		}
	}
}

// handleCallback processes callback queries from inline buttons
func (b *Bot) handleCallback(callback *tgbotapi.CallbackQuery) {
	log.Printf("Received callback from user %s: %s", callback.From.UserName, callback.Data)
	
	var data CallbackData
	if err := json.Unmarshal([]byte(callback.Data), &data); err != nil {
		log.Printf("Error unmarshaling callback data: %v", err)
		b.sendCallbackError(callback, "Error processing button data")
		return
	}

	switch data.Action {
	case ActionHelp:
		b.sendHelpMessage(callback.Message.Chat.ID, callback.Message.MessageID)
		b.answerCallback(callback, "")
	
	case ActionListUnread:
		b.sendMangaList(callback.Message.Chat.ID, callback.Message.MessageID, "unread", 0)
		b.answerCallback(callback, "Loading unread manga...")
	
	case ActionListReading:
		b.sendMangaList(callback.Message.Chat.ID, callback.Message.MessageID, "reading", 0)
		b.answerCallback(callback, "Loading reading manga...")
	
	case ActionSearch:
	    //b.processSearch(callback)
		// Deleta menu antigo
		b.api.Send(tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID))
		// Chama a mesma função de envio de mensagem nova
		b.processSearch(callback.Message.Chat.ID)
		b.answerCallback(callback, "Enter your search term...")
	
	// case ActionSearch:
	// 	 // Delete callback message
	// 	b.api.Send(tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID))
	// 	// Call handleSearchCommand normally
	// 	b.handleSearchCommand(callback.Message)
	// 	b.answerCallback(callback, "Enter your search term...")

	case ActionSetRead:
		log.Printf("Processing set_read action for manga %s", data.MangaID)
		
		err := b.updateLastReadChapter(data.MangaID, data.ChapterID)
		if err != nil {
			log.Printf("Error updating last read chapter: %v", err)
			b.answerCallback(callback, "❌ Failed to update")
			return
		}
		
		b.answerCallback(callback, "✓ Chapter marked as read!")
		
		// Edit the message to show updated status
		newCaption := callback.Message.Caption + "\n\n✅ <i>Marked as read!</i>"
		edit := tgbotapi.NewEditMessageCaption(callback.Message.Chat.ID, callback.Message.MessageID, newCaption)
		edit.ParseMode = "HTML"
		b.api.Send(edit)
	
	case ActionChangeStatus:
		b.sendChangeStatusMenu(callback, data.MangaID, data.Status)
		b.answerCallback(callback, "")
	
	case ActionSetStatus:
		b.updateMangaStatus(callback, data.MangaID, data.Status)
		b.answerCallback(callback, "✓ Status updated!")
	
	case ActionCancel:
		// Cancel current operation
		delete(userSessions, callback.Message.Chat.ID)
		
		// Delete the message
		deleteMsg := tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID)
		b.api.Send(deleteMsg)
		
		b.answerCallback(callback, "Cancelled")
	
	case ActionListMore:
		b.sendMangaList(callback.Message.Chat.ID, 0, data.Command, data.Offset)
		b.answerCallback(callback, "Loading more...")
		
		// Delete the "list more" message
		deleteMsg := tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID)
		b.api.Send(deleteMsg)
	
	default:
		log.Printf("Unknown callback action: %s", data.Action)
		b.answerCallback(callback, "Unknown action")
	}
}

// answerCallback answers a callback query
func (b *Bot) answerCallback(callback *tgbotapi.CallbackQuery, text string) {
	msg := tgbotapi.NewCallback(callback.ID, text)
	b.api.Request(msg)
}

// sendChangeStatusMenu sends a menu to change manga status
func (b *Bot) sendChangeStatusMenu(callback *tgbotapi.CallbackQuery, mangaID string, currentStatus int) {
	statuses := []struct {
		ID   int
		Name string
	}{
		{1, "📖 Reading"},
		{2, "✅ Completed"},
		{3, "⏸️ On Hold"},
		{4, "❌ Dropped"},
		{5, "📋 Plan to Read"},
	}

	var buttons [][]tgbotapi.InlineKeyboardButton
	
	for _, status := range statuses {
		label := status.Name
		if status.ID == currentStatus {
			label += " ✓"
		}
		
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				label,
				createCallbackData(ActionSetStatus, mangaID, "", status.ID, 0, ""),
			),
		))
	}
	
	// Add cancel button
	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(
			"↩️ Back",
			createCallbackData(ActionCancel, "", "", 0, 0, ""),
		),
	))
	
	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	
	// Edit the message to show status selection
	edit := tgbotapi.NewEditMessageReplyMarkup(
		callback.Message.Chat.ID,
		callback.Message.MessageID,
		keyboard,
	)
	b.api.Send(edit)
}

// updateMangaStatus updates a manga's status via API
func (b *Bot) updateMangaStatus(callback *tgbotapi.CallbackQuery, mangaIDStr string, newStatus int) {
	// Get manga info first to determine if it's custom or multimanga
	manga, err := b.getMangaInfo(mangaIDStr)
	if err != nil {
		log.Printf("Error getting manga info: %v", err)
		return
	}

	var apiURL string
	if manga.Source == "custom_manga" {
		apiURL = fmt.Sprintf("http://localhost:%s/v1/manga/status?id=%s", 
			config.GlobalConfigs.API.Port, mangaIDStr)
	} else {
		apiURL = fmt.Sprintf("http://localhost:%s/v1/multimanga/status?id=%d",
			config.GlobalConfigs.API.Port, manga.MultiMangaID)
	}

	requestBody := map[string]interface{}{
		"status": newStatus,
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, err := http.NewRequest("PATCH", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("Error creating status update request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error updating status: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("API returned error status: %d", resp.StatusCode)
		return
	}

	log.Printf("Successfully updated status for manga %s to %d", mangaIDStr, newStatus)

	// Update the caption to show new status
	newCaption := callback.Message.Caption
	// Remove old status line and add new one
	lines := strings.Split(newCaption, "\n")
	if len(lines) > 1 {
		lines[1] = getStatusName(newStatus)
		newCaption = strings.Join(lines, "\n")
	}
	newCaption += fmt.Sprintf("\n\n✅ <i>Status changed to %s</i>", getStatusName(newStatus))

	// Restore original buttons
	// mangaID, _ := strconv.Atoi(mangaIDStr)
	
	// We need to get fresh manga data to rebuild buttons properly
	// For now, just restore basic buttons
	var buttons [][]tgbotapi.InlineKeyboardButton
	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(
			"✓ Set as read",
			createCallbackData(ActionSetRead, mangaIDStr, "", 0, 0, ""),
		),
	))
	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(
			"⚙️ Change Status",
			createCallbackData(ActionChangeStatus, mangaIDStr, "", newStatus, 0, ""),
		),
	))
	
	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)

	edit := tgbotapi.NewEditMessageCaption(callback.Message.Chat.ID, callback.Message.MessageID, newCaption)
	edit.ParseMode = "HTML"
	edit.ReplyMarkup = &keyboard
	b.api.Send(edit)
}

// getMangaByID fetches a single manga by ID from the API
func (b *Bot) getMangaByID(mangaID string) (*MangaListItem, error) {
	// Get all manga from API (no direct endpoint for single manga with full details)
	apiURL := fmt.Sprintf("http://localhost:%s/v1/mangas", config.GlobalConfigs.API.Port)
	
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manga list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Mangas []MangaListItem `json:"mangas"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode manga list: %w", err)
	}

	// Find the manga by ID
	mangaIDInt, err := strconv.Atoi(mangaID)
	if err != nil {
		return nil, fmt.Errorf("invalid manga ID: %s", mangaID)
	}

	for _, manga := range result.Mangas {
		if manga.ID == mangaIDInt {
			return &manga, nil
		}
	}

	return nil, fmt.Errorf("manga with ID %s not found", mangaID)
}

// sendCallbackError sends an error message as callback response
func (b *Bot) sendCallbackError(callback *tgbotapi.CallbackQuery, errorMsg string) {
	msg := tgbotapi.NewCallback(callback.ID, errorMsg)
	if _, err := b.api.Request(msg); err != nil {
		log.Printf("Error sending callback error: %v", err)
	}
}

// updateLastReadChapter calls the Mantium API to update a manga's last read chapter
// It determines if the manga is a custom manga or multimanga and calls the appropriate endpoint
func (b *Bot) updateLastReadChapter(mangaIDStr, chapterID string) error {
	// First, get the manga to determine if it's custom or multimanga
	manga, err := b.getMangaInfo(mangaIDStr)
	if err != nil {
		return util.AddErrorContext("failed to get manga info", err)
	}

	// Determine the API endpoint based on manga type
	var apiURL string
	var requestBody map[string]interface{}
	
	if manga.Source == "custom_manga" {
		// Custom manga endpoint: PATCH /v1/custom_manga/last_read_chapter?id={id}
		apiURL = fmt.Sprintf("http://localhost:%s/v1/custom_manga/last_read_chapter?id=%s", 
			config.GlobalConfigs.API.Port, mangaIDStr)
		
		// For custom manga, if no chapter in body, it sets to last released chapter
		requestBody = map[string]interface{}{}
		
	} else {
		// Multimanga endpoint: PATCH /v1/multimanga/last_read_chapter?id={multimanga_id}&manga_id={manga_id}
		apiURL = fmt.Sprintf("http://localhost:%s/v1/multimanga/last_read_chapter?id=%s&manga_id=%s",
			config.GlobalConfigs.API.Port, strconv.Itoa(manga.MultiMangaID), mangaIDStr)
		
		// For multimanga, empty chapter object sets to last released chapter
		requestBody = map[string]interface{}{
			"chapter": map[string]string{},
		}
	}

	// Marshal the request body
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return util.AddErrorContext("failed to marshal request body", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("PATCH", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return util.AddErrorContext("failed to create HTTP request", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return util.AddErrorContext("failed to send HTTP request", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("Successfully updated last read chapter for manga %s", mangaIDStr)
	return nil
}

// getMangaInfo retrieves manga information from the API
func (b *Bot) getMangaInfo(mangaIDStr string) (*MangaInfo, error) {
	apiURL := fmt.Sprintf("http://localhost:%s/v1/manga?id=%s", 
		config.GlobalConfigs.API.Port, mangaIDStr)

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, util.AddErrorContext("failed to get manga info", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Manga MangaInfo `json:"manga"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, util.AddErrorContext("failed to decode manga info", err)
	}

	return &result.Manga, nil
}

// handleCommand processes bot commands
// This function handles commands like /start, /help, /setlastread, etc.
func (b *Bot) handleCommand(message *tgbotapi.Message) {
	log.Printf("Processing command: %s from user %s", message.Command(), message.From.UserName)
	
	switch message.Command() {

	case "start":
		reply := "🎌 <b>Welcome to Mantium Bot!</b>\n\n" +
			"I'll notify you about new manga chapters and help you manage your reading list.\n\n" +
			"Click below to see what I can do!"
		
		// Create inline keyboard with "Available Commands" button
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📚 Available Commands", createCallbackData(ActionHelp, "", "", 0, 0, "")),
			),
		)
		
		msg := tgbotapi.NewMessage(message.Chat.ID, reply)
		msg.ParseMode = "HTML"
		msg.ReplyMarkup = keyboard
		b.api.Send(msg)
	
	case "help":
		b.sendHelpMessage(message.Chat.ID, 0)
	
	case "setlastread":
		b.handleSetLastReadCommand(message)

	case "listunread":
		b.handleListUnreadCommand(message, 0)

	case "listreading":
		b.handleListReadingCommand(message, 0)

	case "search":
		b.handleSearchCommand(message)
	
	default:
		reply := "❓ Unknown command. Type /help for available commands."
		msg := tgbotapi.NewMessage(message.Chat.ID, reply)
		b.api.Send(msg)
	}
}

// createCallbackData creates a compact JSON callback data string
func createCallbackData(action, mangaID, chapterID string, status, offset int, command string) string {
	data := CallbackData{
		Action:    action,
		MangaID:   mangaID,
		ChapterID: chapterID,
		Status:    status,
		Offset:    offset,
		Command:   command,
	}
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

// sendHelpMessage sends the help message with inline buttons
func (b *Bot) sendHelpMessage(chatID int64, messageID int) {
	reply := "📚 <b>Available Commands</b>\n\n" +
		"Use the buttons below to interact with your manga library:\n\n" +
		"• <b>List Unread</b> - Show manga with new chapters\n" +
		"• <b>List Reading</b> - Show all manga you're reading\n" +
		"• <b>Search Library</b> - Find a specific manga\n\n" +
		"<i>You can also use buttons in chapter notifications!</i>"
	
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📖 List Unread", createCallbackData(ActionListUnread, "", "", 0, 0, "")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📚 List Reading", createCallbackData(ActionListReading, "", "", 0, 0, "")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔍 Search Library", createCallbackData(ActionSearch, "", "", 0, 0, "")),
		),
	)
	
	if messageID > 0 {
		// Edit existing message
		edit := tgbotapi.NewEditMessageText(chatID, messageID, reply)
		edit.ParseMode = "HTML"
		edit.ReplyMarkup = &keyboard
		b.api.Send(edit)
	} else {
		// Send new message
		msg := tgbotapi.NewMessage(chatID, reply)
		msg.ParseMode = "HTML"
		msg.ReplyMarkup = keyboard
		b.api.Send(msg)
	}
}

// getStatusName returns the human-readable status name
func getStatusName(status int) string {
	statusNames := map[int]string{
		1: "📖 Reading",
		2: "✅ Completed",
		3: "⏸️ On Hold",
		4: "❌ Dropped",
		5: "📋 Plan to Read",
	}
	name := statusNames[status]
	if name == "" {
		return "Unknown"
	}
	return name
}

// handleSetLastReadCommand handles the /setlastread command
// Usage: /setlastread <manga_id>
func (b *Bot) handleSetLastReadCommand(message *tgbotapi.Message) {
	// Extract manga ID from command arguments
	args := strings.Fields(message.Text)
	if len(args) < 2 {
		reply := "❌ <b>Invalid usage!</b>\n\n" +
			"Usage: /setlastread &lt;manga_id&gt;\n\n" +
			"Example: /setlastread 42"
		msg := tgbotapi.NewMessage(message.Chat.ID, reply)
		msg.ParseMode = "HTML"
		b.api.Send(msg)
		return
	}

	mangaID := args[1]
	
	log.Printf("Processing /setlastread command for manga ID: %s", mangaID)

	// Get manga info first
	manga, err := b.getMangaInfo(mangaID)
	if err != nil {
		log.Printf("Error getting manga info: %v", err)
		reply := fmt.Sprintf("❌ <b>Error:</b> Could not find manga with ID %s", mangaID)
		msg := tgbotapi.NewMessage(message.Chat.ID, reply)
		msg.ParseMode = "HTML"
		b.api.Send(msg)
		return
	}

	// Update last read chapter
	err = b.updateLastReadChapter(mangaID, "")
	if err != nil {
		log.Printf("Error updating last read chapter: %v", err)
		reply := fmt.Sprintf("❌ <b>Error:</b> Failed to update last read chapter for manga '%s'", manga.Name)
		msg := tgbotapi.NewMessage(message.Chat.ID, reply)
		msg.ParseMode = "HTML"
		b.api.Send(msg)
		return
	}

	// Send success message
	reply := fmt.Sprintf("✅ <b>Success!</b>\n\nLast read chapter updated for:\n<b>%s</b>", manga.Name)
	msg := tgbotapi.NewMessage(message.Chat.ID, reply)
	msg.ParseMode = "HTML"
	b.api.Send(msg)
}

// handleListCommand handles the /list command
// Lists all manga with unread chapters (status reading or completed)
// func (b *Bot) handleListCommand(message *tgbotapi.Message) {
// 	log.Printf("Processing /list command from user %s", message.From.UserName)

// 	// Get all manga from API
// 	apiURL := fmt.Sprintf("http://localhost:%s/v1/mangas", config.GlobalConfigs.API.Port)
	
// 	resp, err := http.Get(apiURL)
// 	if err != nil {
// 		log.Printf("Error getting manga list: %v", err)
// 		reply := "❌ <b>Error:</b> Failed to fetch manga list"
// 		msg := tgbotapi.NewMessage(message.Chat.ID, reply)
// 		msg.ParseMode = "HTML"
// 		b.api.Send(msg)
// 		return
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		body, _ := io.ReadAll(resp.Body)
// 		log.Printf("API returned error: %s", string(body))
// 		reply := "❌ <b>Error:</b> Failed to fetch manga list"
// 		msg := tgbotapi.NewMessage(message.Chat.ID, reply)
// 		msg.ParseMode = "HTML"
// 		b.api.Send(msg)
// 		return
// 	}

// 	var result struct {
// 		Mangas []MangaListItem `json:"mangas"`
// 	}
	
// 	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
// 		log.Printf("Error decoding manga list: %v", err)
// 		reply := "❌ <b>Error:</b> Failed to parse manga list"
// 		msg := tgbotapi.NewMessage(message.Chat.ID, reply)
// 		msg.ParseMode = "HTML"
// 		b.api.Send(msg)
// 		return
// 	}

// 	// Filter manga with unread chapters and status reading (1) or completed (2)
// 	var unreadMangas []MangaListItem
// 	for _, manga := range result.Mangas {
// 		if (manga.Status == 1 || manga.Status == 2) && hasUnreadChapters(manga) {
// 			unreadMangas = append(unreadMangas, manga)
// 		}
// 	}

// 	if len(unreadMangas) == 0 {
// 		reply := "✅ <b>All caught up!</b>\n\nYou have no manga with unread chapters."
// 		msg := tgbotapi.NewMessage(message.Chat.ID, reply)
// 		msg.ParseMode = "HTML"
// 		b.api.Send(msg)
// 		return
// 	}

// 	// Build response message
// 	reply := fmt.Sprintf("📖 <b>Manga with unread chapters</b> (%d):\n\n", len(unreadMangas))
	
// 	for i, manga := range unreadMangas {
// 		if i >= 10 { // Limit to 10 manga to avoid message too long
// 			reply += fmt.Sprintf("\n<i>... and %d more</i>", len(unreadMangas)-10)
// 			break
// 		}
		
// 		lastRead := "N/A"
// 		if manga.LastReadChapter != nil {
// 			lastRead = manga.LastReadChapter.Chapter
// 		}
		
// 		lastReleased := "N/A"
// 		if manga.LastReleasedChapter != nil {
// 			lastReleased = manga.LastReleasedChapter.Chapter
// 		}
		
// 		reply += fmt.Sprintf("📚 <b>%s</b>\n", manga.Name)
// 		reply += fmt.Sprintf("   ID: <code>%d</code> | Read: %s → New: %s\n", 
// 			manga.ID, lastRead, lastReleased)
// 		reply += fmt.Sprintf("   /status %d | /setlastread %d\n\n", manga.ID, manga.ID)
// 	}

// 	msg := tgbotapi.NewMessage(message.Chat.ID, reply)
// 	msg.ParseMode = "HTML"
// 	b.api.Send(msg)
// }

// hasUnreadChapters checks if a manga has unread chapters
func hasUnreadChapters(manga MangaListItem) bool {
	if manga.LastReleasedChapter == nil {
		return false
	}
	
	if manga.LastReadChapter == nil {
		return true
	}
	
	return manga.LastReadChapter.Chapter != manga.LastReleasedChapter.Chapter
}

// Novo método para tratar search
func (b *Bot) processSearch(chatID int64) {
    // Atualiza sessão
    userSessions[chatID] = &UserSession{
        WaitingForSearch: true,
        LastCommand:      "search",
    }

    msgText := "🔍 <b>Search Your Library</b>\n\nPlease send a message with the manga name you're looking for:"

    // Inline button cancel
    keyboard := tgbotapi.NewInlineKeyboardMarkup(
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", createCallbackData(ActionHelp, "", "", 0, 0, "")),
        ),
    )

    // Nova mensagem com botão cancel
    msg := tgbotapi.NewMessage(chatID, msgText)
    msg.ParseMode = "HTML"
    msg.ReplyMarkup = keyboard
    b.api.Send(msg)
}

// /search direto
func (b *Bot) handleSearchCommand(message *tgbotapi.Message) {
    b.processSearch(message.Chat.ID)
}


// performSearch searches the user's library for manga matching the query
func (b *Bot) performSearch(chatID int64, query string) {
	log.Printf("Performing search for query: %s", query)

	// Get all manga from API
	apiURL := fmt.Sprintf("http://localhost:%s/v1/mangas", config.GlobalConfigs.API.Port)
	
	resp, err := http.Get(apiURL)
	if err != nil {
		log.Printf("Error getting manga list: %v", err)
		b.sendErrorMessage(chatID, "Failed to search manga")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b.sendErrorMessage(chatID, "Failed to search manga")
		return
	}

	var result struct {
		Mangas []MangaListItem `json:"mangas"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Error decoding manga list: %v", err)
		b.sendErrorMessage(chatID, "Failed to parse search results")
		return
	}

	// Filter manga by query (case-insensitive)
	queryLower := strings.ToLower(query)
	var matches []MangaListItem
	for _, manga := range result.Mangas {
		if strings.Contains(strings.ToLower(manga.Name), queryLower) {
			matches = append(matches, manga)
		}
	}

	if len(matches) == 0 {
		reply := fmt.Sprintf("😔 No manga found matching '<b>%s</b>' in your library", query)
		msg := tgbotapi.NewMessage(chatID, reply)
		msg.ParseMode = "HTML"
		b.api.Send(msg)
		return
	}

	// Send results
	reply := fmt.Sprintf("🔍 <b>Found %d manga matching '%s':</b>\n", len(matches), query)
	msg := tgbotapi.NewMessage(chatID, reply)
	msg.ParseMode = "HTML"
	b.api.Send(msg)

	// Send each manga as a card
	for _, manga := range matches {
		b.sendMangaCard(chatID, manga)
	}
}

// handleListUnreadCommand handles the /listunread command
func (b *Bot) handleListUnreadCommand(message *tgbotapi.Message, offset int) {
	log.Printf("Processing /listunread command (offset: %d)", offset)
	b.sendMangaList(message.Chat.ID, 0, "unread", offset)
}

// handleListReadingCommand handles the /listreading command
func (b *Bot) handleListReadingCommand(message *tgbotapi.Message, offset int) {
	log.Printf("Processing /listreading command (offset: %d)", offset)
	b.sendMangaList(message.Chat.ID, 0, "reading", offset)
}

// sendMangaList sends a list of manga with images and buttons
func (b *Bot) sendMangaList(chatID int64, messageID int, listType string, offset int) {
	// Get all manga from API
	apiURL := fmt.Sprintf("http://localhost:%s/v1/mangas", config.GlobalConfigs.API.Port)
	
	resp, err := http.Get(apiURL)
	if err != nil {
		log.Printf("Error getting manga list: %v", err)
		b.sendErrorMessage(chatID, "Failed to fetch manga list")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("API returned error status: %d", resp.StatusCode)
		b.sendErrorMessage(chatID, "Failed to fetch manga list")
		return
	}

	var result struct {
		Mangas []MangaListItem `json:"mangas"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Error decoding manga list: %v", err)
		b.sendErrorMessage(chatID, "Failed to parse manga list")
		return
	}

	// Filter manga based on list type
	var filteredMangas []MangaListItem
	for _, manga := range result.Mangas {
		if listType == "unread" {
			if (manga.Status == 1 || manga.Status == 2) && hasUnreadChapters(manga) {
				filteredMangas = append(filteredMangas, manga)
			}
		} else if listType == "reading" {
			if manga.Status == 1 {
				filteredMangas = append(filteredMangas, manga)
			}
		}
	}

	if len(filteredMangas) == 0 {
		var emptyMsg string
		if listType == "unread" {
			emptyMsg = "✅ <b>All caught up!</b>\n\nYou have no manga with unread chapters."
		} else {
			emptyMsg = "📚 <b>No manga found</b>\n\nYou have no manga with reading status."
		}
		msg := tgbotapi.NewMessage(chatID, emptyMsg)
		msg.ParseMode = "HTML"
		b.api.Send(msg)
		return
	}

	// Apply pagination
	limit := 5
	start := offset
	end := offset + limit
	if end > len(filteredMangas) {
		end = len(filteredMangas)
	}
	
	hasMore := end < len(filteredMangas)
	mangasToShow := filteredMangas[start:end]

	// Send each manga as a separate message with photo and buttons
	for _, manga := range mangasToShow {
		b.sendMangaCard(chatID, manga)
	}

	// Send "List more" button if there are more results
	if hasMore {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					fmt.Sprintf("📖 Show more (%d remaining)", len(filteredMangas)-end),
					createCallbackData(ActionListMore, "", "", 0, end, listType),
				),
			),
		)
		
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("<i>Showing %d-%d of %d</i>", start+1, end, len(filteredMangas)))
		msg.ParseMode = "HTML"
		msg.ReplyMarkup = keyboard
		b.api.Send(msg)
	}
}

// sendMangaCard sends a single manga as a photo with caption and buttons
// customCaption: if provided, overrides the default caption
func (b *Bot) sendMangaCard(chatID int64, manga MangaListItem, customCaption ...string) {
	var caption string
	
	// Use custom caption if provided, otherwise build default
	if len(customCaption) > 0 && customCaption[0] != "" {
		caption = customCaption[0]
	} else {
		// Build default caption
		caption = fmt.Sprintf("<b>%s</b>\n", manga.Name)
		caption += fmt.Sprintf("%s\n\n", getStatusName(manga.Status))
		
		lastRead := "N/A"
		if manga.LastReadChapter != nil {
			lastRead = manga.LastReadChapter.Chapter
		}
		
		lastReleased := "N/A"
		if manga.LastReleasedChapter != nil {
			lastReleased = manga.LastReleasedChapter.Chapter
		}
		
		caption += fmt.Sprintf("📖 Read: Ch. %s → 🆕 New: Ch. %s", lastRead, lastReleased)
	}

	// Determine which "Read" button to show based on manga state
	var readButtonLabel string
	var readButtonURL string
	
	// Helper function to increment chapter number in URL
	getNextChapterURL := func(currentURL, currentChapter string) string {
		// Try to parse chapter as float to handle decimals (e.g., "5.5")
		chapterNum, err := strconv.ParseFloat(currentChapter, 64)
		if err != nil {
			return currentURL // Return original URL if parsing fails
		}
		
		// Increment chapter by 1
		nextChapter := chapterNum + 1
		nextChapterStr := strconv.FormatFloat(nextChapter, 'f', -1, 64)
		
		// Replace old chapter number with new one in URL
		return strings.Replace(currentURL, currentChapter, nextChapterStr, 1)
	}
	
	// Logic to determine which button to show:
	// 1. If lastRead < lastReleased → "Read Next" (next chapter after lastRead)
	// 2. If no lastRead → "Read Latest" (lastReleased)
	// 3. If lastRead == lastReleased → "Read Again" (lastRead)
	if manga.LastReadChapter != nil && manga.LastReleasedChapter != nil {
		lastReadNum, errRead := strconv.ParseFloat(manga.LastReadChapter.Chapter, 64)
		lastReleasedNum, errReleased := strconv.ParseFloat(manga.LastReleasedChapter.Chapter, 64)
		
		if errRead == nil && errReleased == nil {
			if lastReadNum < lastReleasedNum {
				// Case 1: There are unread chapters - show "Read Next"
				readButtonLabel = "📖 Read Next"
				readButtonURL = getNextChapterURL(manga.LastReadChapter.URL, manga.LastReadChapter.Chapter)
			} else if lastReadNum == lastReleasedNum {
				// Case 3: Up to date - show "Read Again"
				readButtonLabel = "🔄 Read Again"
				readButtonURL = manga.LastReadChapter.URL
			}
		}
	} else if manga.LastReadChapter == nil && manga.LastReleasedChapter != nil {
		// Case 2: No last read - show "Read Latest"
		readButtonLabel = "📖 Read Latest"
		readButtonURL = manga.LastReleasedChapter.URL
	}

	// Create inline keyboard
	var buttons [][]tgbotapi.InlineKeyboardButton
	
	// Row 1: Read button (if URL is valid and not a custom manga)
	if readButtonURL != "" && !strings.HasPrefix(readButtonURL, "http://custom_manga") {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL(readButtonLabel, readButtonURL),
		))
	}
	
	// Row 2: Set as read button
	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(
			"✓ Set as read",
			createCallbackData(ActionSetRead, strconv.Itoa(manga.ID), "", 0, 0, ""),
		),
	))
	
	// Row 3: Change status button
	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(
			"⚙️ Change Status",
			createCallbackData(ActionChangeStatus, strconv.Itoa(manga.ID), "", manga.Status, 0, ""),
		),
	))
	
	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)

	// Get cover image
	coverImg := b.getMangaCoverImage(manga.ID)
	
	if coverImg != nil {
		// Send as photo
		photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{
			Name:  fmt.Sprintf("cover_%d.jpg", manga.ID),
			Bytes: coverImg,
		})
		photo.Caption = caption
		photo.ParseMode = "HTML"
		photo.ReplyMarkup = keyboard
		
		_, err := b.api.Send(photo)
		if err != nil {
			log.Printf("Error sending photo for manga %d: %v", manga.ID, err)
			// Fallback to text message
			msg := tgbotapi.NewMessage(chatID, caption)
			msg.ParseMode = "HTML"
			msg.ReplyMarkup = keyboard
			b.api.Send(msg)
		}
	} else {
		// No cover image, send as text
		msg := tgbotapi.NewMessage(chatID, caption)
		msg.ParseMode = "HTML"
		msg.ReplyMarkup = keyboard
		b.api.Send(msg)
	}
}

// getMangaCoverImage retrieves the cover image for a manga
func (b *Bot) getMangaCoverImage(mangaID int) []byte {
	apiURL := fmt.Sprintf("http://localhost:%s/v1/manga?id=%d", 
		config.GlobalConfigs.API.Port, mangaID)
	
	resp, err := http.Get(apiURL)
	if err != nil {
		log.Printf("Error getting manga cover: %v", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var result struct {
		Manga struct {
			CoverImg []byte `json:"coverImg"`
		} `json:"manga"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Error decoding manga cover: %v", err)
		return nil
	}

	if len(result.Manga.CoverImg) == 0 {
		return nil
	}

	return result.Manga.CoverImg
}

// sendErrorMessage sends a generic error message
func (b *Bot) sendErrorMessage(chatID int64, errorMsg string) {
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ <b>Error:</b> %s", errorMsg))
	msg.ParseMode = "HTML"
	b.api.Send(msg)
}