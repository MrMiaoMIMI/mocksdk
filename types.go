package mocksdk

import "encoding/json"

type Event struct {
	Protocol  string       `json:"protocol"`
	Operation string       `json:"operation,omitempty"`
	Namespace string       `json:"namespace"`
	Request   EventRequest `json:"request"`
	Meta      EventMeta    `json:"meta,omitempty"`
}

type EventRequest map[string]interface{}

type EventMeta struct {
	TraceID string                 `json:"trace_id,omitempty"`
	Source  string                 `json:"source,omitempty"`
	Extra   map[string]interface{} `json:"extra,omitempty"`
}

const DefaultNamespace = "default"

type DecisionKind string

const (
	DecisionKindResponse DecisionKind = "response"
	DecisionKindForward  DecisionKind = "forward"
)

type Decision struct {
	Kind     DecisionKind      `json:"kind"`
	Matched  bool              `json:"matched"`
	Fallback bool              `json:"fallback,omitempty"`
	Trace    DecisionTrace     `json:"trace"`
	Response *ResponseDecision `json:"response,omitempty"`
	Forward  *ForwardDecision  `json:"forward,omitempty"`
	Meta     DecisionMeta      `json:"meta,omitempty"`
}

type DecisionTrace struct {
	RulesetID      string `json:"ruleset_id,omitempty"`
	RuleID         string `json:"rule_id,omitempty"`
	FallbackReason string `json:"fallback_reason,omitempty"`
}

type ResponseDecision struct {
	Status  int                 `json:"status"`
	Headers map[string][]string `json:"headers,omitempty"`
	Body    json.RawMessage     `json:"body,omitempty"`
}

type ForwardDecision struct {
	TimeoutMS int `json:"timeout_ms,omitempty"`
}

type DecisionMeta struct {
	TraceID string `json:"trace_id,omitempty"`
}
