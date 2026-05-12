package httpadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/MrMiaoMIMI/mocksdk"
)

func NormalizeHTTPRequest(r *http.Request, namespace string) (mocksdk.Event, error) {
	bodyBytes, err := readAndRestoreBody(r)
	if err != nil {
		return mocksdk.Event{}, err
	}

	var parsedBody interface{}
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &parsedBody); err != nil {
			parsedBody = nil
		}
	}

	request := mocksdk.EventRequest{
		"method":        r.Method,
		"scheme":        schemeFor(r),
		"host":          normalizedHost(hostFor(r)),
		"original_host": strings.ToLower(originalHostFor(r)),
		"path":          pathFor(r),
		"query":         cloneQuery(r),
		"headers":       cloneHeaders(r.Header),
		"client_ip":     clientIPFor(r),
	}
	if len(bodyBytes) > 0 {
		request["raw_body"] = string(bodyBytes)
		if parsedBody != nil {
			request["body"] = parsedBody
		}
	}

	return mocksdk.Event{
		Protocol:  "http",
		Operation: "request",
		Namespace: mocksdk.NormalizeNamespace(namespace),
		Request:   request,
		Meta: mocksdk.EventMeta{
			TraceID: r.Header.Get("X-Trace-ID"),
		},
	}, nil
}

func Decide(ctx context.Context, client *mocksdk.Client, r *http.Request, namespace string) (mocksdk.Decision, error) {
	event, err := NormalizeHTTPRequest(r, client.NamespaceFor(namespace))
	if err != nil {
		return mocksdk.Decision{}, err
	}
	return client.Decide(ctx, event)
}

func ApplyResponse(w http.ResponseWriter, decision mocksdk.Decision) error {
	if decision.Kind != mocksdk.DecisionKindResponse || decision.Response == nil {
		return &mocksdk.Error{Kind: mocksdk.ErrorKindConfig, Message: "decision is not a response decision"}
	}
	if decision.Response.Status < 100 || decision.Response.Status > 599 {
		return &mocksdk.Error{Kind: mocksdk.ErrorKindConfig, Message: "response decision status must be between 100 and 599"}
	}
	for key, values := range decision.Response.Headers {
		if IsHopByHopHeader(key) {
			continue
		}
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(decision.Response.Status)
	body, err := responseBodyBytes(decision.Response.Body)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return nil
	}
	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("write response decision body: %w", err)
	}
	return nil
}

func readAndRestoreBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	restoreRequestBody(r, bodyBytes)
	return bodyBytes, nil
}

func restoreRequestBody(r *http.Request, bodyBytes []byte) {
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	r.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(bodyBytes)), nil
	}
	r.ContentLength = int64(len(bodyBytes))
	if len(bodyBytes) == 0 {
		r.ContentLength = 0
	}
}

func schemeFor(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		return strings.ToLower(forwarded)
	}
	if r.URL != nil && r.URL.Scheme != "" {
		return strings.ToLower(r.URL.Scheme)
	}
	return "http"
}

func originalHostFor(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); forwarded != "" {
		return forwarded
	}
	if strings.TrimSpace(r.Host) != "" {
		return r.Host
	}
	if r.URL != nil {
		return r.URL.Host
	}
	return ""
}

func hostFor(r *http.Request) string {
	return originalHostFor(r)
}

func normalizedHost(host string) string {
	host = strings.TrimSpace(host)
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	return strings.ToLower(host)
}

func pathFor(r *http.Request) string {
	if r.URL == nil || r.URL.Path == "" {
		return "/"
	}
	return r.URL.Path
}

func cloneQuery(r *http.Request) map[string][]string {
	if r.URL == nil {
		return nil
	}
	query := r.URL.Query()
	if len(query) == 0 {
		return nil
	}
	cloned := make(map[string][]string, len(query))
	for key, values := range query {
		cloned[key] = append([]string(nil), values...)
	}
	return cloned
}

func cloneHeaders(headers http.Header) map[string][]string {
	if len(headers) == 0 {
		return nil
	}
	cloned := make(map[string][]string, len(headers))
	for key, values := range headers {
		cloned[strings.ToLower(key)] = append([]string(nil), values...)
	}
	return cloned
}

func clientIPFor(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func IsHopByHopHeader(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "connection", "content-length", "host", "keep-alive", "proxy-authenticate", "proxy-authorization", "te", "trailer", "transfer-encoding", "upgrade":
		return true
	default:
		return false
	}
}

func responseBodyBytes(raw json.RawMessage) ([]byte, error) {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return []byte(text), nil
	}
	return append([]byte(nil), raw...), nil
}

func readAllAndClose(body io.ReadCloser) ([]byte, error) {
	defer body.Close()
	return io.ReadAll(body)
}
