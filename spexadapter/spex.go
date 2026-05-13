package spexadapter

import (
	"context"
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
