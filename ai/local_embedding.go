package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type LocalEmbeddingClient struct {
	URL   string
	Model string
}

func NewLocalEmbeddingClient(url, model string) *LocalEmbeddingClient {
	return &LocalEmbeddingClient{
		URL:   url,
		Model: model,
	}
}

type ollamaEmbeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type ollamaEmbeddingResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

func (c *LocalEmbeddingClient) EmbedText(ctx context.Context, text string) ([]float32, error) {
	reqBody, _ := json.Marshal(ollamaEmbeddingRequest{
		Model: c.Model,
		Input: text,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", c.URL+"/api/embed", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama error: status %d", resp.StatusCode)
	}

	var res ollamaEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	if len(res.Embeddings) == 0 {
		return nil, fmt.Errorf("ollama returned no embeddings")
	}

	return res.Embeddings[0], nil
}
