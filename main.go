package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// loadAPIKey tries to get the API key from the environment first.
// If not present, it will attempt to read a local .env file in the working directory
// and parse a line starting with `GOOGLE_BOOKS_API_KEY=`. Returns empty string on failure.
func loadAPIKey() string {
	if k := os.Getenv("GOOGLE_BOOKS_API_KEY"); k != "" {
		return k
	}

	data, err := os.ReadFile(".env")
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "GOOGLE_BOOKS_API_KEY=") {
			return strings.TrimPrefix(line, "GOOGLE_BOOKS_API_KEY=")
		}
	}
	return ""
}

func main() {
	// デフォルトルータを初期化
	router := gin.Default()

	// CORS設定を追加
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "https://reading-support.web.app"}, // 許可するオリジン
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// /books エンドポイント: searchQuery と maxResults をクエリパラメータで受け取り、
	// Google Books API を叩いて得たレスポンスをそのまま返す
	router.GET("/books", func(c *gin.Context) {
		searchQuery := c.Query("searchQuery")
		maxResults := c.Query("maxResults")

		if searchQuery == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing searchQuery parameter"})
			return
		}

		if maxResults == "" {
			maxResults = "20" // デフォルト
		}

		apiKey := loadAPIKey()
		if apiKey == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "API key not configured"})
			return
		}

		// 組み立てとエンコード
		q := url.QueryEscape(searchQuery)
		key := url.QueryEscape(apiKey)
		// maxResults は整数であることが期待されるが、ここでは呼び出し側に任せる
		target := fmt.Sprintf("https://www.googleapis.com/books/v1/volumes?q=%s&maxResults=%s&key=%s", q, maxResults, key)

		client := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequest("GET", target, nil)
		if err != nil {
			log.Printf("failed to create request: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
			return
		}

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("request error: %v", err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to call external API"})
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("read body error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
			return
		}

		// 受け取ったJSONをそのまま返す
		c.Data(resp.StatusCode, "application/json; charset=utf-8", body)
	})

	// サーバーの起動
	router.Run("localhost:8080") // 0.0.0.0:8080 でサーバーを立てます。
}
