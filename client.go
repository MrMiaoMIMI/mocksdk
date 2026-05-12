package mocksdk

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const decisionPath = "/mockserver/api/v1/sdk/decision"

const (
	EnvMockServerHost = "MOCKSERVER_HOST"
	EnvNamespaceID    = "MOCKSERVER_NAMESPACE_ID"
	EnvTimeoutMS      = "MOCKSERVER_TIMEOUT_MS"
)

type Config struct {
	MockServerURL string
	Namespace     string
	Timeout       time.Duration
	HTTPClient    *http.Client
}

type Client struct {
	mockServerURL string
	namespace     string
	timeout       time.Duration
	httpClient    *http.Client
}

type Decider interface {
	Decide(ctx context.Context, event Event) (Decision, error)
}

func NewClient(config Config) (*Client, error) {
	config, err := resolveConfig(config)
	if err != nil {
		return nil, err
	}
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		mockServerURL: config.MockServerURL,
		namespace:     config.Namespace,
		timeout:       config.Timeout,
		httpClient:    httpClient,
	}, nil
}

func resolveConfig(config Config) (Config, error) {
	if value := strings.TrimSpace(os.Getenv(EnvMockServerHost)); value != "" {
		config.MockServerURL = value
	}
	if value := strings.TrimSpace(os.Getenv(EnvNamespaceID)); value != "" {
		config.Namespace = value
	}
	if value := strings.TrimSpace(os.Getenv(EnvTimeoutMS)); value != "" {
		timeoutMS, err := strconv.Atoi(value)
		if err != nil || timeoutMS < 0 {
			return Config{}, newError(ErrorKindConfig, EnvTimeoutMS+" must be a non-negative integer", err)
		}
		if timeoutMS > 0 {
			config.Timeout = time.Duration(timeoutMS) * time.Millisecond
		}
	}

	baseURL := strings.TrimRight(strings.TrimSpace(config.MockServerURL), "/")
	if baseURL == "" {
		return Config{}, newError(ErrorKindConfig, "mockserver url is required", nil)
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return Config{}, newError(ErrorKindConfig, "parse mockserver url", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return Config{}, newError(ErrorKindConfig, "mockserver url must include scheme and host", nil)
	}
	namespace := normalizedNamespace(config.Namespace)
	if !isValidNamespace(namespace) {
		return Config{}, newError(ErrorKindConfig, "namespace may only contain letters, numbers, underscores, and hyphens", nil)
	}
	config.MockServerURL = baseURL
	config.Namespace = namespace
	if config.Timeout < 0 {
		return Config{}, newError(ErrorKindConfig, "timeout must not be negative", nil)
	}
	return config, nil
}

func (c *Client) Decide(ctx context.Context, event Event) (Decision, error) {
	if c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}
	payload, err := json.Marshal(decisionRequest{Event: event})
	if err != nil {
		return Decision{}, newError(ErrorKindConfig, "marshal decision request", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.mockServerURL+decisionPath, bytes.NewReader(payload))
	if err != nil {
		return Decision{}, newError(ErrorKindConfig, "build decision request", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Decision{}, newError(ErrorKindTransport, "call mockserver decision endpoint", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Decision{}, newError(ErrorKindTransport, "read mockserver decision response", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Decision{}, newStatusError(ErrorKindServer, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var envelope decisionEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return Decision{}, newError(ErrorKindDecode, "decode mockserver decision response", err)
	}
	if envelope.Code != 0 {
		if envelope.Message == "" {
			envelope.Message = "mockserver decision failed"
		}
		return Decision{}, newStatusError(ErrorKindServer, resp.StatusCode, envelope.Message)
	}
	if envelope.Data.Decision.Kind == "" {
		return Decision{}, newError(ErrorKindDecode, "mockserver decision response missing decision kind", nil)
	}
	return envelope.Data.Decision, nil
}

func (c *Client) namespaceFor(namespace string) string {
	if strings.TrimSpace(namespace) != "" {
		return normalizedNamespace(namespace)
	}
	return c.namespace
}

func (c *Client) NamespaceFor(namespace string) string {
	return c.namespaceFor(namespace)
}

func isValidNamespace(namespace string) bool {
	if namespace == "" {
		return false
	}
	for _, item := range namespace {
		if item >= 'a' && item <= 'z' || item >= 'A' && item <= 'Z' || item >= '0' && item <= '9' || item == '_' || item == '-' {
			continue
		}
		return false
	}
	return true
}

type decisionRequest struct {
	Event Event `json:"event"`
}

type decisionEnvelope struct {
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
	Data    struct {
		Decision Decision `json:"decision"`
	} `json:"data"`
}
