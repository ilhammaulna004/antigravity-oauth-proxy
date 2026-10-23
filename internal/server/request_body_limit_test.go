package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadLimitedRequestBody(t *testing.T) {
	for _, tt := range []struct {
		name   string
		body   string
		status int
	}{
		{name: "at limit", body: "1234", status: http.StatusOK},
		{name: "over limit", body: "12345", status: http.StatusRequestEntityTooLarge},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			body, ok := readLimitedRequestBody(w, r, 4)
			if ok != (tt.status == http.StatusOK) || w.Code != tt.status {
				t.Fatalf("ok = %v, status = %d; want status %d", ok, w.Code, tt.status)
			}
			if ok && string(body) != tt.body {
				t.Fatalf("body = %q, want %q", body, tt.body)
			}
		})
	}
}

func TestGenerationHandlersRejectOversizedBody(t *testing.T) {
	for _, tt := range []struct {
		name    string
		handler http.HandlerFunc
	}{
		{name: "OpenAI", handler: (&Server{}).openAIChatCompletionsHandler},
		{name: "Gemini", handler: func(w http.ResponseWriter, r *http.Request) { (&Server{}).handleGenerateContent(w, r, "gemini-2.5-flash") }},
		{name: "Gemini stream", handler: func(w http.ResponseWriter, r *http.Request) { (&Server{}).handleStreamGenerateContent(w, r, "gemini-2.5-flash") }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", maxGenerationRequestBody+1)))
			tt.handler(w, r)
			if w.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf("status = %d, want %d", w.Code, http.StatusRequestEntityTooLarge)
			}
		})
	}
}
