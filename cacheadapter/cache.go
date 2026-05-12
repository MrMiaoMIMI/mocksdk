package cacheadapter

import (
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
