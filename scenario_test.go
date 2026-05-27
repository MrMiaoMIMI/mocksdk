package mocksdk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateScenarioID(t *testing.T) {
	for _, scenarioID := range []string{
		"",
		"scn_1234",
		"scn_order_refund_ai_20260526",
		"scn_checkout-timeout-case1",
	} {
		if err := ValidateScenarioID(scenarioID); err != nil {
			t.Fatalf("ValidateScenarioID(%q) error = %v", scenarioID, err)
		}
	}

	longScenarioID := "scn_" + strings.Repeat("a", MaxScenarioIDLength)
	for _, scenarioID := range []string{
		"scenario_1234",
		"scn_1",
		"scn_bad space",
		"scn_bad/path",
		"scn_bad.path",
		"scn_bad:colon",
		"scn_中文",
		longScenarioID,
	} {
		if err := ValidateScenarioID(scenarioID); err == nil {
			t.Fatalf("ValidateScenarioID(%q) expected error", scenarioID)
		}
	}
}

func TestWithScenarioID(t *testing.T) {
	ctx := WithScenarioID(context.Background(), " scn_context_case1 ")
	scenarioID, ok := ScenarioIDFromContext(ctx)
	if !ok {
		t.Fatalf("expected scenario id in context")
	}
	if scenarioID != "scn_context_case1" {
		t.Fatalf("unexpected scenario id: %s", scenarioID)
	}
}

func TestClientDecideAddsScenarioIDFromContext(t *testing.T) {
	var gotScenarioID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req decisionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode decision request: %v", err)
		}
		gotScenarioID = req.Event.Meta.ScenarioID
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{"decision":{"kind":"forward","matched":false}}}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{MockServerURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	_, err = client.Decide(WithScenarioID(context.Background(), "scn_context_case1"), Event{
		Protocol:  "http",
		Namespace: "default",
	})
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if gotScenarioID != "scn_context_case1" {
		t.Fatalf("unexpected scenario id: %s", gotScenarioID)
	}
}

func TestClientDecideKeepsExplicitScenarioID(t *testing.T) {
	var gotScenarioID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req decisionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode decision request: %v", err)
		}
		gotScenarioID = req.Event.Meta.ScenarioID
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{"decision":{"kind":"forward","matched":false}}}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{MockServerURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	_, err = client.Decide(WithScenarioID(context.Background(), "scn_context_case1"), Event{
		Protocol:  "http",
		Namespace: "default",
		Meta:      EventMeta{ScenarioID: "scn_explicit_case1"},
	})
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if gotScenarioID != "scn_explicit_case1" {
		t.Fatalf("unexpected scenario id: %s", gotScenarioID)
	}
}

func TestClientDecideRejectsInvalidScenarioID(t *testing.T) {
	client, err := NewClient(Config{MockServerURL: "http://mockserver.local"})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	_, err = client.Decide(WithScenarioID(context.Background(), "bad scenario"), Event{
		Protocol:  "http",
		Namespace: "default",
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !IsErrorKind(err, ErrorKindConfig) {
		t.Fatalf("expected config error, got %T %v", err, err)
	}
}
