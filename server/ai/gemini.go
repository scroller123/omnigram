package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type Embedder interface {
	EmbedText(ctx context.Context, text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}

type GeminiClient struct {
	client *genai.Client
}

func NewGeminiClient(ctx context.Context, apiKey string) (*GeminiClient, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}
	return &GeminiClient{client: client}, nil
}

func (c *GeminiClient) Close() {
	c.client.Close()
}

func (c *GeminiClient) EmbedText(ctx context.Context, text string) ([]float32, error) {
	model := c.client.EmbeddingModel("gemini-embedding-001")
	res, err := model.EmbedContent(ctx, genai.Text(text))
	if err != nil {
		return nil, err
	}

	val := res.Embedding.Values
	if len(val) > 768 {
		return val[:768], nil
	}
	return val, nil
}

func (c *GeminiClient) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	model := c.client.EmbeddingModel("gemini-embedding-001")
	batch := model.NewBatch()
	for _, t := range texts {
		batch.AddContent(genai.Text(t))
	}

	res, err := model.BatchEmbedContents(ctx, batch)
	if err != nil {
		return nil, err
	}

	var results [][]float32
	for _, emb := range res.Embeddings {
		val := emb.Values
		if len(val) > 768 {
			val = val[:768]
		}
		results = append(results, val)
	}
	return results, nil
}

func (c *GeminiClient) AnalyzeMessages(ctx context.Context, query string, messages []string) (string, error) {
	model := c.client.GenerativeModel("gemini-2.5-flash")

	prompt := fmt.Sprintf("Print result in russian. Analyze the following Telegram messages to answer the query: \"%s\"\n\nMessages:\n%s\n\nAnswer concisely.", query, strings.Join(messages, "\n"))

	res, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", err
	}

	if len(res.Candidates) == 0 || len(res.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from Gemini")
	}

	var builder strings.Builder
	for _, part := range res.Candidates[0].Content.Parts {
		builder.WriteString(fmt.Sprint(part))
	}

	return builder.String(), nil
}
