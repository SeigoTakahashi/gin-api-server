package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestFetchBooksByQueryRejectsNonSuccessHTTPStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"API key invalid"}}`, http.StatusForbidden)
	}))
	defer server.Close()

	client := &http.Client{Transport: booksURLTransport(server.URL)}
	_, err := FetchBooksByQuery("test", "1", "test-key", client)
	if err == nil {
		t.Fatal("FetchBooksByQuery() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("FetchBooksByQuery() error = %q, want HTTP status", err)
	}
}

func TestFetchBooksByQueryRetriesServiceUnavailable(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte(`{"items":[]}`))
	}))
	defer server.Close()

	client := &http.Client{Transport: booksURLTransport(server.URL)}
	books, err := FetchBooksByQuery("test", "1", "test-key", client)
	if err != nil {
		t.Fatalf("FetchBooksByQuery() error = %v", err)
	}
	if books == nil {
		t.Fatal("FetchBooksByQuery() = nil, want response")
	}
	if got := requests.Load(); got != 2 {
		t.Errorf("request count = %d, want 2", got)
	}
}

func TestFetchBooksByQueryStopsAfterRetryLimit(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := &http.Client{Transport: booksURLTransport(server.URL)}
	_, err := FetchBooksByQuery("test", "1", "test-key", client)
	if err == nil || !strings.Contains(err.Error(), "HTTP 503") {
		t.Fatalf("FetchBooksByQuery() error = %v, want HTTP 503 error", err)
	}
	if got := requests.Load(); got != googleBooksMaxAttempts {
		t.Errorf("request count = %d, want %d", got, googleBooksMaxAttempts)
	}
}

type booksURLTransport string

func (transport booksURLTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(string(transport), "http://")
	return http.DefaultTransport.RoundTrip(req)
}
