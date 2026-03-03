package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"myflixbot.local/internal/config"
)

type GeminiClient struct {
	cfg        *config.Config
	httpClient *http.Client
}

func NewGeminiClient(cfg *config.Config, client *http.Client) *GeminiClient {
	return &GeminiClient{
		cfg:        cfg,
		httpClient: client,
	}
}

type GeminiRequest struct {
	Contents         []Content        `json:"contents"`
	GenerationConfig GenerationConfig `json:"generationConfig"`
}

type Content struct {
	Parts []Part `json:"parts"`
}

type Part struct {
	Text string `json:"text"`
}

type GenerationConfig struct {
	Temperature      float32 `json:"temperature"`
	ResponseMimeType string  `json:"responseMimeType"`
}

type GeminiResponse struct {
	Candidates []Candidate `json:"candidates"`
}

type Candidate struct {
	Content Content `json:"content"`
}

func (g *GeminiClient) StreamGptJson(ctx context.Context, query string) string {
	// Basic sanitization to prevent prompt injection
	safeQuery := strings.ReplaceAll(query, "'", "\\'")
	safeQuery = strings.ReplaceAll(safeQuery, "\n", " ")

	apiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1/models/gemini-1.5-flash:generateContent?key=%s", g.cfg.GeminiKey)
	prompt := fmt.Sprintf("JSON Schema only for: '%s'. Keys: type(movie|series), year(int), country(ISO), title(str)", safeQuery)

	payload := GeminiRequest{
		Contents: []Content{
			{Parts: []Part{{Text: prompt}}},
		},
		GenerationConfig: GenerationConfig{
			Temperature:      0.1,
			ResponseMimeType: "application/json",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		slog.Error("Failed to marshal Gemini request", "error", err)
		return ""
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(body))
	if err != nil {
		slog.Error("Failed to create Gemini request", "error", err)
		return ""
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		slog.Error("Failed to execute Gemini request", "error", err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("Gemini API returned non-200 status", "status", resp.StatusCode)
		return ""
	}

	var result GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		slog.Error("Failed to decode Gemini response", "error", err)
		return ""
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return ""
	}
	return result.Candidates[0].Content.Parts[0].Text
}
