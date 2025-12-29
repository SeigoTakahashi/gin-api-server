package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// loadAPIKeyFromEnv は環境変数またはワーキングディレクトリの .env ファイルから
// 指定されたキーの値を取得する。
// keyName: 環境変数名またはプレフィックス（例：GOOGLE_BOOKS_API_KEY, GEMINI_API_KEY）
// 取得できない場合は空文字列を返す。
func loadAPIKeyFromEnv(keyName string) string {
	if k := os.Getenv(keyName); k != "" {
		return k
	}

	data, err := os.ReadFile(".env")
	if err != nil {
		return ""
	}

	prefix := keyName + "="
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			return strings.TrimPrefix(line, prefix)
		}
	}
	return ""
}

// Initialize は .env ファイルが存在する場合、そこから環境変数を読み込む。
// ローカル開発では .env を使用し、本番環境（Render など）では
// 実際の環境変数を使用するため、.env が見つからない場合はエラーにしない。
func Initialize() {
	if err := godotenv.Load(); err != nil {
		log.Println(".env ファイルが見つかりません。環境変数を使用します")
	}
}

// アプリケーション設定定数
const (
	// デフォルトのmaxResults値（Google Books API）
	DefaultMaxResults = "20"
	// HTTPクライアントのタイムアウト
	DefaultHTTPTimeout = 10 * time.Second
	// CORS MaxAge設定
	CORSMaxAge = 12 * time.Hour
	// ローカル開発用のデフォルトポート
	DefaultPort = "8080"
)

// AppConfig はアプリケーションの設定を管理する構造体
type AppConfig struct {
	GoogleBooksAPIKey string
	GeminiAPIKey      string
	HTTPClient        *http.Client
}

// NewAppConfig はアプリケーション起動時に全ての API キーを読み込んで設定を初期化する
func NewAppConfig() (*AppConfig, error) {
	googleBooksAPIKey := loadAPIKeyFromEnv("GOOGLE_BOOKS_API_KEY")
	if googleBooksAPIKey == "" {
		return nil, fmt.Errorf("google Books API キーが設定されていません")
	}

	geminiAPIKey := loadAPIKeyFromEnv("GEMINI_API_KEY")
	if geminiAPIKey == "" {
		return nil, fmt.Errorf("gemini API キーが設定されていません")
	}

	log.Println("API キーの読み込みに成功しました")
	return &AppConfig{
		GoogleBooksAPIKey: googleBooksAPIKey,
		HTTPClient:        &http.Client{Timeout: DefaultHTTPTimeout},
		GeminiAPIKey:      geminiAPIKey,
	}, nil
}

func main() {
	// .env ファイルの読み込みを試行（エラーは許容）
	// ローカル開発は .env を、本番環境（Render）は実際の環境変数を使用
	Initialize()

	// アプリケーション設定を初期化
	cfg, err := NewAppConfig()
	if err != nil {
		log.Fatalf("API キーの初期化に失敗しました: %v", err)
	}

	// デフォルトルータを初期化
	router := gin.Default()

	// CORS設定を追加
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "https://reading-support.web.app"}, // 許可するオリジン
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           CORSMaxAge,
	}))

	// /books エンドポイント: searchQuery と maxResults をクエリパラメータで受け取り、
	// Google Books API を叩いて得たレスポンスを返す
	router.GET("/books", func(c *gin.Context) {
		// クエリパラメータから検索キーワードと取得数を取得
		searchQuery := c.Query("searchQuery")
		maxResults := c.Query("maxResults")

		if searchQuery == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "searchQuery パラメータが指定されていません"})
			return
		}

		if maxResults == "" {
			maxResults = DefaultMaxResults
		}

		// Google Books API で書籍を検索
		booksResp, err := FetchBooksByQuery(searchQuery, maxResults, cfg.GoogleBooksAPIKey, cfg.HTTPClient)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}

		// 取得した書籍情報を JSON で返す
		c.JSON(http.StatusOK, booksResp)
	})

	// /recommends エンドポイント: ユーザーのマイライブラリをもとに
	// Gemini API でおすすめ書籍 3 冊を取得し、Google Books API で詳細情報を返す
	router.POST("/recommends", func(c *gin.Context) {
		// リクエストボディからマイライブラリを取得
		var req LibraryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "リクエストボディが不正です"})
			return
		}

		if len(req.Records) == 0 {
			c.JSON(http.StatusOK, gin.H{"message": "マイライブラリに書籍が登録されていません。書籍を追加してください。", "items": []BookInfo{}})
			return
		}

		// Gemini API を使用して、ユーザーの読書傾向に基づくおすすめ書籍を取得
		ctx := context.Background()

		recommendations, err := GetRecommendedBooks(ctx, req, cfg.GeminiAPIKey)
		if err != nil {
			log.Printf("おすすめ書籍取得エラー: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 各おすすめ書籍について Google Books API から詳細情報を取得
		var results []BookInfo
		for _, rec := range recommendations {
			// タイトルと著者で検索クエリを構築
			// 形式: intitle:タイトル+inauthor:著者名
			searchQuery := "intitle:" + rec.Title + "+inauthor:" + rec.Author
			booksResp, err := FetchBooksByQuery(searchQuery, "1", cfg.GoogleBooksAPIKey, cfg.HTTPClient)
			if err != nil {
				log.Printf("書籍情報取得エラー『%s』: %v", rec.Title, err)
				continue
			}
			if booksResp != nil && len(booksResp.Items) > 0 {
				volumeInfo := booksResp.Items[0].VolumeInfo
				// 著者情報を抽出
				authors := ""
				if len(volumeInfo.Authors) > 0 {
					authors = volumeInfo.Authors[0]
				}
				// ISBN を抽出
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
		}

		// 取得したおすすめ書籍の詳細情報を JSON で返す
		c.JSON(http.StatusOK, gin.H{"items": results})
	})

	// /summary エンドポイント: ユーザーのマイライブラリから読書傾向を分析し、
	// 要約と行動提案をテキストとして返す
	router.POST("/summary", func(c *gin.Context) {
		// リクエストボディからマイライブラリを取得
		var req LibraryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "リクエストボディが不正です"})
			return
		}

		if len(req.Records) == 0 {
			c.JSON(http.StatusOK, gin.H{"message": "マイライブラリに書籍が登録されていません。書籍を追加してください。", "summary": ""})
			return
		}

		// Gemini API を使用して読書傾向を分析
		ctx := context.Background()
		summary, err := GetReadingSummary(ctx, req, cfg.GeminiAPIKey)
		if err != nil {
			log.Printf("読書傾向要約生成エラー: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// テキスト形式でそのままコピペできるように返す
		c.JSON(http.StatusOK, gin.H{"summary": summary})
	})

	// サーバーの起動
	// PORT 環境変数が設定されていればそれを使用、なければデフォルトポートで起動
	port := os.Getenv("PORT")
	if port == "" {
		port = DefaultPort
	}
	router.Run(":" + port)
}
