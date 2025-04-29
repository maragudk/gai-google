package google_test

import (
	"testing"

	"maragu.dev/env"
	"maragu.dev/is"

	google "maragu.dev/gai-google"
)

func TestNewClient(t *testing.T) {
	t.Run("can create a new client with a key", func(t *testing.T) {
		client := google.NewClient(google.NewClientOptions{Key: "123"})
		is.NotNil(t, client)
	})
}

func newClient() *google.Client {
	_ = env.Load(".env.test.local")

	return google.NewClient(google.NewClientOptions{Key: env.GetStringOrDefault("GOOGLE_KEY", "")})
}
