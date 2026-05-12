package cacheadapter

import "testing"

func TestEventBuildsGenericEvent(t *testing.T) {
	event := Event("workspace", Request{
		Operation: " GET ",
		Key:       "user:123",
		TTLMS:     3000,
		Value: map[string]interface{}{
			"name": "demo",
		},
	})

	if event.Protocol != Protocol || event.Operation != "request" {
		t.Fatalf("unexpected protocol/operation: %+v", event)
	}
	if event.Namespace != "workspace" {
		t.Fatalf("unexpected namespace: %s", event.Namespace)
	}
	if event.Request["operation"] != "get" {
		t.Fatalf("unexpected operation: %#v", event.Request["operation"])
	}
	if event.Request["key"] != "user:123" {
		t.Fatalf("unexpected key: %#v", event.Request["key"])
	}
	if event.Request["ttl_ms"] != 3000 {
		t.Fatalf("unexpected ttl_ms: %#v", event.Request["ttl_ms"])
	}
	value, ok := event.Request["value"].(map[string]interface{})
	if !ok || value["name"] != "demo" {
		t.Fatalf("unexpected value: %#v", event.Request["value"])
	}
}
