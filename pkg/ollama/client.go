package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type GenerateRequest struct {
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
	Stream  bool   `json:"stream"`
}

type GenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &Client{
		baseURL: baseURL,
		http: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (c *Client) Generate(model, prompt string) (string, error) {
	body := GenerateRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.http.Post(c.baseURL+"/api/generate", "application/json", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("post request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var result GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return result.Response, nil
}
