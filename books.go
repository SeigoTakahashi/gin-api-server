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

// FetchBooksByQuery は Google Books API で書籍を検索し、整形された書籍情報を返す
// searchQuery: 検索クエリ（例: "intitle:こころ+inauthor:夏目漱石"）
// maxResults: 取得する書籍数の上限
// apiKey: Google Books API キー
func FetchBooksByQuery(searchQuery string, maxResults string, apiKey string) ([]BookInfo, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("APIキーが設定されていません")
	}

	// 検索クエリと API キーを URL エンコード
	q := url.QueryEscape(searchQuery)
	key := url.QueryEscape(apiKey)
	// Google Books API の URL を構築
	target := fmt.Sprintf("https://www.googleapis.com/books/v1/volumes?q=%s&maxResults=%s&key=%s", q, maxResults, key)

	// HTTP クライアントを作成（タイムアウト: 10秒）
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", target, nil)
	if err != nil {
		log.Printf("リクエスト作成エラー: %v", err)
		return nil, fmt.Errorf("リクエスト作成に失敗しました")
	}

	// API にリクエストを送信
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("API呼び出しエラー: %v", err)
		return nil, fmt.Errorf("外部APIの呼び出しに失敗しました")
	}
	defer resp.Body.Close()

	// レスポンスボディを読み込む
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("レスポンス読み込みエラー: %v", err)
		return nil, fmt.Errorf("レスポンスの読み込みに失敗しました")
	}

	// JSON レスポンスをパース
	var booksResp GoogleBooksResponse
	if err := json.Unmarshal(body, &booksResp); err != nil {
		log.Printf("レスポンスパースエラー: %v", err)
		return nil, fmt.Errorf("レスポンスのパースに失敗しました")
	}

	var results []BookInfo
	for _, item := range booksResp.Items {
		volumeInfo := item.VolumeInfo

		// 著者情報を抽出（複数存在する場合は最初の1つを使用）
		authors := ""
		if len(volumeInfo.Authors) > 0 {
			authors = volumeInfo.Authors[0]
		}

		// ISBN を抽出（ISBN_13 または ISBN_10）
		isbn := ""
		for _, id := range volumeInfo.IndustryIdentifiers {
			if id.Type == "ISBN_13" || id.Type == "ISBN_10" {
				isbn = id.Identifier
				break
			}
		}

		// 書籍情報を構築
		book := BookInfo{
			Title:         volumeInfo.Title,
			Author:        authors,
			Thumbnail:     volumeInfo.ImageLinks.Thumbnail,
			Publisher:     volumeInfo.Publisher,
			PublishedDate: volumeInfo.PublishedDate,
			Description:   volumeInfo.Description,
			Pages:         fmt.Sprintf("%d", volumeInfo.PageCount),
			ISBN:          isbn,
		}
		results = append(results, book)
	}

	return results, nil
}
