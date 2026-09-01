package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	DefaultBaseURL = "https://api.telegram.org"
	DefaultTimeout = 10 * time.Second
)

var ErrRecipientUnreachable = errors.New("telegram recipient unreachable")

type Client interface {
	GetMe(ctx context.Context) (*BotIdentity, error)
	GetMyName(ctx context.Context) (string, error)
	SetMyName(ctx context.Context, name string) error
	GetMyDescription(ctx context.Context) (string, error)
	SetMyDescription(ctx context.Context, description string) error
	GetMyShortDescription(ctx context.Context) (string, error)
	SetMyShortDescription(ctx context.Context, shortDescription string) error
	SetMyProfilePhoto(ctx context.Context, request SetMyProfilePhotoRequest) error
	EnsureBotProfile(ctx context.Context, config BotProfileConfig) (*BotProfileSyncResult, error)
	GetWebhookInfo(ctx context.Context) (*WebhookInfo, error)
	PollUpdates(ctx context.Context, request PollUpdatesRequest) ([]Update, error)
	GetChat(ctx context.Context, chatID string) (*ChatInfo, error)
	GetChatMember(ctx context.Context, chatID string, userID int64) (*ChatMemberInfo, error)
	SetChatTitle(ctx context.Context, chatID string, title string) error
	SetChatDescription(ctx context.Context, chatID string, description string) error
	SetChatPhoto(ctx context.Context, chatID string, request SetChatPhotoRequest) error
	EnsureChatProfile(ctx context.Context, config ChatProfileConfig) (*ChatProfileSyncResult, error)
	CreateForumTopic(ctx context.Context, request CreateForumTopicRequest) (*ForumTopic, error)
	SendMessage(ctx context.Context, request SendMessageRequest) (*SendMessageResponse, error)
}

type Config struct {
	BotToken string
	BaseURL  string
	Timeout  time.Duration
}

type SendMessageRequest struct {
	ChatID          string
	Text            string
	MessageThreadID int
	ParseMode       models.ParseMode
}

type SendMessageResponse struct {
	MessageID int
}

type CreateForumTopicRequest struct {
	ChatID string
	Name   string
}

type ForumTopic struct {
	MessageThreadID int
	Name            string
}

type BotIdentity struct {
	ID        int64
	IsBot     bool
	FirstName string
	Username  string
}

type WebhookInfo struct {
	URL                string
	PendingUpdateCount int
}

type PollUpdatesRequest struct {
	Offset  int64
	Timeout time.Duration
}

type Update struct {
	ID           int64
	Message      *Message
	MyChatMember *MyChatMemberUpdate
}

type Message struct {
	ChatID    int64
	ChatType  string
	UserID    int64
	IsBot     bool
	Username  string
	FirstName string
	LastName  string
	Text      string
}

type MyChatMemberUpdate struct {
	ChatID    int64
	ChatType  string
	UserID    int64
	IsBot     bool
	Username  string
	FirstName string
	LastName  string
	NewStatus string
}

type SetMyProfilePhotoRequest struct {
	Filename string
	Data     []byte
}

type BotProfileConfig struct {
	Name             string
	Description      string
	ShortDescription string
	ProfilePhoto     SetMyProfilePhotoRequest
}

type BotProfileSyncResult struct {
	Bot                     BotIdentity
	NameUpdated             bool
	DescriptionUpdated      bool
	ShortDescriptionUpdated bool
	ProfilePhotoUpdated     bool
}

type ChatInfo struct {
	ID          int64
	Type        string
	Title       string
	Description string
	IsForum     bool
}

type ChatMemberInfo struct {
	Status          string
	CanChangeInfo   bool
	CanManageTopics bool
}

type SetChatPhotoRequest struct {
	Filename string
	Data     []byte
}

type ChatProfileConfig struct {
	ChatID      string
	Title       string
	Description string
	Photo       SetChatPhotoRequest
}

type ChatProfileSyncResult struct {
	ChatID             int64
	ChatType           string
	IsForum            bool
	TitleUpdated       bool
	DescriptionUpdated bool
	PhotoUpdated       bool
}

type clientImpl struct {
	config         Config
	bot            *tgbot.Bot
	pollHTTPClient *http.Client
}

var _ Client = (*clientImpl)(nil)

func NewClient(config Config) (Client, error) {
	config = config.withDefaults()
	if strings.TrimSpace(config.BotToken) == "" {
		return nil, errors.New("telegram bot token is required")
	}
	if _, err := url.ParseRequestURI(config.BaseURL); err != nil {
		return nil, fmt.Errorf("invalid telegram base url: %w", err)
	}

	botClient, err := tgbot.New(
		config.BotToken,
		tgbot.WithServerURL(config.BaseURL),
		tgbot.WithHTTPClient(config.Timeout, &http.Client{Timeout: config.Timeout}),
		tgbot.WithSkipGetMe(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot client: %w", sanitizeTelegramRequestError(err))
	}

	return &clientImpl{
		config:         config,
		bot:            botClient,
		pollHTTPClient: &http.Client{},
	}, nil
}

func (c Config) WithDefaults() Config {
	return c.withDefaults()
}

func (c Config) withDefaults() Config {
	c.BotToken = strings.TrimSpace(c.BotToken)
	c.BaseURL = strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if c.BaseURL == "" {
		c.BaseURL = DefaultBaseURL
	}
	if c.Timeout <= 0 {
		c.Timeout = DefaultTimeout
	}
	return c
}

func (c *clientImpl) SendMessage(ctx context.Context, request SendMessageRequest) (*SendMessageResponse, error) {
	chatID, err := normalizeChatID(request.ChatID)
	if err != nil {
		return nil, err
	}
	text := strings.TrimSpace(request.Text)
	if text == "" {
		return nil, errors.New("telegram message text is required")
	}

	params := &tgbot.SendMessageParams{
		ChatID:          chatID,
		Text:            text,
		MessageThreadID: request.MessageThreadID,
		ParseMode:       request.ParseMode,
	}

	message, err := c.bot.SendMessage(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to send telegram message: %w", sanitizeTelegramRequestError(err))
	}
	return &SendMessageResponse{MessageID: message.ID}, nil
}

func (c *clientImpl) CreateForumTopic(ctx context.Context, request CreateForumTopicRequest) (*ForumTopic, error) {
	chatID, err := normalizeChatID(request.ChatID)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return nil, errors.New("telegram forum topic name is required")
	}
	if utf8.RuneCountInString(name) > 128 {
		return nil, errors.New("telegram forum topic name must be at most 128 characters")
	}

	topic, err := c.bot.CreateForumTopic(ctx, &tgbot.CreateForumTopicParams{
		ChatID: chatID,
		Name:   name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram forum topic: %w", sanitizeTelegramRequestError(err))
	}
	return &ForumTopic{MessageThreadID: topic.MessageThreadID, Name: topic.Name}, nil
}

func (c *clientImpl) GetMe(ctx context.Context) (*BotIdentity, error) {
	user, err := c.bot.GetMe(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get telegram bot identity: %w", sanitizeTelegramRequestError(err))
	}
	return &BotIdentity{
		ID:        user.ID,
		IsBot:     user.IsBot,
		FirstName: user.FirstName,
		Username:  user.Username,
	}, nil
}

func (c *clientImpl) GetWebhookInfo(ctx context.Context) (*WebhookInfo, error) {
	info, err := c.bot.GetWebhookInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get telegram webhook info: %w", sanitizeTelegramRequestError(err))
	}
	return &WebhookInfo{
		URL:                strings.TrimSpace(info.URL),
		PendingUpdateCount: info.PendingUpdateCount,
	}, nil
}

func (c *clientImpl) PollUpdates(ctx context.Context, request PollUpdatesRequest) ([]Update, error) {
	if request.Offset < 0 {
		return nil, errors.New("telegram update offset must not be negative")
	}
	pollTimeout := request.Timeout
	if pollTimeout <= 0 {
		pollTimeout = 30 * time.Second
	}
	if pollTimeout > 50*time.Second {
		return nil, errors.New("telegram update poll timeout must be at most 50 seconds")
	}

	payload, err := json.Marshal(struct {
		Offset         int64    `json:"offset"`
		Timeout        int      `json:"timeout"`
		AllowedUpdates []string `json:"allowed_updates"`
	}{
		Offset:         request.Offset,
		Timeout:        int(pollTimeout / time.Second),
		AllowedUpdates: []string{models.AllowedUpdateMessage, models.AllowedUpdateMyChatMember},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode telegram getUpdates request: %w", err)
	}

	requestCtx, cancel := context.WithTimeout(ctx, pollTimeout+c.config.Timeout)
	defer cancel()
	httpRequest, err := http.NewRequestWithContext(
		requestCtx,
		http.MethodPost,
		c.config.BaseURL+"/bot"+c.config.BotToken+"/getUpdates",
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram getUpdates request: %w", sanitizeTelegramRequestError(err))
	}
	httpRequest.Header.Set("Content-Type", "application/json")

	httpResponse, err := c.pollHTTPClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to poll telegram updates: %w", sanitizeTelegramRequestError(err))
	}
	defer func() {
		_ = httpResponse.Body.Close()
	}()
	body, err := io.ReadAll(io.LimitReader(httpResponse.Body, 16<<20))
	if err != nil {
		return nil, fmt.Errorf("failed to read telegram getUpdates response: %w", err)
	}

	var response telegramAPIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode telegram getUpdates response: %w", err)
	}
	if !response.OK {
		return nil, telegramAPIResponseError(response)
	}

	var telegramUpdates []models.Update
	if len(response.Result) > 0 {
		if err := json.Unmarshal(response.Result, &telegramUpdates); err != nil {
			return nil, fmt.Errorf("failed to decode telegram updates: %w", err)
		}
	}
	updates := make([]Update, 0, len(telegramUpdates))
	for _, telegramUpdate := range telegramUpdates {
		update := Update{ID: telegramUpdate.ID}
		if telegramUpdate.Message != nil {
			message := telegramUpdate.Message
			mapped := &Message{
				ChatID:   message.Chat.ID,
				ChatType: string(message.Chat.Type),
				Text:     message.Text,
			}
			if message.From != nil {
				mapped.UserID = message.From.ID
				mapped.IsBot = message.From.IsBot
				mapped.Username = message.From.Username
				mapped.FirstName = message.From.FirstName
				mapped.LastName = message.From.LastName
			}
			update.Message = mapped
		}
		if telegramUpdate.MyChatMember != nil {
			member := telegramUpdate.MyChatMember
			update.MyChatMember = &MyChatMemberUpdate{
				ChatID:    member.Chat.ID,
				ChatType:  string(member.Chat.Type),
				UserID:    member.From.ID,
				IsBot:     member.From.IsBot,
				Username:  member.From.Username,
				FirstName: member.From.FirstName,
				LastName:  member.From.LastName,
				NewStatus: string(member.NewChatMember.Type),
			}
		}
		updates = append(updates, update)
	}
	return updates, nil
}

type telegramAPIResponse struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result"`
	Description string          `json:"description"`
	ErrorCode   int             `json:"error_code"`
	Parameters  struct {
		RetryAfter int `json:"retry_after"`
	} `json:"parameters"`
}

func telegramAPIResponseError(response telegramAPIResponse) error {
	description := strings.TrimSpace(response.Description)
	if telegramRecipientUnreachableDescription(description) {
		return telegramRecipientUnreachableError(tgbot.ErrorBadRequest)
	}
	switch response.ErrorCode {
	case http.StatusForbidden:
		return telegramRecipientUnreachableError(tgbot.ErrorForbidden)
	case http.StatusBadRequest:
		return fmt.Errorf("telegram request rejected: %w", tgbot.ErrorBadRequest)
	case http.StatusUnauthorized:
		return fmt.Errorf("telegram request unauthorized: %w", tgbot.ErrorUnauthorized)
	case http.StatusNotFound:
		return telegramRecipientUnreachableError(tgbot.ErrorNotFound)
	case http.StatusConflict:
		return fmt.Errorf("telegram request conflict: %w", tgbot.ErrorConflict)
	case http.StatusTooManyRequests:
		return &tgbot.TooManyRequestsError{
			Message:    tgbot.ErrorTooManyRequests.Error(),
			RetryAfter: response.Parameters.RetryAfter,
		}
	default:
		return fmt.Errorf("telegram request failed with provider code %d", response.ErrorCode)
	}
}

func sanitizeTelegramRequestError(err error) error {
	if err == nil {
		return errors.New("telegram request failed")
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	var rateLimitError *tgbot.TooManyRequestsError
	if errors.As(err, &rateLimitError) {
		return &tgbot.TooManyRequestsError{
			Message:    tgbot.ErrorTooManyRequests.Error(),
			RetryAfter: rateLimitError.RetryAfter,
		}
	}
	if errors.Is(err, tgbot.ErrorForbidden) {
		return telegramRecipientUnreachableError(tgbot.ErrorForbidden)
	}
	if errors.Is(err, tgbot.ErrorNotFound) {
		return telegramRecipientUnreachableError(tgbot.ErrorNotFound)
	}
	if errors.Is(err, tgbot.ErrorBadRequest) {
		if telegramRecipientUnreachableDescription(err.Error()) {
			return telegramRecipientUnreachableError(tgbot.ErrorBadRequest)
		}
		return fmt.Errorf("telegram request rejected: %w", tgbot.ErrorBadRequest)
	}
	if errors.Is(err, tgbot.ErrorUnauthorized) {
		return fmt.Errorf("telegram request unauthorized: %w", tgbot.ErrorUnauthorized)
	}
	if errors.Is(err, tgbot.ErrorConflict) {
		return fmt.Errorf("telegram request conflict: %w", tgbot.ErrorConflict)
	}
	var migrateError *tgbot.MigrateError
	if errors.As(err, &migrateError) {
		return fmt.Errorf("telegram chat migration required: %w", tgbot.ErrorBadRequest)
	}
	var requestError *url.Error
	if errors.As(err, &requestError) {
		if requestError.Err == nil {
			return errors.New("telegram HTTP request failed")
		}
		cause := sanitizeTelegramRequestError(requestError.Err)
		op := strings.TrimSpace(requestError.Op)
		if op == "" {
			return cause
		}
		return fmt.Errorf("telegram HTTP %s failed: %w", op, cause)
	}
	return errors.New("telegram request failed")
}

func telegramRecipientUnreachableDescription(value string) bool {
	value = strings.ToLower(value)
	return strings.Contains(value, "chat not found") ||
		strings.Contains(value, "bot was blocked by the user") ||
		strings.Contains(value, "user is deactivated")
}

func telegramRecipientUnreachableError(providerError error) error {
	return fmt.Errorf("%w: %w", ErrRecipientUnreachable, providerError)
}

func (c *clientImpl) GetMyName(ctx context.Context) (string, error) {
	name, err := c.bot.GetMyName(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get telegram bot name: %w", sanitizeTelegramRequestError(err))
	}
	return name.Name, nil
}

func (c *clientImpl) SetMyName(ctx context.Context, name string) error {
	if _, err := c.bot.SetMyName(ctx, &tgbot.SetMyNameParams{Name: name}); err != nil {
		return fmt.Errorf("failed to set telegram bot name: %w", sanitizeTelegramRequestError(err))
	}
	return nil
}

func (c *clientImpl) GetMyDescription(ctx context.Context) (string, error) {
	description, err := c.bot.GetMyDescription(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get telegram bot description: %w", sanitizeTelegramRequestError(err))
	}
	return description.Description, nil
}

func (c *clientImpl) SetMyDescription(ctx context.Context, description string) error {
	if _, err := c.bot.SetMyDescription(ctx, &tgbot.SetMyDescriptionParams{Description: description}); err != nil {
		return fmt.Errorf("failed to set telegram bot description: %w", sanitizeTelegramRequestError(err))
	}
	return nil
}

func (c *clientImpl) GetMyShortDescription(ctx context.Context) (string, error) {
	description, err := c.bot.GetMyShortDescription(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get telegram bot short description: %w", sanitizeTelegramRequestError(err))
	}
	return description.ShortDescription, nil
}

func (c *clientImpl) SetMyShortDescription(ctx context.Context, shortDescription string) error {
	if _, err := c.bot.SetMyShortDescription(ctx, &tgbot.SetMyShortDescriptionParams{ShortDescription: shortDescription}); err != nil {
		return fmt.Errorf("failed to set telegram bot short description: %w", sanitizeTelegramRequestError(err))
	}
	return nil
}

func (c *clientImpl) SetMyProfilePhoto(ctx context.Context, request SetMyProfilePhotoRequest) error {
	request.Filename = strings.TrimSpace(request.Filename)
	if request.Filename == "" {
		return errors.New("telegram bot profile photo filename is required")
	}
	if len(request.Data) == 0 {
		return errors.New("telegram bot profile photo data is required")
	}
	photo := models.InputProfilePhotoStatic{
		Photo:           "attach://" + request.Filename,
		MediaAttachment: bytes.NewReader(request.Data),
	}
	if _, err := c.bot.SetMyProfilePhoto(ctx, &tgbot.SetMyProfilePhotoParams{Photo: photo}); err != nil {
		return fmt.Errorf("failed to set telegram bot profile photo: %w", sanitizeTelegramRequestError(err))
	}
	return nil
}

func (c *clientImpl) EnsureBotProfile(ctx context.Context, config BotProfileConfig) (*BotProfileSyncResult, error) {
	config, err := normalizeBotProfileConfig(config)
	if err != nil {
		return nil, err
	}

	bot, err := c.GetMe(ctx)
	if err != nil {
		return nil, err
	}
	if !bot.IsBot {
		return nil, errors.New("telegram identity is not a bot")
	}
	result := &BotProfileSyncResult{Bot: *bot}

	currentName, err := c.GetMyName(ctx)
	if err != nil {
		return nil, err
	}
	if currentName != config.Name {
		if err := c.SetMyName(ctx, config.Name); err != nil {
			return nil, err
		}
		result.NameUpdated = true
	}

	currentDescription, err := c.GetMyDescription(ctx)
	if err != nil {
		return nil, err
	}
	if currentDescription != config.Description {
		if err := c.SetMyDescription(ctx, config.Description); err != nil {
			return nil, err
		}
		result.DescriptionUpdated = true
	}

	currentShortDescription, err := c.GetMyShortDescription(ctx)
	if err != nil {
		return nil, err
	}
	if currentShortDescription != config.ShortDescription {
		if err := c.SetMyShortDescription(ctx, config.ShortDescription); err != nil {
			return nil, err
		}
		result.ShortDescriptionUpdated = true
	}

	if botProfilePhotoConfigured(config.ProfilePhoto) {
		if err := c.SetMyProfilePhoto(ctx, config.ProfilePhoto); err != nil {
			return nil, err
		}
		result.ProfilePhotoUpdated = true
	}
	return result, nil
}

func (c *clientImpl) GetChat(ctx context.Context, rawChatID string) (*ChatInfo, error) {
	chatID, err := normalizeChatID(rawChatID)
	if err != nil {
		return nil, err
	}
	chat, err := c.bot.GetChat(ctx, &tgbot.GetChatParams{ChatID: chatID})
	if err != nil {
		return nil, fmt.Errorf("failed to get telegram chat: %w", sanitizeTelegramRequestError(err))
	}
	return &ChatInfo{
		ID:          chat.ID,
		Type:        string(chat.Type),
		Title:       chat.Title,
		Description: chat.Description,
		IsForum:     chat.IsForum,
	}, nil
}

func (c *clientImpl) GetChatMember(ctx context.Context, rawChatID string, userID int64) (*ChatMemberInfo, error) {
	chatID, err := normalizeChatID(rawChatID)
	if err != nil {
		return nil, err
	}
	member, err := c.bot.GetChatMember(ctx, &tgbot.GetChatMemberParams{ChatID: chatID, UserID: userID})
	if err != nil {
		return nil, fmt.Errorf("failed to get telegram chat member: %w", sanitizeTelegramRequestError(err))
	}
	info := &ChatMemberInfo{Status: string(member.Type)}
	if member.Administrator != nil {
		info.CanChangeInfo = member.Administrator.CanChangeInfo
		info.CanManageTopics = member.Administrator.CanManageTopics
	}
	return info, nil
}

func (c *clientImpl) SetChatTitle(ctx context.Context, rawChatID string, title string) error {
	chatID, err := normalizeChatID(rawChatID)
	if err != nil {
		return err
	}
	if _, err := c.bot.SetChatTitle(ctx, &tgbot.SetChatTitleParams{ChatID: chatID, Title: title}); err != nil {
		return fmt.Errorf("failed to set telegram chat title: %w", sanitizeTelegramRequestError(err))
	}
	return nil
}

func (c *clientImpl) SetChatDescription(ctx context.Context, rawChatID string, description string) error {
	chatID, err := normalizeChatID(rawChatID)
	if err != nil {
		return err
	}
	if _, err := c.bot.SetChatDescription(ctx, &tgbot.SetChatDescriptionParams{ChatID: chatID, Description: description}); err != nil {
		return fmt.Errorf("failed to set telegram chat description: %w", sanitizeTelegramRequestError(err))
	}
	return nil
}

func (c *clientImpl) SetChatPhoto(ctx context.Context, rawChatID string, request SetChatPhotoRequest) error {
	chatID, err := normalizeChatID(rawChatID)
	if err != nil {
		return err
	}
	request.Filename = strings.TrimSpace(request.Filename)
	if request.Filename == "" {
		return errors.New("telegram chat photo filename is required")
	}
	if len(request.Data) == 0 {
		return errors.New("telegram chat photo data is required")
	}
	photo := &models.InputFileUpload{
		Filename: request.Filename,
		Data:     bytes.NewReader(request.Data),
	}
	if _, err := c.bot.SetChatPhoto(ctx, &tgbot.SetChatPhotoParams{ChatID: chatID, Photo: photo}); err != nil {
		return fmt.Errorf("failed to set telegram chat photo: %w", sanitizeTelegramRequestError(err))
	}
	return nil
}

func (c *clientImpl) EnsureChatProfile(ctx context.Context, config ChatProfileConfig) (*ChatProfileSyncResult, error) {
	config, err := normalizeChatProfileConfig(config)
	if err != nil {
		return nil, err
	}

	bot, err := c.GetMe(ctx)
	if err != nil {
		return nil, err
	}
	if !bot.IsBot {
		return nil, errors.New("telegram identity is not a bot")
	}

	chat, err := c.GetChat(ctx, config.ChatID)
	if err != nil {
		return nil, err
	}
	if chat.Type != string(models.ChatTypeGroup) && chat.Type != string(models.ChatTypeSupergroup) {
		return nil, fmt.Errorf("telegram chat type %q is not a group or supergroup", chat.Type)
	}

	member, err := c.GetChatMember(ctx, config.ChatID, bot.ID)
	if err != nil {
		return nil, err
	}
	if err := validateChatAdminRights(chat, member); err != nil {
		return nil, err
	}

	result := &ChatProfileSyncResult{
		ChatID:   chat.ID,
		ChatType: chat.Type,
		IsForum:  chat.IsForum,
	}
	if chat.Title != config.Title {
		if err := c.SetChatTitle(ctx, config.ChatID, config.Title); err != nil {
			return nil, err
		}
		result.TitleUpdated = true
	}
	if chat.Description != config.Description {
		if err := c.SetChatDescription(ctx, config.ChatID, config.Description); err != nil {
			return nil, err
		}
		result.DescriptionUpdated = true
	}
	if chatPhotoConfigured(config.Photo) {
		if err := c.SetChatPhoto(ctx, config.ChatID, config.Photo); err != nil {
			return nil, err
		}
		result.PhotoUpdated = true
	}
	return result, nil
}

func normalizeBotProfileConfig(config BotProfileConfig) (BotProfileConfig, error) {
	config.Name = strings.TrimSpace(config.Name)
	config.Description = strings.TrimSpace(config.Description)
	config.ShortDescription = strings.TrimSpace(config.ShortDescription)
	config.ProfilePhoto.Filename = strings.TrimSpace(config.ProfilePhoto.Filename)

	if config.Name == "" {
		return BotProfileConfig{}, errors.New("telegram bot profile name is required")
	}
	if utf8.RuneCountInString(config.Name) > 64 {
		return BotProfileConfig{}, errors.New("telegram bot profile name must be at most 64 characters")
	}
	if utf8.RuneCountInString(config.Description) > 512 {
		return BotProfileConfig{}, errors.New("telegram bot profile description must be at most 512 characters")
	}
	if utf8.RuneCountInString(config.ShortDescription) > 120 {
		return BotProfileConfig{}, errors.New("telegram bot profile short description must be at most 120 characters")
	}
	if !botProfilePhotoConfigured(config.ProfilePhoto) && len(config.ProfilePhoto.Data) == 0 {
		return config, nil
	}
	if config.ProfilePhoto.Filename == "" {
		return BotProfileConfig{}, errors.New("telegram bot profile photo filename is required")
	}
	if len(config.ProfilePhoto.Data) == 0 {
		return BotProfileConfig{}, errors.New("telegram bot profile photo data is required")
	}
	return config, nil
}

func normalizeChatProfileConfig(config ChatProfileConfig) (ChatProfileConfig, error) {
	config.ChatID = strings.TrimSpace(config.ChatID)
	config.Title = strings.TrimSpace(config.Title)
	config.Description = strings.TrimSpace(config.Description)
	config.Photo.Filename = strings.TrimSpace(config.Photo.Filename)

	if _, err := normalizeChatID(config.ChatID); err != nil {
		return ChatProfileConfig{}, err
	}
	if config.Title == "" {
		return ChatProfileConfig{}, errors.New("telegram chat title is required")
	}
	if utf8.RuneCountInString(config.Title) > 128 {
		return ChatProfileConfig{}, errors.New("telegram chat title must be at most 128 characters")
	}
	if utf8.RuneCountInString(config.Description) > 255 {
		return ChatProfileConfig{}, errors.New("telegram chat description must be at most 255 characters")
	}
	if !chatPhotoConfigured(config.Photo) && len(config.Photo.Data) == 0 {
		return config, nil
	}
	if config.Photo.Filename == "" {
		return ChatProfileConfig{}, errors.New("telegram chat photo filename is required")
	}
	if len(config.Photo.Data) == 0 {
		return ChatProfileConfig{}, errors.New("telegram chat photo data is required")
	}
	return config, nil
}

func normalizeChatID(value string) (string, error) {
	chatID := strings.TrimSpace(value)
	if chatID == "" {
		return "", errors.New("telegram chat id is required")
	}
	return chatID, nil
}

func botProfilePhotoConfigured(photo SetMyProfilePhotoRequest) bool {
	return strings.TrimSpace(photo.Filename) != ""
}

func chatPhotoConfigured(photo SetChatPhotoRequest) bool {
	return strings.TrimSpace(photo.Filename) != ""
}

func validateChatAdminRights(chat *ChatInfo, member *ChatMemberInfo) error {
	switch member.Status {
	case string(models.ChatMemberTypeOwner):
		return nil
	case string(models.ChatMemberTypeAdministrator):
		if !member.CanChangeInfo {
			return errors.New("telegram bot administrator must have can_change_info")
		}
		if chat.IsForum && !member.CanManageTopics {
			return errors.New("telegram bot administrator must have can_manage_topics for forum chats")
		}
		return nil
	case string(models.ChatMemberTypeMember), string(models.ChatMemberTypeRestricted):
		return fmt.Errorf("telegram bot is in chat but is not an administrator: %s", member.Status)
	case string(models.ChatMemberTypeLeft), string(models.ChatMemberTypeBanned):
		return fmt.Errorf("telegram bot is not in chat: %s", member.Status)
	default:
		return fmt.Errorf("telegram bot chat member status is not administrator: %s", member.Status)
	}
}
