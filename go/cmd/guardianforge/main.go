// Command guardianforge runs the GuardianForge governance service. It monitors a synthetic
// agent fleet and enforces policies through the deterministic engine, using the mock
// supervisor by default (no API key). Point it at an OpenAI-compatible endpoint with the
// LLM_* environment variables to use a live model for supervisor synthesis.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/parag-labs/guardianforge/go/internal/agents"
	"github.com/parag-labs/guardianforge/go/internal/agents/llm"
	"github.com/parag-labs/guardianforge/go/internal/api"
	"github.com/parag-labs/guardianforge/go/internal/eval"
	"github.com/parag-labs/guardianforge/go/internal/fleet"
	"github.com/parag-labs/guardianforge/go/internal/runtime"
)

func main() {
	addr := flag.String("addr", envOr("ADDR", ":8080"), "listen address")
	runEval := flag.Bool("eval", false, "run the governance scorecard and exit")
	flag.Parse()

	if *runEval {
		runScorecard()
		return
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	model := selectModel(log)
	sup := agents.NewSupervisor(model)
	eng := runtime.New(fleet.DefaultPolicies(), fleet.New(), sup, nil)
	srv := api.New(eng, log)

	httpSrv := &http.Server{Addr: *addr, Handler: srv.Handler(), ReadHeaderTimeout: 5 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Info("guardianforge listening", "addr", *addr, "model", model.Name())
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)
}

func runScorecard() {
	r := eval.Run()
	fmt.Print(r.Format())
	if r.DetectionAccuracy() < 1.0 || r.FalsePositives > 0 || !r.AuditIntact {
		fmt.Fprintln(os.Stderr, "FAIL: governance quality bar not met")
		os.Exit(1)
	}
}

func selectModel(log *slog.Logger) llm.LLMClient {
	base := os.Getenv("LLM_BASE_URL")
	if base == "" {
		return llm.MockLLM{}
	}
	log.Info("using OpenAI-compatible model", "base_url", base)
	return llm.NewOpenAI(llm.OpenAIConfig{
		BaseURL: base,
		APIKey:  os.Getenv("LLM_API_KEY"),
		Model:   envOr("LLM_MODEL", "gpt-4o-mini"),
	})
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
