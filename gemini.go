package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"

	"google.golang.org/genai"
)

// callGeminiAPI は Gemini API に共通のプロンプトで問い合わせ、テキストレスポンスを返す
// ctx: コンテキスト
// prompt: Gemini に送信するプロンプト
// model: 使用する Gemini モデル（例：gemini-2.0-flash）
// geminiKey: Gemini API キー
func callGeminiAPI(ctx context.Context, prompt string, model string, geminiKey string) (string, error) {
	if geminiKey == "" {
		return "", fmt.Errorf("gemini API キーが設定されていません")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: geminiKey})
	if err != nil {
		log.Printf("Gemini クライアント作成エラー: %v", err)
		return "", fmt.Errorf("gemini クライアントの作成に失敗しました")
	}

	// Gemini API を呼び出し
	result, err := client.Models.GenerateContent(ctx, model, genai.Text(prompt), nil)
	if err != nil {
		log.Printf("Gemini API 呼び出しエラー: %v", err)
		return "", fmt.Errorf("gemini API の呼び出しに失敗しました")
	}

	// レスポンスからテキストを抽出

	responseText := result.Text()

	if responseText == "" {
		return "", fmt.Errorf("gemini レスポンスにテキストコンテンツがありません")
	}

	return responseText, nil
}

// MyLibraryRecord はマイライブラリの書籍情報を表す構造体
type MyLibraryRecord struct {
	ID            string `json:"id"`
	BookID        string `json:"bookId"`
	Title         string `json:"title"`         // 書籍タイトル
	Thumbnail     string `json:"thumbnail"`     // 表紙画像URL
	Authors       string `json:"authors"`       // 著者名（カンマ区切り）
	Pages         string `json:"pages"`         // ページ数
	Publisher     string `json:"publisher"`     // 出版社
	PublishedDate string `json:"publishedDate"` // 出版日
	Description   string `json:"description"`   // 説明
	ISBN          string `json:"isbn"`          // ISBN
	Genre         string `json:"genre"`         // ジャンル
	Status        string `json:"status"`        // 読書状態
	Rating        int    `json:"rating"`        // 評価
	Review        string `json:"review"`        // レビュー
}

// LibraryRequest はクライアントから送信されるマイライブラリの形式
type LibraryRequest struct {
	Success bool              `json:"success"`
	Records []MyLibraryRecord `json:"records"`
}

// RecommendedBook はおすすめ書籍の情報を表す構造体
type RecommendedBook struct {
	Title  string `json:"title"`  // 書籍タイトル
	Author string `json:"author"` // 著者名
}

// GetRecommendedBooks は Gemini API を使用してユーザーのマイライブラリに基づいて、
// 3つのおすすめ書籍を取得する
func GetRecommendedBooks(ctx context.Context, library LibraryRequest, geminiKey string) ([]RecommendedBook, error) {
	// ユーザーのマイライブラリ情報をプロンプトに組み込む
	libraryInfo := "ユーザーのマイライブラリ:\n"
	for _, book := range library.Records {
		// 読了状況や評価を考慮した情報を追加
		status := book.Status
		ratingStr := ""
		if book.Rating > 0 {
			ratingStr = fmt.Sprintf(" (評価: %d/5)", book.Rating)
		}
		libraryInfo += fmt.Sprintf("- 『%s』 著者: %s (ジャンル: %s, 状態: %s%s)\n",
			book.Title, book.Authors, book.Genre, status, ratingStr)
	}

	// Gemini API に送信するプロンプト
	prompt := libraryInfo + `
上記のマイライブラリに基づいて、ユーザーに合いそうなおすすめ書籍を「正確に5冊」提案してください。
レスポンスは以下の JSON 形式でのみ返してください。他のテキストは一切含めないでください。

[
  {"title": "書籍タイトル", "author": "著者名"},
  {"title": "書籍タイトル", "author": "著者名"},
  {"title": "書籍タイトル", "author": "著者名"},
  {"title": "書籍タイトル", "author": "著者名"},
  {"title": "書籍タイトル", "author": "著者名"}
]

注意:
1. JSON配列のみを返す
2. 正確に3冊のみを返す
3. マイライブラリに既に存在する書籍は避ける
4. 実在する書籍と著者を提案する`

	// 共通の Gemini API 呼び出し関数を使用
	responseText, err := callGeminiAPI(ctx, prompt, "gemini-2.5-flash", geminiKey)
	if err != nil {
		return nil, err
	}

	// レスポンスからJSONを抽出（マークダウンのコードブロックを除去）
	jsonText := extractJSON(responseText)

	// JSON レスポンスをパース
	var recommendations []RecommendedBook
	if err := json.Unmarshal([]byte(jsonText), &recommendations); err != nil {
		log.Printf("gemini レスポンスのパースエラー: %v, レスポンス内容: %s", err, responseText)
		return nil, fmt.Errorf("gemini レスポンスのパースに失敗しました")
	}

	return recommendations, nil
}

// extractJSON はレスポンステキストからJSON部分を抽出する
// マークダウンのコードブロック（```json ... ```）やその他の装飾を除去する
func extractJSON(text string) string {
	// マークダウンコードブロックを除去
	re := regexp.MustCompile("(?s)```(?:json)?\\s*([\\s\\S]*?)```")
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// コードブロックがない場合は、JSON配列またはオブジェクトを探す
	// [ または { で始まる部分を抽出
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "[") || strings.HasPrefix(text, "{") {
		return text
	}

	// それでも見つからない場合は元のテキストを返す
	return text
}

// SummaryResponse は読書傾向要約のレスポンス構造体
type SummaryResponse struct {
	Summary string `json:"summary"` // 読書傾向の要約
}

// GetReadingSummary は Gemini API を使用してユーザーのマイライブラリから
// 読書傾向の要約と行動提案を生成する
func GetReadingSummary(ctx context.Context, library LibraryRequest, geminiKey string) (string, error) {
	// ユーザーのマイライブラリ情報を詳細に構築
	libraryInfo := "ユーザーのマイライブラリ:\n\n"

	// 読了書籍と未読書籍を分類
	var readBooks []MyLibraryRecord
	var unreadBooks []MyLibraryRecord

	for _, book := range library.Records {
		if book.Status == "読了" {
			readBooks = append(readBooks, book)
		} else {
			unreadBooks = append(unreadBooks, book)
		}
	}

	// 読了書籍の情報を追加
	if len(readBooks) > 0 {
		libraryInfo += "【読了書籍】\n"
		for _, book := range readBooks {
			ratingStr := ""
			if book.Rating > 0 {
				ratingStr = fmt.Sprintf(" | 評価: %d/5", book.Rating)
			}
			reviewStr := ""
			if book.Review != "" {
				reviewStr = fmt.Sprintf(" | レビュー: %s", book.Review)
			}
			libraryInfo += fmt.Sprintf("- 『%s』著者: %s (ジャンル: %s%s%s)\n",
				book.Title, book.Authors, book.Genre, ratingStr, reviewStr)
		}
		libraryInfo += "\n"
	}

	// 未読書籍の情報を追加
	if len(unreadBooks) > 0 {
		libraryInfo += "【未読書籍】\n"
		for _, book := range unreadBooks {
			libraryInfo += fmt.Sprintf("- 『%s』著者: %s (ジャンル: %s)\n",
				book.Title, book.Authors, book.Genre)
		}
		libraryInfo += "\n"
	}

	// Gemini API に送信するプロンプト
	prompt := libraryInfo + `
上記のマイライブラリに基づいて、以下の内容を日本語で分析して簡潔にまとめてください。
各番号の各項目に対して、1文程度の箇条書きで簡潔に回答してください。
返すテキストはそのまま表示・コピペできるようにしてください。段落は改行で区切ってください。

1. 【読書傾向の分析】
   - ユーザーの主な読書ジャンル
   - 好みの著者や作風の特徴
   - 読書スタイル（多読傾向、じっくり読み傾向など）

2. 【読書の強み】
   - 現在の読書習慣で良い点

3. 【行動提案】
   - 読書習慣の改善提案
   - 新しいジャンルや著者の提案
   - 効率的な読書方法の提案

返すときは、上記の構成を維持したテキストで、数字や記号は最小限にしてください。`

	// 共通の Gemini API 呼び出し関数を使用
	responseText, err := callGeminiAPI(ctx, prompt, "gemini-2.5-flash", geminiKey)
	if err != nil {
		return "", err
	}

	return responseText, nil
}
