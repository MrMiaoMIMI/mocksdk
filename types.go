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
	TraceID    string                 `json:"trace_id,omitempty"`
	Source     string                 `json:"source,omitempty"`
	ScenarioID string                 `json:"scenario_id,omitempty"`
	Extra      map[string]interface{} `json:"extra,omitempty"`
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
	Protocol string            `json:"protocol,omitempty"`
	Trace    DecisionTrace     `json:"trace"`
	Response *ResponseDecision `json:"response,omitempty"`
	Forward  *ForwardDecision  `json:"forward,omitempty"`
	Meta     DecisionMeta      `json:"meta,omitempty"`
}

type DecisionTrace struct {
	RulesetID      string `json:"ruleset_id,omitempty"`
	RuleID         string `json:"rule_id,omitempty"`
	SnapshotID     string `json:"snapshot_id,omitempty"`
	FallbackReason string `json:"fallback_reason,omitempty"`
}

type ResponseDecision struct {
	Protocol string          `json:"protocol,omitempty"`
	Payload  json.RawMessage `json:"payload,omitempty"`
}

func (r *ResponseDecision) DecodePayload(target interface{}) error {
	if r == nil {
		return newError(ErrorKindConfig, "response decision is nil", nil)
	}
	if len(r.Payload) == 0 {
		return newError(ErrorKindConfig, "response decision payload is empty", nil)
	}
	if err := json.Unmarshal(r.Payload, target); err != nil {
		return newError(ErrorKindConfig, "decode response decision payload", err)
	}
	return nil
}

type ForwardDecision struct {
	TimeoutMS int `json:"timeout_ms,omitempty"`
}

type DecisionMeta struct {
	TraceID    string `json:"trace_id,omitempty"`
	ScenarioID string `json:"scenario_id,omitempty"`
}
