package google_test

import (
	"testing"

	"maragu.dev/gai"
	"maragu.dev/is"

	google "maragu.dev/gai-google"
)

func TestChatCompleter_ChatComplete(t *testing.T) {
	t.Run("can chat-complete", func(t *testing.T) {
		c := newClient()

		cc := c.NewChatCompleter(google.NewChatCompleterOptions{
			Model: google.ChatCompleteModelGemini2_0Flash,
		})

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
			output += part.Text()
		}
		is.Equal(t, "Hi there! How can I help you today?\n", output)
	})
}
