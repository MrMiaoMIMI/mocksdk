package mocksdk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientDecideReturnsResponseDecision(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != decisionPath {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var req decisionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode decision request: %v", err)
		}
		if req.Event.Namespace != "default" {
			t.Fatalf("unexpected namespace: %s", req.Event.Namespace)
		}
		if req.Event.Request["path"] != "/api/sdk/hit" {
			t.Fatalf("unexpected request path: %s", req.Event.Request["path"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"code": 0,
			"data": {
				"decision": {
					"kind": "response",
					"matched": true,
					"trace": {
						"ruleset_id": "sdk-ruleset",
						"rule_id": "sdk-rule"
					},
					"response": {
						"status": 202,
						"headers": {
							"x-sdk-decision": ["hit"]
						},
						"body": {
							"source": "sdk",
							"hit": true
						}
					},
					"meta": {
						"trace_id": "trace-sdk"
					}
				}
			}
		}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{MockServerURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	decision, err := client.Decide(context.Background(), Event{
		Protocol:  "http",
		Namespace: "default",
		Request: EventRequest{
			"method": "GET",
			"host":   "demo.com",
			"path":   "/api/sdk/hit",
		},
		Meta: EventMeta{TraceID: "trace-sdk"},
	})
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if decision.Kind != DecisionKindResponse {
		t.Fatalf("unexpected decision kind: %s", decision.Kind)
	}
	if !decision.Matched {
		t.Fatalf("expected matched decision")
	}
	if decision.Trace.RulesetID != "sdk-ruleset" || decision.Trace.RuleID != "sdk-rule" {
		t.Fatalf("unexpected trace: %+v", decision.Trace)
	}
	if decision.Response == nil {
		t.Fatalf("expected response decision")
	}
	if decision.Response.Status != http.StatusAccepted {
		t.Fatalf("unexpected response status: %d", decision.Response.Status)
	}
	if got := decision.Response.Headers["x-sdk-decision"]; len(got) != 1 || got[0] != "hit" {
		t.Fatalf("unexpected response headers: %+v", decision.Response.Headers)
	}
	if !strings.Contains(string(decision.Response.Body), `"source"`) || !strings.Contains(string(decision.Response.Body), `"sdk"`) {
		t.Fatalf("unexpected response body: %s", string(decision.Response.Body))
	}
	if decision.Meta.TraceID != "trace-sdk" {
		t.Fatalf("unexpected trace id: %s", decision.Meta.TraceID)
	}
}

func TestClientDecideReturnsForwardDecision(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"code": 0,
			"data": {
				"decision": {
					"kind": "forward",
					"matched": false,
					"fallback": true,
					"trace": {
						"fallback_reason": "rule_miss"
					},
					"forward": {
						"timeout_ms": 5000
					},
					"meta": {
						"trace_id": "trace-forward"
					}
				}
			}
		}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{MockServerURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	decision, err := client.Decide(context.Background(), Event{
		Protocol:  "http",
		Namespace: "default",
		Request: EventRequest{
			"method": "GET",
			"host":   "demo.com",
			"path":   "/api/miss",
		},
		Meta: EventMeta{TraceID: "trace-forward"},
	})
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if decision.Kind != DecisionKindForward {
		t.Fatalf("unexpected decision kind: %s", decision.Kind)
	}
	if decision.Matched {
		t.Fatalf("expected unmatched forward decision")
	}
	if !decision.Fallback {
		t.Fatalf("expected fallback forward decision")
	}
	if decision.Trace.FallbackReason != "rule_miss" {
		t.Fatalf("unexpected fallback reason: %s", decision.Trace.FallbackReason)
	}
	if decision.Response != nil {
		t.Fatalf("forward decision should not include response payload")
	}
	if decision.Forward == nil {
		t.Fatalf("expected forward decision payload")
	}
	if decision.Forward.TimeoutMS != 5000 {
		t.Fatalf("unexpected forward timeout: %d", decision.Forward.TimeoutMS)
	}
	if decision.Meta.TraceID != "trace-forward" {
		t.Fatalf("unexpected trace id: %s", decision.Meta.TraceID)
	}
}
