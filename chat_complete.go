package google

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"google.golang.org/genai"
	"maragu.dev/gai"
	"maragu.dev/gai-google/internal/schema"
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
	if len(req.Messages) == 0 {
		panic("no messages")
	}

	if req.Messages[len(req.Messages)-1].Role != gai.MessageRoleUser {
		panic("last message must have user role")
	}

	var config genai.GenerateContentConfig
	if req.Temperature != nil {
		config.Temperature = gai.Ptr(float32(*req.Temperature))
	}
	if req.System != nil {
		config.SystemInstruction = genai.NewContentFromText(*req.System, genai.RoleUser)
	}

	if len(req.Tools) > 0 {
		tools, err := schema.ConvertTools(req.Tools)
		if err != nil {
			return gai.ChatCompleteResponse{}, fmt.Errorf("converting tools: %w", err)
		}
		config.Tools = tools
		config.ToolConfig = &genai.ToolConfig{
			FunctionCallingConfig: &genai.FunctionCallingConfig{
				Mode: genai.FunctionCallingConfigModeAuto,
			},
		}
	}

	var history []*genai.Content
	for _, m := range req.Messages {
		var content genai.Content

		switch m.Role {
		case gai.MessageRoleUser:
			content.Role = genai.RoleUser
		case gai.MessageRoleModel:
			content.Role = genai.RoleModel
		default:
			panic("unknown role " + m.Role)
		}

		for _, part := range m.Parts {
			switch part.Type {
			case gai.MessagePartTypeText:
				content.Parts = append(content.Parts, &genai.Part{Text: part.Text()})
			case gai.MessagePartTypeToolCall:
				toolCall := part.ToolCall()
				args := make(map[string]any)
				if err := json.Unmarshal(toolCall.Args, &args); err != nil {
					return gai.ChatCompleteResponse{}, fmt.Errorf("unmarshaling tool call args: %w", err)
				}
				content.Parts = append(content.Parts, &genai.Part{
					FunctionCall: &genai.FunctionCall{
						Name: toolCall.Name,
						Args: args,
					},
				})
			case gai.MessagePartTypeToolResult:
				toolResult := part.ToolResult()
				resp := map[string]any{"output": toolResult.Content}
				if toolResult.Err != nil {
					resp = map[string]any{"error": toolResult.Err.Error()}
				}
				content.Parts = append(content.Parts, &genai.Part{
					FunctionResponse: &genai.FunctionResponse{
						Name:     toolResult.ID,
						Response: resp,
					},
				})
			default:
				panic("unknown part type " + part.Type)
			}
		}

		history = append(history, &content)
	}

	// Delete the last content from the history, because SendMessageStream expects it as varargs
	lastContent := history[len(history)-1]
	history = history[:len(history)-1]

	chat, err := c.Client.Chats.Create(ctx, string(c.model), &config, history)
	if err != nil {
		return gai.ChatCompleteResponse{}, err
	}

	return gai.NewChatCompleteResponse(func(yield func(gai.MessagePart, error) bool) {
		for chunk, err := range chat.SendStream(ctx, lastContent.Parts...) {
			if err != nil {
				yield(gai.MessagePart{}, err)
				return
			}

			if len(chunk.Candidates) > 0 && chunk.Candidates[0].Content != nil {
				for _, part := range chunk.Candidates[0].Content.Parts {
					if part.Text != "" {
						if !yield(gai.TextMessagePart(part.Text), nil) {
							return
						}
					}
					if part.FunctionCall != nil {
						args, err := json.Marshal(part.FunctionCall.Args)
						if err != nil {
							yield(gai.MessagePart{}, fmt.Errorf("marshaling function args: %w", err))
							return
						}
						id := part.FunctionCall.ID
						if id == "" {
							id = part.FunctionCall.Name
						}
						if !yield(gai.ToolCallPart(id, part.FunctionCall.Name, args), nil) {
							return
						}
					}
				}
			}
		}
	}), nil
}

var _ gai.ChatCompleter = (*ChatCompleter)(nil)
