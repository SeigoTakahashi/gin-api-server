# Gin API Server For Bookworm

読書管理システム **「Bookworm」** のバックエンドを担う API サーバーです。
Go（Gin）を使用し、書籍検索・AI レコメンド・画像アップロード機能を提供します。

---

## 🌟 主な機能

### 📚 書籍検索

* **エンドポイント**: `GET /books`
* **概要**: Google Books API を利用し、キーワードに基づいた書籍情報を取得します。

### 🤖 AI レコメンド

* **エンドポイント**: `POST /recommends`
* **概要**: ユーザーの読書履歴を Gemini API（AI）が分析し、好みに合わせた書籍を提案します。

### 🧠 読書傾向要約

* **エンドポイント**: `POST /summary`
* **概要**: Gemini API を使用して、蓄積された読書データからユーザーの読書傾向を要約し、次のアクションを提案します。

### 🖼 画像アップロード

* **エンドポイント**: `POST /upload`
* **概要**: ImgBB API を介して、書籍のカバー画像などをホスティングします。

---

## 🛠 技術スタック

* **Language**: Go 1.21
* **Framework**: Gin Web Framework
* **API Integration**:

  * Google Books API
  * Gemini API
  * ImgBB API
* **Tooling**:

  * `godotenv`：環境変数管理
  * `gin-contrib/cors`：CORS 設定

---

## 🔑 環境変数の設定

実行には以下の API キーが必要です。
プロジェクトルートに `.env` ファイルを作成し、設定してください。

```env
# Google Books API Key
GOOGLE_BOOKS_API_KEY=your_google_books_api_key

# Gemini API Key (Google AI Studio)
GEMINI_API_KEY=your_gemini_api_key

# ImgBB API Key
IMGBB_API_KEY=your_imgbb_api_key

# Server Port (Optional: Default 8080)
PORT=8080
```

---

## 🛣 API エンドポイント一覧

| メソッド | エンドポイント       | 説明         | パラメータ / ボディ                        |
| ---- | ------------- | ---------- | ---------------------------------- |
| GET  | `/books`      | 書籍検索       | `searchQuery`, `maxResults`（Query） |
| POST | `/recommends` | AI レコメンド生成 | `LibraryRequest`（JSON）             |
| POST | `/summary`    | 読書傾向の要約    | `LibraryRequest`（JSON）             |
| POST | `/upload`     | 画像アップロード   | `image`（multipart/form-data）       |

---

## 🚀 開発環境の起動

### 1. 依存関係のインストール

```bash
go mod tidy
```

### 2. サーバーの起動

```bash
# .env を読み込んで起動します
go run main.go
```

### 3. 動作確認

```bash
curl "http://localhost:8080/books?searchQuery=Go言語"
```
