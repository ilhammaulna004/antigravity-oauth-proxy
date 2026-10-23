package antigravity

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type streamHTTPClientFunc func(*http.Request) (*http.Response, error)

func (f streamHTTPClientFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }

type closeSignalBody struct {
	io.Reader
	closed chan struct{}
}

func (b *closeSignalBody) Close() error {
	close(b.closed)
	return nil
}

func TestStreamGenerateContentClosesBodyWhenConsumerStops(t *testing.T) {
	body := &closeSignalBody{Reader: strings.NewReader("data: hello\n"), closed: make(chan struct{})}
	client := &Client{
		provider: stubCredentialsProvider{},
		httpClient: streamHTTPClientFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
		}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := make(chan string) // No consumer: sender must stop when context is canceled.
	if err := client.StreamGenerateContent(ctx, &GenerateContentRequest{Model: "gemini-3.1-pro-low"}, out); err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case <-body.closed:
	case <-time.After(time.Second):
		t.Fatal("stream body not closed after cancellation")
	}
	select {
	case _, ok := <-out:
		if ok {
			t.Fatal("stream channel still open")
		}
	default:
		t.Fatal("stream channel not closed")
	}
}
