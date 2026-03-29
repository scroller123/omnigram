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
	// Some versions of the SDK support OutputDimensionality.
	// If this causes a build error, we can remove it and truncate manually.
	// However, we want the API to do it if possible for efficiency.
	// Let's try to set it via the task-specific config or model field if available.
	// For now, let's just use the model name and truncate if the returned length is > 768.
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
