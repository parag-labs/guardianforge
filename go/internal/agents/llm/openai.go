package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/parag-labs/guardianforge/go/internal/models"
)

// OpenAIConfig configures the OpenAI-compatible client (OpenAI, Azure, Ollama, vLLM).
type OpenAIConfig struct {
	BaseURL string
	APIKey  string
	Model   string
	Timeout time.Duration
}

// OpenAILLM is the production client. Compiled and usable, but never exercised in CI -
// tests use the mock/scripted models, so no API key is ever required.
type OpenAILLM struct {
	cfg  OpenAIConfig
	http *http.Client
}

// NewOpenAI builds a client.
func NewOpenAI(cfg OpenAIConfig) *OpenAILLM {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	return &OpenAILLM{cfg: cfg, http: &http.Client{Timeout: cfg.Timeout}}
}

// Name identifies the provider.
func (c *OpenAILLM) Name() string { return "openai:" + c.cfg.Model }

const supervisorSystem = `You are GuardianForge's supervisor. You receive an aggregated signal about one
agent event (policy violations, anomalies, trust, the strictest policy mode). Return a single JSON object:
{"intervene":bool,"type":"PAUSE|REVOKE_TOOL|INJECT_CONSTRAINT|FORCE_REPLAN|NOTIFY|ESCALATE","reason":string,
"escalate":bool,"confidence":number}. If the policy mode is OBSERVE you must not intervene. Never invent a type.`

// Synthesize asks the model for a decision.
func (c *OpenAILLM) Synthesize(ctx context.Context, s models.Signal) (string, error) {
	payload, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return c.chat(ctx, supervisorSystem, string(payload))
}

// Explain asks the model for a narrative.
func (c *OpenAILLM) Explain(ctx context.Context, i models.Intervention, s models.Signal) (string, error) {
	payload, _ := json.Marshal(map[string]any{"intervention": i, "signal": s})
	return c.chat(ctx, "Explain, in 2-3 sentences for a compliance reviewer, why this intervention was issued.", string(payload))
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (c *OpenAILLM) chat(ctx context.Context, system, user string) (string, error) {
	reqBody, err := json.Marshal(map[string]any{
		"model": c.cfg.Model,
		"messages": []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	})
	if err != nil {
		return "", err
	}
	url := strings.TrimRight(c.cfg.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var parsed struct {
		Choices []struct {
			Message chatMessage `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("llm returned no choices")
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}
