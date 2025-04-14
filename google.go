package google

import (
	"context"
	"log/slog"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
	"maragu.dev/gai"
)

type Client struct {
	Client *genai.Client
	log    *slog.Logger
}

type NewClientOptions struct {
	Key string
	Log *slog.Logger
}

func NewClient(opts NewClientOptions) *Client {
	if opts.Log == nil {
		opts.Log = slog.New(slog.DiscardHandler)
	}

	client, err := genai.NewClient(context.Background(), option.WithAPIKey(opts.Key))
	if err != nil {
		panic(err)
	}

	return &Client{
		Client: client,
		log:    opts.Log,
	}
}

type ChatCompleteModel string

const (
	ChatCompleteModelGemini1_5Flash = ChatCompleteModel("models/gemini-1.5-flash")
	ChatCompleteModelGemini2_0Flash = ChatCompleteModel("models/gemini-2.0-flash")
	ChatCompleteModelGemini1_5Pro   = ChatCompleteModel("models/gemini-1.5-pro")
)

type ChatCompleter struct {
	Client *genai.Client
	log    *slog.Logger
	model  ChatCompleteModel
}

type NewChatCompleterOptions struct {
	Model ChatCompleteModel
}

func (c *Client) NewChatCompleter(opts NewChatCompleterOptions) *ChatCompleter {
	return &ChatCompleter{
		Client: c.Client,
		log:    c.log,
		model:  opts.Model,
	}
}

func (c *ChatCompleter) ChatComplete(ctx context.Context, p gai.ChatCompleteRequest) (gai.ChatCompleteResponse, error) {
	return gai.NewChatCompleteResponse(func(yield func(gai.MessagePart, error) bool) {
		panic("not implemented")
	}), nil
}

var _ gai.ChatCompleter = (*ChatCompleter)(nil)
