package mocksdk

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClientEnvironmentOverridesOptions(t *testing.T) {
	t.Setenv(EnvMockServerHost, "http://env-mockserver.local/")
	t.Setenv(EnvMockServerAPIPrefix, "/env-prefix")
	t.Setenv(EnvNamespaceID, "env_namespace")
	t.Setenv(EnvTimeoutMS, "250")

	client, err := NewClient(Config{
		MockServerURL: "http://option-mockserver.local",
		APIPrefix:     "/option-prefix",
		Namespace:     "option_namespace",
		Timeout:       time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client.mockServerURL != "http://env-mockserver.local/env-prefix" {
		t.Fatalf("unexpected mockserver url: %s", client.mockServerURL)
	}
	if client.namespace != "env_namespace" {
		t.Fatalf("unexpected namespace: %s", client.namespace)
	}
	if client.timeout != 250*time.Millisecond {
		t.Fatalf("unexpected timeout: %s", client.timeout)
	}
}

func TestNewClientAppliesAPIPrefix(t *testing.T) {
	for name, tc := range map[string]struct {
		config Config
		want   string
	}{
		"option prefix": {
			config: Config{MockServerURL: "http://mockserver.local", APIPrefix: "tenant-a/"},
			want:   "http://mockserver.local/tenant-a",
		},
		"base url already contains prefix": {
			config: Config{MockServerURL: "http://mockserver.local/tenant-a/", APIPrefix: "/tenant-a"},
			want:   "http://mockserver.local/tenant-a",
		},
		"base url path plus prefix": {
			config: Config{MockServerURL: "http://mockserver.local/proxy", APIPrefix: "/tenant-a"},
			want:   "http://mockserver.local/proxy/tenant-a",
		},
	} {
		t.Run(name, func(t *testing.T) {
			client, err := NewClient(tc.config)
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}
			if client.mockServerURL != tc.want {
				t.Fatalf("unexpected mockserver url: %s", client.mockServerURL)
			}
		})
	}
}

func TestNewClientValidationErrorsAreConfigErrors(t *testing.T) {
	for name, config := range map[string]Config{
		"missing url":        {},
		"invalid url":        {MockServerURL: "://bad"},
		"relative url":       {MockServerURL: "mockserver.local"},
		"query in url":       {MockServerURL: "http://mockserver.local?tenant=a"},
		"invalid api prefix": {MockServerURL: "http://mockserver.local", APIPrefix: "http://proxy.example.com"},
		"invalid namespace":  {MockServerURL: "http://mockserver.local", Namespace: "bad namespace"},
		"negative timeout":   {MockServerURL: "http://mockserver.local", Timeout: -time.Second},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := NewClient(config)
			if err == nil {
				t.Fatalf("expected error")
			}
			if !IsErrorKind(err, ErrorKindConfig) {
				t.Fatalf("expected config error, got %T %v", err, err)
			}
		})
	}
}

func TestClientDecideClassifiesServerDecodeTransportAndTimeoutErrors(t *testing.T) {
	t.Run("server status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not available", http.StatusServiceUnavailable)
		}))
		defer server.Close()

		client, err := NewClient(Config{MockServerURL: server.URL})
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
		_, err = client.Decide(context.Background(), Event{Protocol: "http", Namespace: "default"})
		if !IsErrorKind(err, ErrorKindServer) {
			t.Fatalf("expected server error, got %T %v", err, err)
		}
		var sdkErr *Error
		if !errors.As(err, &sdkErr) || sdkErr.StatusCode != http.StatusServiceUnavailable {
			t.Fatalf("expected status on sdk error, got %v", err)
		}
	})

	t.Run("malformed json", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`not-json`))
		}))
		defer server.Close()

		client, err := NewClient(Config{MockServerURL: server.URL})
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
		_, err = client.Decide(context.Background(), Event{Protocol: "http", Namespace: "default"})
		if !IsErrorKind(err, ErrorKindDecode) {
			t.Fatalf("expected decode error, got %T %v", err, err)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		}))
		defer server.Close()

		client, err := NewClient(Config{MockServerURL: server.URL, Timeout: time.Nanosecond})
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
		_, err = client.Decide(context.Background(), Event{Protocol: "http", Namespace: "default"})
		if !IsErrorKind(err, ErrorKindTransport) {
			t.Fatalf("expected transport error, got %T %v", err, err)
		}
	})
}

func TestClientUsesCustomHTTPClient(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"code":0,"data":{"decision":{"kind":"forward","matched":false}}}`)),
			Request:    r,
		}, nil
	})
	client, err := NewClient(Config{
		MockServerURL: "http://mockserver.local",
		HTTPClient:    &http.Client{Transport: transport},
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	decision, err := client.Decide(context.Background(), Event{Protocol: "http", Namespace: "default"})
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if decision.Kind != DecisionKindForward {
		t.Fatalf("unexpected decision kind: %s", decision.Kind)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
