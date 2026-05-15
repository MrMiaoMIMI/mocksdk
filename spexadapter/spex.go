package spexadapter

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/MrMiaoMIMI/mocksdk"
)

const Protocol = "spex" // SPEX is a self-developed RPC framework

type Request struct {
	Cmd     string
	Req     interface{}
	Param   string
	TraceID string
}

type Response struct {
	Code int
	Resp json.RawMessage
}

type SPEXPayload struct {
	Code int             `json:"code,omitempty"`
	Resp json.RawMessage `json:"resp,omitempty"`
}

func Event(namespace string, request Request) mocksdk.Event {
	document := mocksdk.EventRequest{
		"cmd": strings.TrimSpace(request.Cmd),
	}
	if request.Req != nil {
		document["req"] = request.Req
	}
	if strings.TrimSpace(request.Param) != "" {
		document["param"] = strings.TrimSpace(request.Param)
	}
	return mocksdk.Event{
		Protocol:  Protocol,
		Operation: "request",
		Namespace: mocksdk.NormalizeNamespace(namespace),
		Request:   document,
		Meta: mocksdk.EventMeta{
			TraceID: strings.TrimSpace(request.TraceID),
		},
	}
}

func Decide(ctx context.Context, client mocksdk.Decider, namespace string, request Request) (mocksdk.Decision, error) {
	return client.Decide(ctx, Event(namespace, request))
}

func ResponseFromDecision(decision mocksdk.Decision) (Response, error) {
	payload, err := PayloadFromDecision(decision)
	if err != nil {
		return Response{}, err
	}
	return Response{Code: payload.Code, Resp: payload.Resp}, nil
}

func PayloadFromDecision(decision mocksdk.Decision) (SPEXPayload, error) {
	if decision.Kind != mocksdk.DecisionKindResponse || decision.Response == nil {
		return SPEXPayload{}, &mocksdk.Error{Kind: mocksdk.ErrorKindConfig, Message: "decision is not a response decision"}
	}
	if decision.Response.Protocol != "" && !strings.EqualFold(decision.Response.Protocol, Protocol) {
		return SPEXPayload{}, &mocksdk.Error{Kind: mocksdk.ErrorKindConfig, Message: "response decision protocol is not spex"}
	}
	if len(decision.Response.Payload) == 0 || string(decision.Response.Payload) == "null" {
		return SPEXPayload{}, nil
	}
	var raw struct {
		Code int             `json:"code,omitempty"`
		Resp json.RawMessage `json:"resp,omitempty"`
	}
	if err := json.Unmarshal(decision.Response.Payload, &raw); err != nil {
		return SPEXPayload{}, &mocksdk.Error{Kind: mocksdk.ErrorKindConfig, Message: "decode spex response payload", Err: err}
	}
	resp, err := decodeJSONResp(raw.Resp)
	if err != nil {
		return SPEXPayload{}, err
	}
	return SPEXPayload{Code: raw.Code, Resp: resp}, nil
}

func decodeJSONResp(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		resp := json.RawMessage(text)
		if !json.Valid(resp) {
			return nil, &mocksdk.Error{Kind: mocksdk.ErrorKindConfig, Message: "spex response resp must be valid JSON"}
		}
		return append(json.RawMessage(nil), resp...), nil
	}
	if !json.Valid(raw) {
		return nil, &mocksdk.Error{Kind: mocksdk.ErrorKindConfig, Message: "spex response resp must be valid JSON"}
	}
	return append(json.RawMessage(nil), raw...), nil
}
