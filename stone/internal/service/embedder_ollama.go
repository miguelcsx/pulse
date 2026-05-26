package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// OllamaEmbedder uses a local Ollama instance to generate semantic vector embeddings.
type OllamaEmbedder struct {
	baseURL string
	model   string
	dim     int
	client  *http.Client
}

func NewOllamaEmbedder(baseURL, model string, dim int) *OllamaEmbedder {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "qwen3-embedding"
	}
	if dim <= 0 {
		dim = 1024 // default for qwen3-embedding
	}
	return &OllamaEmbedder{
		baseURL: baseURL,
		model:   model,
		dim:     dim,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (e *OllamaEmbedder) Dimensions() int {
	return e.dim
}

func (e *OllamaEmbedder) EmbedText(text string) ([]float64, error) {
	if text == "" {
		return make([]float64, e.dim), nil
	}

	payload := map[string]any{
		"model": e.model,
		"input": text,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ollama request: %w", err)
	}

	req, err := http.NewRequest("POST", e.baseURL+"/api/embed", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create ollama request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		slog.Error("ollama embedding request failed", "error", err)
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		slog.Error("ollama embedding returned non-200", "status", resp.StatusCode, "body", string(respBody))
		return nil, fmt.Errorf("ollama API error: status=%d", resp.StatusCode)
	}

	// /api/embed returns {"embeddings": [[...]]} — outer array wraps each input.
	var result struct {
		Embeddings [][]float64 `json:"embeddings"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode ollama response: %w", err)
	}

	if len(result.Embeddings) == 0 || len(result.Embeddings[0]) == 0 {
		return nil, fmt.Errorf("ollama returned no embedding data")
	}

	vec := result.Embeddings[0]

	// Ensure the returned embedding matches the expected dimensionality.
	// If it doesn't, truncate or pad it.
	if len(vec) != e.dim {
		adjusted := make([]float64, e.dim)
		copy(adjusted, vec)
		vec = adjusted
	}

	return vec, nil
}
