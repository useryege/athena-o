package telegram

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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
	GetChat(ctx context.Context) (*ChatInfo, error)
	GetChatMember(ctx context.Context, userID int64) (*ChatMemberInfo, error)
	SetChatTitle(ctx context.Context, title string) error
	SetChatDescription(ctx context.Context, description string) error
	SetChatPhoto(ctx context.Context, request SetChatPhotoRequest) error
	EnsureChatProfile(ctx context.Context, config ChatProfileConfig) (*ChatProfileSyncResult, error)
	SendMessage(ctx context.Context, request SendMessageRequest) (*SendMessageResponse, error)
}

type Config struct {
	BotToken string
	ChatID   string
	BaseURL  string
	Timeout  time.Duration
}

type SendMessageRequest struct {
	Text string
}

type SendMessageResponse struct {
	MessageID int
}

type BotIdentity struct {
	ID        int64
	IsBot     bool
	FirstName string
	Username  string
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
	config Config
	bot    *tgbot.Bot
}

var _ Client = (*clientImpl)(nil)

func NewClient(config Config) (Client, error) {
	config = config.withDefaults()
	if strings.TrimSpace(config.BotToken) == "" {
		return nil, errors.New("telegram bot token is required")
	}
	if strings.TrimSpace(config.ChatID) == "" {
		return nil, errors.New("telegram chat id is required")
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
		return nil, fmt.Errorf("failed to create telegram bot client: %w", err)
	}

	return &clientImpl{
		config: config,
		bot:    botClient,
	}, nil
}

func (c Config) WithDefaults() Config {
	return c.withDefaults()
}

func (c Config) withDefaults() Config {
	c.BotToken = strings.TrimSpace(c.BotToken)
	c.ChatID = strings.TrimSpace(c.ChatID)
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
	text := strings.TrimSpace(request.Text)
	if text == "" {
		return nil, errors.New("telegram message text is required")
	}

	message, err := c.bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: c.config.ChatID,
		Text:   text,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to send telegram message: %w", err)
	}
	return &SendMessageResponse{MessageID: message.ID}, nil
}

func (c *clientImpl) GetMe(ctx context.Context) (*BotIdentity, error) {
	user, err := c.bot.GetMe(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get telegram bot identity: %w", err)
	}
	return &BotIdentity{
		ID:        user.ID,
		IsBot:     user.IsBot,
		FirstName: user.FirstName,
		Username:  user.Username,
	}, nil
}

func (c *clientImpl) GetMyName(ctx context.Context) (string, error) {
	name, err := c.bot.GetMyName(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get telegram bot name: %w", err)
	}
	return name.Name, nil
}

func (c *clientImpl) SetMyName(ctx context.Context, name string) error {
	if _, err := c.bot.SetMyName(ctx, &tgbot.SetMyNameParams{Name: name}); err != nil {
		return fmt.Errorf("failed to set telegram bot name: %w", err)
	}
	return nil
}

func (c *clientImpl) GetMyDescription(ctx context.Context) (string, error) {
	description, err := c.bot.GetMyDescription(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get telegram bot description: %w", err)
	}
	return description.Description, nil
}

func (c *clientImpl) SetMyDescription(ctx context.Context, description string) error {
	if _, err := c.bot.SetMyDescription(ctx, &tgbot.SetMyDescriptionParams{Description: description}); err != nil {
		return fmt.Errorf("failed to set telegram bot description: %w", err)
	}
	return nil
}

func (c *clientImpl) GetMyShortDescription(ctx context.Context) (string, error) {
	description, err := c.bot.GetMyShortDescription(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get telegram bot short description: %w", err)
	}
	return description.ShortDescription, nil
}

func (c *clientImpl) SetMyShortDescription(ctx context.Context, shortDescription string) error {
	if _, err := c.bot.SetMyShortDescription(ctx, &tgbot.SetMyShortDescriptionParams{ShortDescription: shortDescription}); err != nil {
		return fmt.Errorf("failed to set telegram bot short description: %w", err)
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
		return fmt.Errorf("failed to set telegram bot profile photo: %w", err)
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

	if err := c.SetMyProfilePhoto(ctx, config.ProfilePhoto); err != nil {
		return nil, err
	}
	result.ProfilePhotoUpdated = true
	return result, nil
}

func (c *clientImpl) GetChat(ctx context.Context) (*ChatInfo, error) {
	chat, err := c.bot.GetChat(ctx, &tgbot.GetChatParams{ChatID: c.config.ChatID})
	if err != nil {
		return nil, fmt.Errorf("failed to get telegram chat: %w", err)
	}
	return &ChatInfo{
		ID:          chat.ID,
		Type:        string(chat.Type),
		Title:       chat.Title,
		Description: chat.Description,
		IsForum:     chat.IsForum,
	}, nil
}

func (c *clientImpl) GetChatMember(ctx context.Context, userID int64) (*ChatMemberInfo, error) {
	member, err := c.bot.GetChatMember(ctx, &tgbot.GetChatMemberParams{ChatID: c.config.ChatID, UserID: userID})
	if err != nil {
		return nil, fmt.Errorf("failed to get telegram chat member: %w", err)
	}
	info := &ChatMemberInfo{Status: string(member.Type)}
	if member.Administrator != nil {
		info.CanChangeInfo = member.Administrator.CanChangeInfo
		info.CanManageTopics = member.Administrator.CanManageTopics
	}
	return info, nil
}

func (c *clientImpl) SetChatTitle(ctx context.Context, title string) error {
	if _, err := c.bot.SetChatTitle(ctx, &tgbot.SetChatTitleParams{ChatID: c.config.ChatID, Title: title}); err != nil {
		return fmt.Errorf("failed to set telegram chat title: %w", err)
	}
	return nil
}

func (c *clientImpl) SetChatDescription(ctx context.Context, description string) error {
	if _, err := c.bot.SetChatDescription(ctx, &tgbot.SetChatDescriptionParams{ChatID: c.config.ChatID, Description: description}); err != nil {
		return fmt.Errorf("failed to set telegram chat description: %w", err)
	}
	return nil
}

func (c *clientImpl) SetChatPhoto(ctx context.Context, request SetChatPhotoRequest) error {
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
	if _, err := c.bot.SetChatPhoto(ctx, &tgbot.SetChatPhotoParams{ChatID: c.config.ChatID, Photo: photo}); err != nil {
		return fmt.Errorf("failed to set telegram chat photo: %w", err)
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

	chat, err := c.GetChat(ctx)
	if err != nil {
		return nil, err
	}
	if chat.Type != string(models.ChatTypeGroup) && chat.Type != string(models.ChatTypeSupergroup) {
		return nil, fmt.Errorf("telegram chat type %q is not a group or supergroup", chat.Type)
	}

	member, err := c.GetChatMember(ctx, bot.ID)
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
		if err := c.SetChatTitle(ctx, config.Title); err != nil {
			return nil, err
		}
		result.TitleUpdated = true
	}
	if chat.Description != config.Description {
		if err := c.SetChatDescription(ctx, config.Description); err != nil {
			return nil, err
		}
		result.DescriptionUpdated = true
	}
	if err := c.SetChatPhoto(ctx, config.Photo); err != nil {
		return nil, err
	}
	result.PhotoUpdated = true
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
	if config.ProfilePhoto.Filename == "" {
		return BotProfileConfig{}, errors.New("telegram bot profile photo filename is required")
	}
	if len(config.ProfilePhoto.Data) == 0 {
		return BotProfileConfig{}, errors.New("telegram bot profile photo data is required")
	}
	return config, nil
}

func normalizeChatProfileConfig(config ChatProfileConfig) (ChatProfileConfig, error) {
	config.Title = strings.TrimSpace(config.Title)
	config.Description = strings.TrimSpace(config.Description)
	config.Photo.Filename = strings.TrimSpace(config.Photo.Filename)

	if config.Title == "" {
		return ChatProfileConfig{}, errors.New("telegram chat title is required")
	}
	if utf8.RuneCountInString(config.Title) > 128 {
		return ChatProfileConfig{}, errors.New("telegram chat title must be at most 128 characters")
	}
	if utf8.RuneCountInString(config.Description) > 255 {
		return ChatProfileConfig{}, errors.New("telegram chat description must be at most 255 characters")
	}
	if config.Photo.Filename == "" {
		return ChatProfileConfig{}, errors.New("telegram chat photo filename is required")
	}
	if len(config.Photo.Data) == 0 {
		return ChatProfileConfig{}, errors.New("telegram chat photo data is required")
	}
	return config, nil
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
