package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUploadToImgBB(t *testing.T) {
	t.Run("returns image URL from successful response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %q, want POST", r.Method)
			}
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Errorf("ParseMultipartForm() error = %v", err)
			}
			file, header, err := r.FormFile("image")
			if err != nil {
				t.Errorf("FormFile() error = %v", err)
			} else {
				defer file.Close()
				content, err := io.ReadAll(file)
				if err != nil {
					t.Errorf("ReadAll() error = %v", err)
				}
				if string(content) != "image data" || header.Filename != "cover.jpg" {
					t.Errorf("uploaded file = (%q, %q), want (%q, %q)", content, header.Filename, "image data", "cover.jpg")
				}
			}
			w.Write([]byte(`{"success":true,"data":{"url":"https://images.example/cover.jpg"}}`))
		}))
		defer server.Close()

		client := &http.Client{}
		client.Transport = rewriteImgBBURL(server.URL)
		got, err := uploadToImgBB(strings.NewReader("image data"), "cover.jpg", "test-key", client)
		if err != nil {
			t.Fatalf("uploadToImgBB() error = %v", err)
		}
		if got != "https://images.example/cover.jpg" {
			t.Errorf("uploadToImgBB() = %q, want image URL", got)
		}
	})

	t.Run("rejects unsuccessful ImgBB response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"success":false,"data":{"url":""}}`))
		}))
		defer server.Close()

		client := &http.Client{Transport: rewriteImgBBURL(server.URL)}
		if _, err := uploadToImgBB(strings.NewReader("image data"), "cover.jpg", "test-key", client); err == nil {
			t.Fatal("uploadToImgBB() error = nil, want error")
		}
	})

	t.Run("rejects non-success HTTP status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "upstream failure", http.StatusBadRequest)
		}))
		defer server.Close()

		client := &http.Client{Transport: rewriteImgBBURL(server.URL)}
		_, err := uploadToImgBB(strings.NewReader("image data"), "cover.jpg", "test-key", client)
		if err == nil {
			t.Fatal("uploadToImgBB() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "upstream failure") {
			t.Errorf("uploadToImgBB() error = %q, want upstream response body", err)
		}
	})
}

type imgBBURLTransport string

func (transport imgBBURLTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(string(transport), "http://")
	return http.DefaultTransport.RoundTrip(req)
}

func rewriteImgBBURL(serverURL string) http.RoundTripper {
	return imgBBURLTransport(serverURL)
}
