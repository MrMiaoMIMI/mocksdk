package httpadapter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/MrMiaoMIMI/mocksdk"
)

func TestNormalizeHTTPRequestPreservesRequestShapeAndBodyReplay(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "http://mockserver.local/api/v1/search?q=one&q=two", strings.NewReader(`{"name":"demo"}`))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Host = "Demo.COM:8443"
	req.RemoteAddr = "10.1.2.3:4567"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Trace-ID", "trace-normalize")
	req.Header.Set("X-Forwarded-For", "192.168.0.10, 10.0.0.1")
	req.Header.Set("X-Forwarded-Host", "Upstream.EXAMPLE:9443")
	req.Header.Set("X-Forwarded-Proto", "https")

	event, err := NormalizeHTTPRequest(req, "workspace")
	if err != nil {
		t.Fatalf("NormalizeHTTPRequest() error = %v", err)
	}

	if event.Protocol != "http" || event.Operation != "request" {
		t.Fatalf("unexpected protocol/operation: %+v", event)
	}
	if event.Namespace != "workspace" {
		t.Fatalf("unexpected namespace: %s", event.Namespace)
	}
	if event.Request["method"] != http.MethodPost {
		t.Fatalf("unexpected method: %s", event.Request["method"])
	}
	if event.Request["scheme"] != "https" {
		t.Fatalf("unexpected scheme: %s", event.Request["scheme"])
	}
	if event.Request["host"] != "upstream.example" {
		t.Fatalf("unexpected host: %s", event.Request["host"])
	}
	if event.Request["original_host"] != "upstream.example:9443" {
		t.Fatalf("unexpected original host: %s", event.Request["original_host"])
	}
	if event.Request["path"] != "/api/v1/search" {
		t.Fatalf("unexpected path: %s", event.Request["path"])
	}
	query := event.Request["query"].(map[string][]string)
	if got := query["q"]; len(got) != 2 || got[0] != "one" || got[1] != "two" {
		t.Fatalf("unexpected query: %+v", event.Request["query"])
	}
	headers := event.Request["headers"].(map[string][]string)
	if got := headers["content-type"]; len(got) != 1 || got[0] != "application/json" {
		t.Fatalf("unexpected headers: %+v", event.Request["headers"])
	}
	if event.Request["raw_body"] != `{"name":"demo"}` {
		t.Fatalf("unexpected raw body: %s", event.Request["raw_body"])
	}
	parsed, ok := event.Request["body"].(map[string]interface{})
	if !ok || parsed["name"] != "demo" {
		t.Fatalf("unexpected parsed body: %#v", event.Request["body"])
	}
	if event.Request["client_ip"] != "192.168.0.10" {
		t.Fatalf("unexpected client ip: %s", event.Request["client_ip"])
	}
	if event.Meta.TraceID != "trace-normalize" {
		t.Fatalf("unexpected trace id: %s", event.Meta.TraceID)
	}

	replayed, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read replayed body: %v", err)
	}
	if string(replayed) != `{"name":"demo"}` {
		t.Fatalf("unexpected replayed body: %s", string(replayed))
	}
	secondBody, err := req.GetBody()
	if err != nil {
		t.Fatalf("GetBody() error = %v", err)
	}
	defer secondBody.Close()
	secondReplay, err := io.ReadAll(secondBody)
	if err != nil {
		t.Fatalf("read GetBody body: %v", err)
	}
	if string(secondReplay) != `{"name":"demo"}` {
		t.Fatalf("unexpected GetBody body: %s", string(secondReplay))
	}
}

func TestDecideNormalizesHTTPEventAndCallsClient(t *testing.T) {
	var captured map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{"decision":{"kind":"forward","fallback":true,"trace":{"fallback_reason":"rule_miss"},"forward":{"timeout_ms":1000}}}}`))
	}))
	defer server.Close()

	client, err := mocksdk.NewClient(mocksdk.Config{MockServerURL: server.URL, Namespace: "configured"})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	req, err := http.NewRequest(http.MethodGet, "http://example.com/api/check", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	decision, err := Decide(context.Background(), client, req, "")
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	event := captured["event"].(map[string]interface{})
	request := event["request"].(map[string]interface{})
	if event["protocol"] != "http" || event["namespace"] != "configured" {
		t.Fatalf("unexpected event: %+v", event)
	}
	if request["path"] != "/api/check" {
		t.Fatalf("unexpected request path: %+v", request)
	}
	if decision.Kind != mocksdk.DecisionKindForward || decision.Forward.TimeoutMS != 1000 {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestNormalizeHTTPRequestHandlesPlainTextAndEmptyBody(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "http://example.com/plain", strings.NewReader("plain text"))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	plainEvent, err := NormalizeHTTPRequest(req, "")
	if err != nil {
		t.Fatalf("NormalizeHTTPRequest() error = %v", err)
	}
	if plainEvent.Namespace != mocksdk.DefaultNamespace {
		t.Fatalf("unexpected default namespace: %s", plainEvent.Namespace)
	}
	if plainEvent.Request["raw_body"] != "plain text" {
		t.Fatalf("unexpected plain raw body: %s", plainEvent.Request["raw_body"])
	}
	if _, ok := plainEvent.Request["body"]; ok {
		t.Fatalf("plain text should not be parsed as JSON: %#v", plainEvent.Request["body"])
	}

	emptyReq, err := http.NewRequest(http.MethodGet, "http://example.com/empty", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	emptyEvent, err := NormalizeHTTPRequest(emptyReq, "default")
	if err != nil {
		t.Fatalf("NormalizeHTTPRequest() error = %v", err)
	}
	if _, ok := emptyEvent.Request["raw_body"]; ok {
		t.Fatalf("empty request should not include raw_body: %#v", emptyEvent.Request)
	}
	if _, ok := emptyEvent.Request["body"]; ok {
		t.Fatalf("empty request should not include body: %#v", emptyEvent.Request)
	}
}

func TestIsHopByHopHeader(t *testing.T) {
	for _, key := range []string{"Connection", "content-length", "Transfer-Encoding", "upgrade"} {
		if !IsHopByHopHeader(key) {
			t.Fatalf("expected %s to be hop-by-hop", key)
		}
	}
	if IsHopByHopHeader("X-Request-ID") {
		t.Fatalf("x-request-id should not be hop-by-hop")
	}
}

func TestApplyResponseWritesStatusHeadersAndBody(t *testing.T) {
	for name, decision := range map[string]mocksdk.Decision{
		"json": {
			Kind:     mocksdk.DecisionKindResponse,
			Response: httpResponseDecision(http.StatusCreated, `{"Content-Type":["application/json"],"X-Multi":["one","two"]}`, `{"ok":true}`),
		},
		"text": {
			Kind:     mocksdk.DecisionKindResponse,
			Response: httpResponseDecision(http.StatusAccepted, `{"Content-Type":["text/plain"]}`, `"plain text"`),
		},
		"empty": {
			Kind:     mocksdk.DecisionKindResponse,
			Response: httpResponseDecision(http.StatusNoContent, `null`, `null`),
		},
	} {
		t.Run(name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			if err := ApplyResponse(recorder, decision); err != nil {
				t.Fatalf("ApplyResponse() error = %v", err)
			}
			resp := recorder.Result()
			status := responseStatusForTest(t, decision.Response)
			if resp.StatusCode != status {
				t.Fatalf("unexpected status: got=%d want=%d", resp.StatusCode, status)
			}
			body, err := readAllAndClose(resp.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			switch name {
			case "json":
				if string(body) != `{"ok":true}` {
					t.Fatalf("unexpected json body: %s", string(body))
				}
				if got := resp.Header.Values("X-Multi"); len(got) != 2 || got[0] != "one" || got[1] != "two" {
					t.Fatalf("unexpected multi header: %+v", resp.Header)
				}
			case "text":
				if string(body) != "plain text" {
					t.Fatalf("unexpected text body: %s", string(body))
				}
			case "empty":
				if len(body) != 0 {
					t.Fatalf("unexpected empty body: %s", string(body))
				}
			}
		})
	}
}

func httpResponseDecision(status int, headers string, body string) *mocksdk.ResponseDecision {
	return &mocksdk.ResponseDecision{
		Protocol: Protocol,
		Payload:  json.RawMessage(`{"status":` + strconv.Itoa(status) + `,"headers":` + headers + `,"body":` + body + `}`),
	}
}

func responseStatusForTest(t *testing.T, response *mocksdk.ResponseDecision) int {
	t.Helper()
	payload, err := PayloadFromDecision(mocksdk.Decision{
		Kind:     mocksdk.DecisionKindResponse,
		Response: response,
	})
	if err != nil {
		t.Fatalf("PayloadFromDecision() error = %v", err)
	}
	return payload.Status
}
