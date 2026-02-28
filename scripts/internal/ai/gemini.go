package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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

func (g *GeminiClient) StreamGptJson(ctx context.Context, query string) string {
	apiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1/models/gemini-1.5-flash:generateContent?key=%s", g.cfg.GeminiKey)
	prompt := fmt.Sprintf("JSON Schema only for: '%s'. Keys: type(movie|series), year(int), country(ISO), title(str)", query)

	payload := map[string]interface{}{
		"contents": []interface{}{map[string]interface{}{"parts": []interface{}{map[string]interface{}{"text": prompt}}}},
		"generationConfig": map[string]interface{}{"temperature": 0.1, "responseMimeType": "application/json"},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil || resp.StatusCode != 200 { return "" }
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	candidates, ok := result["candidates"].([]interface{})
	if !ok || len(candidates) == 0 { return "" }
	content := candidates[0].(map[string]interface{})["content"].(map[string]interface{})
	parts := content["parts"].([]interface{})
	if len(parts) == 0 { return "" }
	return parts[0].(map[string]interface{})["text"].(string)
}
