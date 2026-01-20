package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
)

// ImgBBのレスポンス構造体
type ImgBBResponse struct {
	Data struct {
		URL string `json:"url"`
	} `json:"data"`
	Success bool `json:"success"`
}

// ImgBBへ画像をアップロードする内部関数
func uploadToImgBB(file io.Reader, filename string, apiKey string) (string, error) {
	if apiKey == "" {
		return "", nil
	}

	apiURL := "https://api.imgbb.com/1/upload?key=" + apiKey

	// マルチパートボディの作成
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("image", filename)
	if err != nil {
		return "", err
	}
	io.Copy(part, file)
	writer.Close()

	// リクエスト送信
	req, _ := http.NewRequest("POST", apiURL, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// レスポンス解析
	var imgBBRes ImgBBResponse
	if err := json.NewDecoder(resp.Body).Decode(&imgBBRes); err != nil {
		return "", err
	}

	return imgBBRes.Data.URL, nil
}
