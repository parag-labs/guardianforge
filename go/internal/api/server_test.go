package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/parag-labs/guardianforge/go/internal/agents"
	"github.com/parag-labs/guardianforge/go/internal/agents/llm"
	"github.com/parag-labs/guardianforge/go/internal/fleet"
	"github.com/parag-labs/guardianforge/go/internal/runtime"
)

func testServer() http.Handler {
	f := fleet.New()
	sup := agents.NewSupervisor(llm.MockLLM{})
	eng := runtime.New(fleet.DefaultPolicies(), f, sup, func() time.Time { return time.Unix(0, 0).UTC() })
	return New(eng, nil).Handler()
}

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestIngestEventReturnsOutcome(t *testing.T) {
	h := testServer()
	rr := do(t, h, "POST", "/events", `{"agent_id":"reaper","type":"TOOL_CALL","tool":"delete_database"}`)
	if rr.Code != 200 {
		t.Fatalf("want 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "REVOKE_TOOL") {
		t.Fatalf("destructive tool should be revoked: %s", rr.Body.String())
	}
}

func TestRunScenarioAndAudit(t *testing.T) {
	h := testServer()
	rr := do(t, h, "POST", "/scenarios/privilege_escalation", "")
	if rr.Code != 200 || !strings.Contains(rr.Body.String(), "ESCALATE") {
		t.Fatalf("privilege escalation should escalate: %d %s", rr.Code, rr.Body.String())
	}
	// The audit endpoint should report an intact chain.
	a := do(t, h, "GET", "/audit", "")
	if !strings.Contains(a.Body.String(), `"intact":true`) {
		t.Fatalf("audit chain should be intact: %s", a.Body.String())
	}
}

func TestHITLFlow(t *testing.T) {
	h := testServer()
	do(t, h, "POST", "/scenarios/privilege_escalation", "")
	list := do(t, h, "GET", "/hitl", "")
	if !strings.Contains(list.Body.String(), "intv-") {
		t.Fatalf("expected a pending escalation: %s", list.Body.String())
	}
}

func TestPoliciesRoundTrip(t *testing.T) {
	h := testServer()
	get := do(t, h, "GET", "/policies", "")
	if get.Code != 200 || !strings.Contains(get.Body.String(), "destructive-tools") {
		t.Fatalf("default policies should be present: %s", get.Body.String())
	}
	put := do(t, h, "PUT", "/policies", `[]`)
	if put.Code != 200 || !strings.Contains(put.Body.String(), `"count":0`) {
		t.Fatalf("policies should be replaceable: %d %s", put.Code, put.Body.String())
	}
}

func TestHealthz(t *testing.T) {
	h := testServer()
	if do(t, h, "GET", "/healthz", "").Code != 200 {
		t.Fatal("healthz should be 200")
	}
}
