package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
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
	Model string      `json:"model"`
	Input interface{} `json:"input"` // Can be string or []string
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
func (c *LocalEmbeddingClient) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	reqBody, _ := json.Marshal(ollamaEmbeddingRequest{
		Model: c.Model,
		Input: texts,
	})

	log.Printf("Embedding %d messages, request body size %d", len(texts), len(reqBody))

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
		body, _ := io.ReadAll(resp.Body)
		errMsg := string(body)
		log.Printf("Ollama error (status %d): %s", resp.StatusCode, errMsg)

		// Handle context length error: "input length exceeds the context length"
		if resp.StatusCode == 400 && (strings.Contains(errMsg, "exceeds the context length") || strings.Contains(errMsg, "too large")) {
			if len(texts) > 1 {
				// Split batch in half and retry
				log.Printf("Ollama context length exceeded. Splitting batch of %d into two chunks.", len(texts))
				mid := len(texts) / 2
				first, err1 := c.EmbedBatch(ctx, texts[:mid])
				if err1 != nil {
					return nil, err1
				}
				second, err2 := c.EmbedBatch(ctx, texts[mid:])
				if err2 != nil {
					return nil, err2
				}
				return append(first, second...), nil
			} else if len(texts) == 1 {
				// If a single message is too long, we must truncate it.
				// Based on nomic-embed-text's 512/2048 ctx limit, 1500 chars is safe for 512 tokens.
				log.Printf("Single message exceeds context length. Truncating to 1500 characters.")
				truncated := texts[0]
				if len(truncated) > 1500 {
					truncated = truncated[:1500]
				} else {
					// Extremely long tokens? Let's just drop a few more chars.
					truncated = truncated[:len(truncated)*3/4]
				}
				return c.EmbedBatch(ctx, []string{truncated})
			}
		}
		return nil, fmt.Errorf("ollama error: status %d - %s", resp.StatusCode, errMsg)
	}

	var res ollamaEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	return res.Embeddings, nil
}
