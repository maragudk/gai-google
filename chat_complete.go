package google

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"maragu.dev/gai"
)

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

func (c *ChatCompleter) ChatComplete(ctx context.Context, req gai.ChatCompleteRequest) (gai.ChatCompleteResponse, error) {
	model := c.Client.GenerativeModel(string(c.model))

	if req.Temperature != nil {
		model.SetTemperature(float32(*req.Temperature))
	}

	session := model.StartChat()

	for _, m := range req.Messages {
		var content genai.Content

		switch m.Role {
		case gai.MessageRoleUser:
			content.Role = "user"
		case gai.MessageRoleAssistant:
			content.Role = "model"
		default:
			panic("unknown role " + m.Role)
		}

		for _, part := range m.Parts {
			switch part.Type {
			case gai.MessagePartTypeText:
				content.Parts = append(content.Parts, genai.Text(part.Text()))
			default:
				panic("unknown part type " + part.Type)
			}
		}

		session.History = append(session.History, &content)
	}

	// TODO check that the last history part is role user and handle

	// Delete the last content from the history, because SendMessageStream expects it as varargs
	lastContent := session.History[len(session.History)-1]
	session.History = session.History[:len(session.History)-1]

	iter := session.SendMessageStream(ctx, lastContent.Parts...)

	return gai.NewChatCompleteResponse(func(yield func(gai.MessagePart, error) bool) {
		for {
			res, err := iter.Next()
			if err != nil {
				if errors.Is(err, iterator.Done) {
					break
				}

			}

			if err != nil {
				yield(gai.MessagePart{}, err)
				return
			}

			if len(res.Candidates) > 0 {
				for _, part := range res.Candidates[0].Content.Parts {
					if textPart, ok := part.(genai.Text); ok {
						if !yield(gai.TextMessagePart(string(textPart)), nil) {
							return
						}
					}
				}
			}
		}
	}), nil
}

var _ gai.ChatCompleter = (*ChatCompleter)(nil)
