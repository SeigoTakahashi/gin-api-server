package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
)

// ImgBBのレスポンス構造体
type ImgBBResponse struct {
	Data struct {
		URL string `json:"url"`
	} `json:"data"`
	Success bool `json:"success"`
}

// ImgBBへ画像をアップロードする内部関数
func uploadToImgBB(file io.Reader, filename string, apiKey string, client *http.Client) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("imgBB API キーが設定されていません")
	}

	apiURL := "https://api.imgbb.com/1/upload?key=" + url.QueryEscape(apiKey)

	// マルチパートボディの作成
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("image", filename)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("画像データの読み込みに失敗しました: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("マルチパートデータの作成に失敗しました: %w", err)
	}

	// リクエスト送信
	req, err := http.NewRequest("POST", apiURL, body)
	if err != nil {
		return "", fmt.Errorf("ImgBB リクエストの作成に失敗しました: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ImgBB API の呼び出しに失敗しました: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("ImgBB API が HTTP %d を返しました", resp.StatusCode)
	}

	// レスポンス解析
	var imgBBRes ImgBBResponse
	if err := json.NewDecoder(resp.Body).Decode(&imgBBRes); err != nil {
		return "", fmt.Errorf("ImgBB レスポンスの解析に失敗しました: %w", err)
	}
	if !imgBBRes.Success || imgBBRes.Data.URL == "" {
		return "", fmt.Errorf("ImgBB API のレスポンスに画像URLが含まれていません")
	}

	return imgBBRes.Data.URL, nil
}
