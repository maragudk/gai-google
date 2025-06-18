package google_test

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"maragu.dev/gai"
	"maragu.dev/gai/tools"
	"maragu.dev/is"

	google "maragu.dev/gai-google"
)

//go:embed testdata/logo.jpg
var image []byte

//go:embed testdata/hello-there.m4a
var audio []byte

//go:embed testdata/thumbs-up.mov
var video []byte

func TestChatCompleter_ChatComplete(t *testing.T) {
	t.Run("can chat-complete", func(t *testing.T) {
		cc := newChatCompleter(t)

		req := gai.ChatCompleteRequest{
			Messages: []gai.Message{
				gai.NewUserTextMessage("Hi!"),
			},
			Temperature: gai.Ptr(gai.Temperature(0)),
		}

		res, err := cc.ChatComplete(t.Context(), req)
		is.NotError(t, err)

		var output string
		for part, err := range res.Parts() {
			is.NotError(t, err)

			switch part.Type {
			case gai.MessagePartTypeText:
				output += part.Text()

			default:
				t.Fatal("unexpected message parts")
			}
		}

		is.Equal(t, "Hi there! How can I help you today?", output)

		req.Messages = append(req.Messages, gai.NewModelTextMessage("Hi there! How can I help you today?\n"))
		req.Messages = append(req.Messages, gai.NewUserTextMessage("What does the acronym AI stand for? Be brief."))

		res, err = cc.ChatComplete(t.Context(), req)
		is.NotError(t, err)

		output = ""
		for part, err := range res.Parts() {
			is.NotError(t, err)

			switch part.Type {
			case gai.MessagePartTypeText:
				output += part.Text()

			default:
				t.Fatal("unexpected message parts")
			}
		}
		is.Equal(t, "Artificial Intelligence.", output)
	})

	t.Run("can use a tool", func(t *testing.T) {
		cc := newChatCompleter(t)

		root, err := os.OpenRoot("testdata")
		is.NotError(t, err)

		req := gai.ChatCompleteRequest{
			Messages: []gai.Message{
				gai.NewUserTextMessage("What is in the readme.txt file?"),
			},
			Temperature: gai.Ptr(gai.Temperature(0)),
			Tools: []gai.Tool{
				tools.NewReadFile(root),
			},
		}

		res, err := cc.ChatComplete(t.Context(), req)
		is.NotError(t, err)

		var output string
		var found bool
		var parts []gai.MessagePart
		var result gai.ToolResult
		for part, err := range res.Parts() {
			is.NotError(t, err)

			parts = append(parts, part)

			switch part.Type {
			case gai.MessagePartTypeToolCall:
				toolCall := part.ToolCall()
				for _, tool := range req.Tools {
					if tool.Name == toolCall.Name {
						found = true
						content, err := tool.Execute(t.Context(), toolCall.Args)
						result = gai.ToolResult{
							ID:      toolCall.ID,
							Name:    tool.Name,
							Content: content,
							Err:     err,
						}
						break
					}
				}

			case gai.MessagePartTypeText:
				output += part.Text()

			default:
				t.Fatal("unexpected message parts")
			}
		}

		is.Equal(t, "", output)
		is.True(t, found, "tool not found")
		is.Equal(t, "Hi!\n", result.Content)
		is.NotError(t, result.Err)

		req.Messages = []gai.Message{
			gai.NewUserTextMessage("What is in the readme.txt file?"),
			{Role: gai.MessageRoleModel, Parts: parts},
			gai.NewUserToolResultMessage(result),
		}

		res, err = cc.ChatComplete(t.Context(), req)
		is.NotError(t, err)

		output = ""
		for part, err := range res.Parts() {
			is.NotError(t, err)

			switch part.Type {
			case gai.MessagePartTypeText:
				output += part.Text()

			default:
				t.Fatal("unexpected message parts")
			}
		}

		is.Equal(t, `The readme.txt file contains "Hi!".`, output)
	})

	t.Run("can use a tool with no args", func(t *testing.T) {
		cc := newChatCompleter(t)

		root, err := os.OpenRoot("testdata")
		is.NotError(t, err)

		req := gai.ChatCompleteRequest{
			Messages: []gai.Message{
				gai.NewUserTextMessage("What is in the current directory?"),
			},
			Temperature: gai.Ptr(gai.Temperature(0)),
			Tools: []gai.Tool{
				tools.NewListDir(root),
			},
		}

		res, err := cc.ChatComplete(t.Context(), req)
		is.NotError(t, err)

		var output string
		var found bool
		var parts []gai.MessagePart
		var result gai.ToolResult
		for part, err := range res.Parts() {
			is.NotError(t, err)

			parts = append(parts, part)

			switch part.Type {
			case gai.MessagePartTypeToolCall:
				toolCall := part.ToolCall()
				for _, tool := range req.Tools {
					if tool.Name == toolCall.Name {
						found = true
						content, err := tool.Execute(t.Context(), toolCall.Args)
						result = gai.ToolResult{
							ID:      toolCall.ID,
							Name:    toolCall.Name,
							Content: content,
							Err:     err,
						}
						break
					}
				}

			case gai.MessagePartTypeText:
				output += part.Text()

			default:
				t.Fatal("unexpected message parts")
			}
		}

		is.Equal(t, "", output)
		is.True(t, found, "tool not found")
		is.Equal(t, `["hello-there.m4a","logo.jpg","readme.txt","thumbs-up.mov"]`, result.Content)
		is.NotError(t, result.Err)
	})

	t.Run("can use a system prompt", func(t *testing.T) {
		cc := newChatCompleter(t)

		req := gai.ChatCompleteRequest{
			Messages: []gai.Message{
				gai.NewUserTextMessage("Hi!"),
			},
			System:      gai.Ptr("You always respond in French."),
			Temperature: gai.Ptr(gai.Temperature(0)),
		}

		res, err := cc.ChatComplete(t.Context(), req)
		is.NotError(t, err)

		var output string
		for part, err := range res.Parts() {
			is.NotError(t, err)

			switch part.Type {
			case gai.MessagePartTypeText:
				output += part.Text()

			default:
				t.Fatal("unexpected message parts")
			}
		}

		is.Equal(t, "Bonjour !", output)
	})

	t.Run("can use structured output", func(t *testing.T) {
		cc := newChatCompleter(t)

		type BookRecommendation struct {
			Title  string `json:"title"`
			Author string `json:"author"`
			Year   int    `json:"year"`
		}

		req := gai.ChatCompleteRequest{
			Messages: []gai.Message{
				gai.NewUserTextMessage("Recommend a science fiction book. Include the title, author, and the year it was published."),
			},
			ResponseSchema: gai.Ptr(gai.GenerateSchema[BookRecommendation]()),
			Temperature:    gai.Ptr(gai.Temperature(0)),
		}

		res, err := cc.ChatComplete(t.Context(), req)
		is.NotError(t, err)

		var output string
		for part, err := range res.Parts() {
			is.NotError(t, err)

			switch part.Type {
			case gai.MessagePartTypeText:
				output += part.Text()

			default:
				t.Fatal("unexpected message parts")
			}
		}

		// Verify it's valid JSON with the expected structure
		var book BookRecommendation
		err = json.Unmarshal([]byte(output), &book)
		is.NotError(t, err)

		// Check that all fields are populated
		is.Equal(t, "Dune", book.Title)
		is.Equal(t, "Frank Herbert", book.Author)
		is.Equal(t, 1965, book.Year)
	})

	t.Run("can describe an image", func(t *testing.T) {
		cc := newChatCompleter(t)

		req := gai.ChatCompleteRequest{
			Messages: []gai.Message{
				gai.NewUserDataMessage("image/jpeg", bytes.NewReader(image)),
			},
			System:      gai.Ptr("Describe this image concisely."),
			Temperature: gai.Ptr(gai.Temperature(0)),
		}

		res, err := cc.ChatComplete(t.Context(), req)
		is.NotError(t, err)

		var output string
		for part, err := range res.Parts() {
			is.NotError(t, err)

			switch part.Type {
			case gai.MessagePartTypeText:
				output += part.Text()

			default:
				t.Fatal("unexpected message parts")
			}
		}

		is.Equal(t, "A cute, stylized teal or turquoise gopher-like creature with a grainy texture and thick black outlines. It features large, expressive eyes, prominent buck teeth, and its small paws clasped together, conveying a shy or contemplative expression. The image is set against a solid light pink background.", output)
	})

	t.Run("can describe audio", func(t *testing.T) {
		cc := newChatCompleter(t)

		req := gai.ChatCompleteRequest{
			Messages: []gai.Message{
				gai.NewUserDataMessage("audio/mp4", bytes.NewReader(audio)),
			},
			System:      gai.Ptr("Describe this audio concisely."),
			Temperature: gai.Ptr(gai.Temperature(0)),
		}

		res, err := cc.ChatComplete(t.Context(), req)
		is.NotError(t, err)

		var output string
		for part, err := range res.Parts() {
			is.NotError(t, err)

			switch part.Type {
			case gai.MessagePartTypeText:
				output += part.Text()

			default:
				t.Fatal("unexpected message parts")
			}
		}

		t.Log(output)
		is.True(t, strings.Contains(output, "Hello there"), "should contain greeting")
	})

	t.Run("can describe a video", func(t *testing.T) {
		cc := newChatCompleter(t)

		req := gai.ChatCompleteRequest{
			Messages: []gai.Message{
				gai.NewUserDataMessage("video/quicktime", bytes.NewReader(video)),
			},
			System:      gai.Ptr("Describe this video concisely."),
			Temperature: gai.Ptr(gai.Temperature(0)),
		}

		res, err := cc.ChatComplete(t.Context(), req)
		is.NotError(t, err)

		var output string
		for part, err := range res.Parts() {
			is.NotError(t, err)

			switch part.Type {
			case gai.MessagePartTypeText:
				output += part.Text()

			default:
				t.Fatal("unexpected message parts")
			}
		}

		t.Log(output)
		is.True(t, strings.Contains(output, "thumbs-up gesture"), "should contain thumbs-up gesture")
	})

	t.Run("tracks token usage", func(t *testing.T) {
		cc := newChatCompleter(t)

		req := gai.ChatCompleteRequest{
			Messages: []gai.Message{
				gai.NewUserTextMessage("Hi!"),
			},
			Temperature: gai.Ptr(gai.Temperature(0)),
		}

		res, err := cc.ChatComplete(t.Context(), req)
		is.NotError(t, err)

		// Consume the response to ensure token usage is populated
		var output string
		for part, err := range res.Parts() {
			is.NotError(t, err)
			if part.Type == gai.MessagePartTypeText {
				output += part.Text()
			}
		}

		// Check that we got a response
		is.True(t, len(output) > 0, "should have response text")

		// Check token usage in Meta.Usage
		is.NotNil(t, res.Meta, "should have metadata")
		is.True(t, res.Meta.Usage.PromptTokens > 0, "should have prompt tokens")
		is.True(t, res.Meta.Usage.CompletionTokens > 0, "should have completion tokens")
		is.True(t, res.Meta.Usage.ThoughtsTokens > 0, "should have thoughts tokens")
		is.True(t, res.Meta.Usage.TotalTokens > 0, "should have total tokens")

		expectedTotal := res.Meta.Usage.PromptTokens + res.Meta.Usage.ThoughtsTokens + res.Meta.Usage.CompletionTokens
		is.Equal(t, expectedTotal, res.Meta.Usage.TotalTokens)
	})
}

func newChatCompleter(t *testing.T) *google.ChatCompleter {
	c := newClient(t)
	cc := c.NewChatCompleter(google.NewChatCompleterOptions{
		Model: google.ChatCompleteModelGemini2_5Flash,
	})
	return cc
}
