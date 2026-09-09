package utils

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestQuickAnswerPostsPlainTextQuestion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if contentType := r.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "text/plain") {
			t.Errorf("Content-Type = %q, want text/plain", contentType)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if question := string(body); question != "What is KISS?" {
			t.Errorf("question = %q", question)
		}
		_, _ = w.Write([]byte("Keep it simple."))
	}))
	defer server.Close()

	previousURL := quickAnswerURL
	quickAnswerURL = server.URL
	t.Cleanup(func() { quickAnswerURL = previousURL })

	if answer := QuickAnswer("What is KISS?"); answer != "Keep it simple." {
		t.Fatalf("answer = %q", answer)
	}
}

func TestQuickAnswerReturnsPlainTextError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("Rate limit reached. Try again later."))
	}))
	defer server.Close()

	previousURL := quickAnswerURL
	quickAnswerURL = server.URL
	t.Cleanup(func() { quickAnswerURL = previousURL })

	if answer := QuickAnswer("Question"); answer != "Rate limit reached. Try again later." {
		t.Fatalf("answer = %q", answer)
	}
}
