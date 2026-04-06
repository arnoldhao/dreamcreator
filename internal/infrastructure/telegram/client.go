package telegram

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

const defaultTimeout = 10 * time.Second

type MenuCommand = telego.BotCommand

type BotConfig struct {
	Token      string
	HTTPClient *http.Client
	APIServer  string
	Debug      bool
}

func NewBot(cfg BotConfig) (*telego.Bot, error) {
	token := strings.TrimSpace(cfg.Token)
	if token == "" {
		return nil, fmt.Errorf("telegram bot token is required")
	}
	options := make([]telego.BotOption, 0, 4)
	if cfg.HTTPClient != nil {
		options = append(options, telego.WithHTTPClient(cfg.HTTPClient))
	}
	if apiServer := strings.TrimSpace(cfg.APIServer); apiServer != "" {
		options = append(options, telego.WithAPIServer(strings.TrimRight(apiServer, "/")))
	}
	options = append(options, telego.WithLogger(newFilteredLogger(token, cfg.Debug)))
	return telego.NewBot(token, options...)
}

type Client struct {
	token      string
	httpClient *http.Client
	apiServer  string
	bot        *telego.Bot
	initErr    error
	mu         sync.Mutex
}

func NewClient(token string, client *http.Client) *Client {
	return NewClientWithOptions(token, client, ClientOptions{})
}

type ClientOptions struct {
	APIServer string
}

func NewClientWithOptions(token string, client *http.Client, options ClientOptions) *Client {
	if client == nil {
		client = &http.Client{Timeout: defaultTimeout}
	}
	return &Client{
		token:      strings.TrimSpace(token),
		httpClient: client,
		apiServer:  strings.TrimSpace(options.APIServer),
	}
}

type GetUpdatesParams struct {
	Offset         int64
	Limit          int
	TimeoutSeconds int
	AllowedUpdates []string
}

type SendMessageParams struct {
	ChatID              int64
	Chat                string
	Text                string
	ReplyToMessageID    int64
	MessageThreadID     int64
	DisableLinkPreview  bool
	DisableNotification bool
	ParseMode           string
	Buttons             [][]InlineButton
}

type SendPhotoParams struct {
	ChatID              int64
	Chat                string
	Photo               string
	PhotoData           []byte
	Filename            string
	Caption             string
	ReplyToMessageID    int64
	MessageThreadID     int64
	DisableNotification bool
	Buttons             [][]InlineButton
}

type SendDocumentParams struct {
	ChatID              int64
	Chat                string
	Document            string
	DocumentData        []byte
	Filename            string
	Caption             string
	ReplyToMessageID    int64
	MessageThreadID     int64
	DisableNotification bool
	Buttons             [][]InlineButton
}

type SendPollParams struct {
	ChatID              int64
	Chat                string
	Question            string
	Options             []string
	IsAnonymous         *bool
	ReplyToMessageID    int64
	MessageThreadID     int64
	DisableNotification bool
	OpenPeriodSeconds   int
	AllowMultiple       bool
}

type InlineButton struct {
	Text         string
	CallbackData string
	Style        string
}

type SetMessageReactionParams struct {
	ChatID    int64
	Chat      string
	MessageID int64
	Emoji     string
	Remove    bool
}

type DeleteMessageParams struct {
	ChatID    int64
	Chat      string
	MessageID int64
}

type EditMessageParams struct {
	ChatID    int64
	Chat      string
	MessageID int64
	Text      string
	ParseMode string
	Buttons   [][]InlineButton
}

type SetWebhookParams struct {
	URL               string
	SecretToken       string
	AllowedUpdates    []string
	DropPendingUpdate bool
}

func (api *Client) botOrError() (*telego.Bot, error) {
	if api == nil {
		return nil, fmt.Errorf("telegram api client unavailable")
	}
	api.mu.Lock()
	defer api.mu.Unlock()
	if api.bot != nil {
		return api.bot, nil
	}
	if api.initErr != nil {
		return nil, api.initErr
	}
	bot, err := NewBot(BotConfig{
		Token:      api.token,
		HTTPClient: api.httpClient,
		APIServer:  api.apiServer,
	})
	if err != nil {
		api.initErr = err
		return nil, err
	}
	api.bot = bot
	return api.bot, nil
}

func (api *Client) SetMyCommands(ctx context.Context, commands []MenuCommand) error {
	bot, err := api.botOrError()
	if err != nil {
		return err
	}
	return bot.SetMyCommands(ctx, &telego.SetMyCommandsParams{Commands: commands})
}

func (api *Client) SetMyCommandsScoped(
	ctx context.Context,
	commands []MenuCommand,
	scope telego.BotCommandScope,
	languageCode string,
) error {
	bot, err := api.botOrError()
	if err != nil {
		return err
	}
	params := &telego.SetMyCommandsParams{Commands: commands}
	if scope != nil {
		params.Scope = scope
	}
	if trimmedLanguage := strings.TrimSpace(languageCode); trimmedLanguage != "" {
		params.LanguageCode = trimmedLanguage
	}
	return bot.SetMyCommands(ctx, params)
}

func (api *Client) DeleteMyCommands(ctx context.Context) error {
	bot, err := api.botOrError()
	if err != nil {
		return err
	}
	return bot.DeleteMyCommands(ctx, &telego.DeleteMyCommandsParams{})
}

func (api *Client) DeleteMyCommandsScoped(
	ctx context.Context,
	scope telego.BotCommandScope,
	languageCode string,
) error {
	bot, err := api.botOrError()
	if err != nil {
		return err
	}
	params := &telego.DeleteMyCommandsParams{}
	if scope != nil {
		params.Scope = scope
	}
	if trimmedLanguage := strings.TrimSpace(languageCode); trimmedLanguage != "" {
		params.LanguageCode = trimmedLanguage
	}
	return bot.DeleteMyCommands(ctx, params)
}

func (api *Client) GetMe(ctx context.Context) (User, error) {
	bot, err := api.botOrError()
	if err != nil {
		return User{}, err
	}
	user, err := bot.GetMe(ctx)
	if err != nil {
		return User{}, err
	}
	if user == nil {
		return User{}, nil
	}
	return *user, nil
}

func (api *Client) GetUpdates(ctx context.Context, params GetUpdatesParams) ([]Update, error) {
	bot, err := api.botOrError()
	if err != nil {
		return nil, err
	}
	request := &telego.GetUpdatesParams{}
	if params.Offset > 0 {
		request.Offset = int(params.Offset)
	}
	if params.Limit > 0 {
		request.Limit = params.Limit
	}
	if params.TimeoutSeconds > 0 {
		request.Timeout = params.TimeoutSeconds
	}
	if len(params.AllowedUpdates) > 0 {
		request.AllowedUpdates = params.AllowedUpdates
	}
	return bot.GetUpdates(ctx, request)
}

func (api *Client) SendMessage(ctx context.Context, params SendMessageParams) (Message, error) {
	bot, err := api.botOrError()
	if err != nil {
		return Message{}, err
	}
	msg := tu.Message(resolveTelegramChatID(params.ChatID, params.Chat), params.Text)
	if params.ReplyToMessageID > 0 {
		msg.ReplyParameters = &telego.ReplyParameters{MessageID: int(params.ReplyToMessageID)}
	}
	if params.MessageThreadID > 0 {
		msg.MessageThreadID = int(params.MessageThreadID)
	}
	if params.DisableLinkPreview {
		msg.LinkPreviewOptions = &telego.LinkPreviewOptions{IsDisabled: true}
	}
	if params.DisableNotification {
		msg.DisableNotification = true
	}
	if strings.TrimSpace(params.ParseMode) != "" {
		msg.ParseMode = params.ParseMode
	}
	if markup := buildInlineKeyboard(params.Buttons); markup != nil {
		msg.ReplyMarkup = markup
	}
	sent, err := bot.SendMessage(ctx, msg)
	if err != nil {
		return Message{}, err
	}
	if sent == nil {
		return Message{}, nil
	}
	return *sent, nil
}

func (api *Client) SendPhoto(ctx context.Context, params SendPhotoParams) (Message, error) {
	bot, err := api.botOrError()
	if err != nil {
		return Message{}, err
	}
	input, cleanup, err := resolveTelegramInputFile(params.Photo, params.PhotoData, params.Filename)
	if err != nil {
		return Message{}, err
	}
	if cleanup != nil {
		defer cleanup()
	}
	request := &telego.SendPhotoParams{
		ChatID: resolveTelegramChatID(params.ChatID, params.Chat),
		Photo:  input,
	}
	if params.Caption != "" {
		request.Caption = params.Caption
	}
	if params.ReplyToMessageID > 0 {
		request.ReplyParameters = &telego.ReplyParameters{MessageID: int(params.ReplyToMessageID)}
	}
	if params.MessageThreadID > 0 {
		request.MessageThreadID = int(params.MessageThreadID)
	}
	if params.DisableNotification {
		request.DisableNotification = true
	}
	if markup := buildInlineKeyboard(params.Buttons); markup != nil {
		request.ReplyMarkup = markup
	}
	sent, err := bot.SendPhoto(ctx, request)
	if err != nil {
		return Message{}, err
	}
	if sent == nil {
		return Message{}, nil
	}
	return *sent, nil
}

func (api *Client) SendDocument(ctx context.Context, params SendDocumentParams) (Message, error) {
	bot, err := api.botOrError()
	if err != nil {
		return Message{}, err
	}
	input, cleanup, err := resolveTelegramInputFile(params.Document, params.DocumentData, params.Filename)
	if err != nil {
		return Message{}, err
	}
	if cleanup != nil {
		defer cleanup()
	}
	request := &telego.SendDocumentParams{
		ChatID:   resolveTelegramChatID(params.ChatID, params.Chat),
		Document: input,
	}
	if params.Caption != "" {
		request.Caption = params.Caption
	}
	if params.ReplyToMessageID > 0 {
		request.ReplyParameters = &telego.ReplyParameters{MessageID: int(params.ReplyToMessageID)}
	}
	if params.MessageThreadID > 0 {
		request.MessageThreadID = int(params.MessageThreadID)
	}
	if params.DisableNotification {
		request.DisableNotification = true
	}
	if markup := buildInlineKeyboard(params.Buttons); markup != nil {
		request.ReplyMarkup = markup
	}
	sent, err := bot.SendDocument(ctx, request)
	if err != nil {
		return Message{}, err
	}
	if sent == nil {
		return Message{}, nil
	}
	return *sent, nil
}

func (api *Client) SendPoll(ctx context.Context, params SendPollParams) (Message, error) {
	bot, err := api.botOrError()
	if err != nil {
		return Message{}, err
	}
	options := make([]telego.InputPollOption, 0, len(params.Options))
	for _, option := range params.Options {
		trimmed := strings.TrimSpace(option)
		if trimmed == "" {
			continue
		}
		options = append(options, telego.InputPollOption{Text: trimmed})
	}
	request := &telego.SendPollParams{
		ChatID:   resolveTelegramChatID(params.ChatID, params.Chat),
		Question: params.Question,
		Options:  options,
	}
	if params.IsAnonymous != nil {
		request.IsAnonymous = params.IsAnonymous
	}
	if params.ReplyToMessageID > 0 {
		request.ReplyParameters = &telego.ReplyParameters{MessageID: int(params.ReplyToMessageID)}
	}
	if params.MessageThreadID > 0 {
		request.MessageThreadID = int(params.MessageThreadID)
	}
	if params.DisableNotification {
		request.DisableNotification = true
	}
	if params.OpenPeriodSeconds > 0 {
		request.OpenPeriod = params.OpenPeriodSeconds
	}
	if params.AllowMultiple {
		request.AllowsMultipleAnswers = true
	}
	sent, err := bot.SendPoll(ctx, request)
	if err != nil {
		return Message{}, err
	}
	if sent == nil {
		return Message{}, nil
	}
	return *sent, nil
}

func (api *Client) SetWebhook(ctx context.Context, params SetWebhookParams) error {
	bot, err := api.botOrError()
	if err != nil {
		return err
	}
	request := &telego.SetWebhookParams{
		URL: params.URL,
	}
	if params.SecretToken != "" {
		request.SecretToken = params.SecretToken
	}
	if len(params.AllowedUpdates) > 0 {
		request.AllowedUpdates = params.AllowedUpdates
	}
	if params.DropPendingUpdate {
		request.DropPendingUpdates = true
	}
	return bot.SetWebhook(ctx, request)
}

func (api *Client) SetMessageReaction(ctx context.Context, params SetMessageReactionParams) error {
	bot, err := api.botOrError()
	if err != nil {
		return err
	}
	request := &telego.SetMessageReactionParams{
		ChatID:    resolveTelegramChatID(params.ChatID, params.Chat),
		MessageID: int(params.MessageID),
	}
	if !params.Remove {
		emoji := strings.TrimSpace(params.Emoji)
		if emoji != "" {
			request.Reaction = []telego.ReactionType{
				&telego.ReactionTypeEmoji{
					Type:  telego.ReactionEmoji,
					Emoji: emoji,
				},
			}
		}
	}
	return bot.SetMessageReaction(ctx, request)
}

func (api *Client) DeleteMessage(ctx context.Context, params DeleteMessageParams) error {
	bot, err := api.botOrError()
	if err != nil {
		return err
	}
	return bot.DeleteMessage(ctx, &telego.DeleteMessageParams{
		ChatID:    resolveTelegramChatID(params.ChatID, params.Chat),
		MessageID: int(params.MessageID),
	})
}

func (api *Client) EditMessage(ctx context.Context, params EditMessageParams) (Message, error) {
	bot, err := api.botOrError()
	if err != nil {
		return Message{}, err
	}
	request := &telego.EditMessageTextParams{
		ChatID:    resolveTelegramChatID(params.ChatID, params.Chat),
		MessageID: int(params.MessageID),
		Text:      params.Text,
	}
	if strings.TrimSpace(params.ParseMode) != "" {
		request.ParseMode = params.ParseMode
	}
	if markup := buildInlineKeyboard(params.Buttons); markup != nil {
		request.ReplyMarkup = markup
	}
	edited, err := bot.EditMessageText(ctx, request)
	if err != nil {
		return Message{}, err
	}
	if edited == nil {
		return Message{}, nil
	}
	return *edited, nil
}

func buildInlineKeyboard(rows [][]InlineButton) *telego.InlineKeyboardMarkup {
	if len(rows) == 0 {
		return nil
	}
	keyboard := make([][]telego.InlineKeyboardButton, 0, len(rows))
	for _, row := range rows {
		if len(row) == 0 {
			continue
		}
		buttonRow := make([]telego.InlineKeyboardButton, 0, len(row))
		for _, button := range row {
			text := strings.TrimSpace(button.Text)
			callback := strings.TrimSpace(button.CallbackData)
			if text == "" || callback == "" {
				continue
			}
			item := telego.InlineKeyboardButton{
				Text:         text,
				CallbackData: callback,
			}
			style := strings.ToLower(strings.TrimSpace(button.Style))
			if style == "danger" || style == "success" || style == "primary" {
				item.Style = style
			}
			buttonRow = append(buttonRow, item)
		}
		if len(buttonRow) > 0 {
			keyboard = append(keyboard, buttonRow)
		}
	}
	if len(keyboard) == 0 {
		return nil
	}
	return &telego.InlineKeyboardMarkup{InlineKeyboard: keyboard}
}

func resolveTelegramChatID(numericID int64, raw string) telego.ChatID {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return tu.ID(numericID)
	}
	if parsed, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return tu.ID(parsed)
	}
	return tu.Username(trimmed)
}

type namedBytesReader struct {
	*bytes.Reader
	name string
}

func (reader *namedBytesReader) Name() string {
	if reader == nil {
		return ""
	}
	return reader.name
}

func resolveTelegramInputFile(rawSource string, payload []byte, filename string) (telego.InputFile, func(), error) {
	if len(payload) > 0 {
		resolvedFilename := strings.TrimSpace(filename)
		if resolvedFilename == "" {
			resolvedFilename = "attachment"
		}
		return telego.InputFile{
			File: &namedBytesReader{
				Reader: bytes.NewReader(payload),
				name:   resolvedFilename,
			},
		}, nil, nil
	}
	source := strings.TrimSpace(rawSource)
	if source == "" {
		return telego.InputFile{}, nil, fmt.Errorf("telegram media source is required")
	}
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return telego.InputFile{URL: source}, nil, nil
	}
	if strings.HasPrefix(source, "file://") {
		parsed, err := url.Parse(source)
		if err == nil {
			if path := strings.TrimSpace(parsed.Path); path != "" {
				source = path
			}
		}
	}
	if stat, err := os.Stat(source); err == nil && !stat.IsDir() {
		file, openErr := os.Open(source)
		if openErr != nil {
			return telego.InputFile{}, nil, openErr
		}
		return telego.InputFile{File: file}, func() {
			_ = file.Close()
		}, nil
	}
	return telego.InputFile{FileID: source}, nil, nil
}

func (api *Client) DeleteWebhook(ctx context.Context, dropPending bool) error {
	bot, err := api.botOrError()
	if err != nil {
		return err
	}
	request := &telego.DeleteWebhookParams{}
	if dropPending {
		request.DropPendingUpdates = true
	}
	return bot.DeleteWebhook(ctx, request)
}

func (api *Client) GetFile(ctx context.Context, fileID string) (*File, error) {
	bot, err := api.botOrError()
	if err != nil {
		return nil, err
	}
	return bot.GetFile(ctx, &telego.GetFileParams{FileID: fileID})
}

func (api *Client) FileDownloadURL(filePath string) (string, error) {
	bot, err := api.botOrError()
	if err != nil {
		return "", err
	}
	return bot.FileDownloadURL(filePath), nil
}
