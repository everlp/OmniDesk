package rag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type EmbeddingRequest struct {
	Model   string `json:"model"`
	Content struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"content"`
}

type EmbeddingResponse struct {
	Embedding struct {
		Values []float32 `json:"values"`
	} `json:"embedding"`
}

// GetEmbedding 调用 Gemini API 获取文本向量
func GetEmbedding(apiKey string, text string) ([]float32, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-embedding-2:embedContent?key=%s", apiKey)

	reqBody := EmbeddingRequest{
		Model: "models/gemini-embedding-2",
	}
	reqBody.Content.Parts = append(reqBody.Content.Parts, struct {
		Text string `json:"text"`
	}{Text: text})

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Embedding API error (status %d): %s", resp.StatusCode, string(body))
	}

	var embResp EmbeddingResponse
	if err := json.Unmarshal(body, &embResp); err != nil {
		return nil, err
	}

	return embResp.Embedding.Values, nil
}
