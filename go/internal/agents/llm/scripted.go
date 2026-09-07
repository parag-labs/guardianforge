package llm

import (
	"context"

	"github.com/parag-labs/guardianforge/go/internal/models"
)

// ScriptedLLM returns a fixed raw response (or error) so the supervisor's validator can be
// driven with adversarial output: malformed JSON, an invented intervention type, or an
// intervention proposed when the policy mode is observe-only.
type ScriptedLLM struct {
	Raw       string
	Err       error
	Narrative string
}

// Name identifies the provider.
func (ScriptedLLM) Name() string { return "scripted" }

// Synthesize returns the scripted response.
func (s ScriptedLLM) Synthesize(_ context.Context, _ models.Signal) (string, error) {
	if s.Err != nil {
		return "", s.Err
	}
	return s.Raw, nil
}

// Explain returns the scripted narrative.
func (s ScriptedLLM) Explain(_ context.Context, _ models.Intervention, _ models.Signal) (string, error) {
	return s.Narrative, nil
}
