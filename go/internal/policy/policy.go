// Package policy is the deterministic policy evaluator. It checks structured rules against
// events - equality, contains, regexp, and windowed counts - and emits violations. This
// runs first, before any LLM: the fast deterministic path decides the clear cases, and
// only genuinely ambiguous ones are ever escalated to a model.
package policy

import (
	"regexp"
	"strings"
	"sync"

	"github.com/parag-labs/guardianforge/go/internal/models"
)

// Evaluator holds the policy set and the per-agent sliding history needed for
// count-based rules (e.g. "same tool called N times in a window").
type Evaluator struct {
	mu       sync.Mutex
	policies []models.Policy
	// history maps agentID -> recent event fingerprints for windowed counting.
	history map[string][]string
	regexps map[string]*regexp.Regexp
}

// New builds an evaluator over a policy set.
func New(policies []models.Policy) *Evaluator {
	e := &Evaluator{
		policies: policies,
		history:  map[string][]string{},
		regexps:  map[string]*regexp.Regexp{},
	}
	for _, p := range policies {
		for _, r := range p.Rules {
			if r.Op == models.OpMatches {
				if rx, err := regexp.Compile(r.Value); err == nil {
					e.regexps[r.RuleID] = rx
				}
			}
		}
	}
	return e
}

// Policies returns the current policy set.
func (e *Evaluator) Policies() []models.Policy {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]models.Policy(nil), e.policies...)
}

// SetPolicies replaces the policy set (used by the admin API).
func (e *Evaluator) SetPolicies(p []models.Policy) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.policies = p
	e.regexps = map[string]*regexp.Regexp{}
	for _, pol := range p {
		for _, r := range pol.Rules {
			if r.Op == models.OpMatches {
				if rx, err := regexp.Compile(r.Value); err == nil {
					e.regexps[r.RuleID] = rx
				}
			}
		}
	}
}

// Evaluate checks an event against every applicable policy and returns the violations,
// plus the strictest mode and severity among the policies that fired.
func (e *Evaluator) Evaluate(ev models.AgentEvent) ([]models.Violation, models.PolicyMode, models.Severity) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Record this event in the agent's window for count-based rules.
	fp := string(ev.Type) + ":" + ev.Tool + ":" + ev.Action
	e.history[ev.AgentID] = append(e.history[ev.AgentID], fp)
	if len(e.history[ev.AgentID]) > 256 {
		e.history[ev.AgentID] = e.history[ev.AgentID][len(e.history[ev.AgentID])-256:]
	}

	var violations []models.Violation
	mode := models.ModeObserve
	sev := models.SeverityInfo
	for _, p := range e.policies {
		for _, r := range p.Rules {
			if e.matches(ev, r, fp) {
				violations = append(violations, models.Violation{
					PolicyID: p.PolicyID, RuleID: r.RuleID, AgentID: ev.AgentID,
					Severity: r.Severity,
					Reason:   "rule " + r.RuleID + " matched on " + r.Field,
				})
				mode = stricterMode(mode, p.Mode)
				sev = higherSeverity(sev, r.Severity)
			}
		}
	}
	return violations, mode, sev
}

func (e *Evaluator) matches(ev models.AgentEvent, r models.Rule, fp string) bool {
	if r.Op == models.OpCountOver {
		window := r.WindowSize
		if window <= 0 {
			window = len(e.history[ev.AgentID])
		}
		hist := e.history[ev.AgentID]
		start := 0
		if len(hist) > window {
			start = len(hist) - window
		}
		count := 0
		for _, h := range hist[start:] {
			if strings.Contains(h, r.Value) {
				count++
			}
		}
		return count > r.Threshold
	}

	val := fieldValue(ev, r.Field)
	switch r.Op {
	case models.OpEquals:
		return val == r.Value
	case models.OpNotEquals:
		return val != r.Value
	case models.OpContains:
		return strings.Contains(val, r.Value)
	case models.OpMatches:
		if rx := e.regexps[r.RuleID]; rx != nil {
			return rx.MatchString(val)
		}
		return false
	default:
		return false
	}
}

func fieldValue(ev models.AgentEvent, field string) string {
	switch {
	case field == "type":
		return string(ev.Type)
	case field == "tool":
		return ev.Tool
	case field == "action":
		return ev.Action
	case strings.HasPrefix(field, "metadata."):
		return ev.Metadata[strings.TrimPrefix(field, "metadata.")]
	default:
		return ""
	}
}

// modeRank orders enforcement strictness.
func modeRank(m models.PolicyMode) int {
	switch m {
	case models.ModeObserve:
		return 0
	case models.ModeSoft:
		return 1
	case models.ModeHard:
		return 2
	case models.ModeEscalate:
		return 3
	default:
		return 0
	}
}

func stricterMode(a, b models.PolicyMode) models.PolicyMode {
	if modeRank(b) > modeRank(a) {
		return b
	}
	return a
}

func sevRank(s models.Severity) int {
	switch s {
	case models.SeverityInfo:
		return 0
	case models.SeverityLow:
		return 1
	case models.SeverityMedium:
		return 2
	case models.SeverityHigh:
		return 3
	case models.SeverityCritical:
		return 4
	default:
		return 0
	}
}

func higherSeverity(a, b models.Severity) models.Severity {
	if sevRank(b) > sevRank(a) {
		return b
	}
	return a
}
