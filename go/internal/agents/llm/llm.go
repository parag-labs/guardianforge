// Package llm is GuardianForge's model boundary. The supervisor and audit-explainer
// agents reason through this interface; everything else depends only on it, so a
// deterministic mock powers CI while a real model runs in production. As everywhere in
// this system: the model proposes a decision, and the deterministic layer validates it
// before anything is enforced.
package llm

import (
	"context"

	"github.com/parag-labs/guardianforge/go/internal/models"
)

// LLMClient is the provider-agnostic model interface.
type LLMClient interface {
	// Name identifies the provider for logs.
	Name() string
	// Synthesize turns an aggregated signal into a raw JSON models.Decision. It returns
	// text, not a struct, so the supervisor owns parsing and validation.
	Synthesize(ctx context.Context, signal models.Signal) (raw string, err error)
	// Explain produces a human-readable narrative for an intervention (audit explainer).
	Explain(ctx context.Context, intervention models.Intervention, signal models.Signal) (string, error)
}
