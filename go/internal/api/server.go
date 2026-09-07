// Package api is GuardianForge's admin/control surface (net/http, no framework): ingest
// events, inspect policies, browse the audit trail, and work the human-in-the-loop queue.
// It is read-and-control only - it never lets a caller bypass the deterministic governance
// loop or the audit log.
package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/parag-labs/guardianforge/go/internal/fleet"
	"github.com/parag-labs/guardianforge/go/internal/models"
	"github.com/parag-labs/guardianforge/go/internal/runtime"
)

// Server wraps the governance engine with HTTP handlers.
type Server struct {
	engine *runtime.Engine
	log    *slog.Logger
}

// New builds a server over an engine.
func New(engine *runtime.Engine, log *slog.Logger) *Server {
	if log == nil {
		log = slog.Default()
	}
	return &Server{engine: engine, log: log}
}

// Handler builds the routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("GET /metrics", s.metrics)
	mux.HandleFunc("POST /events", s.ingest)
	mux.HandleFunc("GET /scenarios", s.listScenarios)
	mux.HandleFunc("POST /scenarios/{id}", s.runScenario)
	mux.HandleFunc("GET /policies", s.getPolicies)
	mux.HandleFunc("PUT /policies", s.putPolicies)
	mux.HandleFunc("GET /audit", s.getAudit)
	mux.HandleFunc("GET /hitl", s.getHITL)
	mux.HandleFunc("POST /hitl/{id}/resolve", s.resolveHITL)
	return s.logging(mux)
}

func (s *Server) metrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprint(w, s.engine.Metrics.Expose())
}

func (s *Server) ingest(w http.ResponseWriter, r *http.Request) {
	var ev models.AgentEvent
	if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
		writeErr(w, 400, "invalid event JSON")
		return
	}
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now().UTC()
	}
	out := s.engine.Process(r.Context(), ev)
	writeJSON(w, 200, out)
}

func (s *Server) listScenarios(w http.ResponseWriter, _ *http.Request) {
	type sc struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	var out []sc
	for _, x := range fleet.Scenarios() {
		out = append(out, sc{ID: x.ID, Title: x.Title})
	}
	writeJSON(w, 200, out)
}

func (s *Server) runScenario(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	scn, ok := fleet.ScenarioByID(id)
	if !ok {
		writeErr(w, 404, "unknown scenario")
		return
	}
	var outcomes []runtime.Outcome
	for _, ev := range scn.Events {
		outcomes = append(outcomes, s.engine.Process(r.Context(), ev))
	}
	writeJSON(w, 200, map[string]any{"scenario": id, "outcomes": outcomes, "audit_intact": s.engine.Audit.Verify()})
}

func (s *Server) getPolicies(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, s.engine.Policy.Policies())
}

func (s *Server) putPolicies(w http.ResponseWriter, r *http.Request) {
	var policies []models.Policy
	if err := json.NewDecoder(r.Body).Decode(&policies); err != nil {
		writeErr(w, 400, "invalid policies JSON")
		return
	}
	s.engine.Policy.SetPolicies(policies)
	writeJSON(w, 200, map[string]int{"count": len(policies)})
}

func (s *Server) getAudit(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, map[string]any{"entries": s.engine.Audit.Entries(), "intact": s.engine.Audit.Verify()})
}

func (s *Server) getHITL(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, s.engine.PendingHITL())
}

func (s *Server) resolveHITL(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Decision string `json:"decision"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Decision == "" {
		body.Decision = "acknowledged"
	}
	if !s.engine.ResolveHITL(id, body.Decision) {
		writeErr(w, 404, "no such pending escalation")
		return
	}
	writeJSON(w, 200, map[string]string{"resolved": id})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func (s *Server) logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.log.Info("request", "method", r.Method, "path", r.URL.Path, "dur_ms", time.Since(start).Milliseconds())
	})
}
