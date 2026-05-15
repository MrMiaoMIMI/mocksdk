package cacheadapter

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/MrMiaoMIMI/mocksdk"
)

const Protocol = "cache"

type Request struct {
	Operation string
	Key       string
	TTLMS     int
	Value     interface{}
}

type CachePayload struct {
	Hit   bool            `json:"hit"`
	Value json.RawMessage `json:"value,omitempty"`
}

func Event(namespace string, request Request) mocksdk.Event {
	document := mocksdk.EventRequest{
		"operation": strings.ToLower(strings.TrimSpace(request.Operation)),
		"key":       request.Key,
	}
	if request.TTLMS > 0 {
		document["ttl_ms"] = request.TTLMS
	}
	if request.Value != nil {
		document["value"] = request.Value
	}
	return mocksdk.Event{
		Protocol:  Protocol,
		Operation: "request",
		Namespace: mocksdk.NormalizeNamespace(namespace),
		Request:   document,
	}
}

func PayloadFromDecision(decision mocksdk.Decision) (CachePayload, error) {
	if decision.Kind != mocksdk.DecisionKindResponse || decision.Response == nil {
		return CachePayload{}, &mocksdk.Error{Kind: mocksdk.ErrorKindConfig, Message: "decision is not a response decision"}
	}
	if decision.Response.Protocol != "" && !strings.EqualFold(decision.Response.Protocol, Protocol) {
		return CachePayload{}, &mocksdk.Error{Kind: mocksdk.ErrorKindConfig, Message: "response decision protocol is not cache"}
	}
	payload := CachePayload{Hit: true}
	if len(decision.Response.Payload) == 0 || bytes.Equal(decision.Response.Payload, []byte("null")) {
		return payload, nil
	}
	var raw struct {
		Hit   *bool           `json:"hit"`
		Value json.RawMessage `json:"value"`
	}
	if err := decision.Response.DecodePayload(&raw); err != nil {
		return CachePayload{}, err
	}
	if raw.Hit != nil {
		payload.Hit = *raw.Hit
	}
	payload.Value = raw.Value
	return payload, nil
}
