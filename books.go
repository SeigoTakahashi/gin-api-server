package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

const googleBooksMaxAttempts = 3

// BookInfo はクライアントに返却される書籍情報を表す構造体
type BookInfo struct {
	Title         string `json:"title"`         // 書籍タイトル
	Author        string `json:"author"`        // 著者名
	Thumbnail     string `json:"thumbnail"`     // 表紙画像URL
	Publisher     string `json:"publisher"`     // 出版社
	PublishedDate string `json:"publishedDate"` // 出版日
	Description   string `json:"description"`   // 説明
	Pages         string `json:"pages"`         // ページ数
	ISBN          string `json:"isbn"`          // ISBN
}

// GoogleBooksResponse は Google Books API のレスポンス構造を表す
type GoogleBooksResponse struct {
	Items []struct {
		ID         string `json:"id"`
		VolumeInfo struct {
			Title         string   `json:"title"`
			Authors       []string `json:"authors"`
			Publisher     string   `json:"publisher"`
			PublishedDate string   `json:"publishedDate"`
			Description   string   `json:"description"`
			PageCount     int      `json:"pageCount"`
			ImageLinks    struct {
				Thumbnail string `json:"thumbnail"`
			} `json:"imageLinks"`
			IndustryIdentifiers []struct {
				Type       string `json:"type"`
				Identifier string `json:"identifier"`
			} `json:"industryIdentifiers"`
		} `json:"volumeInfo"`
	} `json:"items"`
}

// FetchBooksByQuery は Google Books API で書籍を検索し、API レスポンスをそのまま返す
// searchQuery: 検索クエリ（例: "intitle:こころ+inauthor:夏目漱石"）
// maxResults: 取得する書籍数の上限
// apiKey: Google Books API キー
// httpClient: 共有の http.Client インスタンス
func FetchBooksByQuery(searchQuery string, maxResults string, apiKey string, httpClient *http.Client) (*GoogleBooksResponse, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("APIキーが設定されていません")
	}

	baseURL, err := url.Parse("https://www.googleapis.com/books/v1/volumes")
	if err != nil {
		return nil, fmt.Errorf("URLのパースに失敗しました")
	}

	params := url.Values{}
	params.Add("q", searchQuery)
	params.Add("maxResults", maxResults)
	params.Add("key", apiKey)

	baseURL.RawQuery = params.Encode()
	target := baseURL.String()

	// HTTP リクエストを作成
	req, err := http.NewRequest("GET", target, nil)
	if err != nil {
		log.Printf("リクエスト作成エラー: %v", err)
		return nil, fmt.Errorf("リクエスト作成に失敗しました")
	}

	// Google Books API の一時的な 503 に備え、短い間隔で最大 3 回試行する。
	var resp *http.Response
	for attempt := 1; attempt <= googleBooksMaxAttempts; attempt++ {
		resp, err = httpClient.Do(req)
		if err != nil {
			log.Printf("API呼び出しエラー: %v", err)
			return nil, fmt.Errorf("外部APIの呼び出しに失敗しました")
		}
		if resp.StatusCode != http.StatusServiceUnavailable || attempt == googleBooksMaxAttempts {
			break
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		time.Sleep(time.Duration(attempt) * 250 * time.Millisecond)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		log.Printf("Google Books API が HTTP %d を返しました", resp.StatusCode)
		if resp.StatusCode == http.StatusServiceUnavailable {
			return nil, fmt.Errorf("Google Books API が一時的に利用できません (HTTP %d)。時間をおいて再試行してください", resp.StatusCode)
		}
		return nil, fmt.Errorf("Google Books API が HTTP %d を返しました", resp.StatusCode)
	}

	// レスポンスボディを読み込む
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("レスポンス読み込みエラー: %v", err)
		return nil, fmt.Errorf("レスポンスの読み込みに失敗しました")
	}

	log.Printf("Google Books API Raw Response: %s", string(body))

	// JSON レスポンスをパース
	var booksResp GoogleBooksResponse
	if err := json.Unmarshal(body, &booksResp); err != nil {
		log.Printf("レスポンスパースエラー: %v", err)
		return nil, fmt.Errorf("レスポンスのパースに失敗しました")
	}

	return &booksResp, nil
}
