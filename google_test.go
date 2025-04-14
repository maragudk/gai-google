package google_test

import (
	"testing"

	google "maragu.dev/gai-google"
	"maragu.dev/is"
)

func TestNewClient(t *testing.T) {
	t.Run("can create a new client with a key", func(t *testing.T) {
		client := google.NewClient(google.NewClientOptions{Key: "123"})
		is.NotNil(t, client)
	})
}
