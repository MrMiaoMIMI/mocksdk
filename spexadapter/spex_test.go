package spexadapter

import (
	"context"
	"testing"

	"github.com/MrMiaoMIMI/mocksdk"
)

type testDecider struct {
	event mocksdk.Event
}

func (d *testDecider) Decide(ctx context.Context, event mocksdk.Event) (mocksdk.Decision, error) {
	_ = ctx
	d.event = event
	return mocksdk.Decision{
		Kind:    mocksdk.DecisionKindForward,
		Matched: false,
	}, nil
}

func TestEventBuildsSPEXEvent(t *testing.T) {
	event := Event("workspace", Request{
		Cmd:     " shop.GetOrder ",
		Req:     map[string]interface{}{"order_id": "1001"},
		Param:   " region=sg ",
		TraceID: " trace-spex ",
	})

	if event.Protocol != Protocol || event.Operation != "request" {
		t.Fatalf("unexpected protocol/operation: %+v", event)
	}
	if event.Namespace != "workspace" {
		t.Fatalf("unexpected namespace: %s", event.Namespace)
	}
	if event.Meta.TraceID != "trace-spex" {
		t.Fatalf("unexpected trace id: %s", event.Meta.TraceID)
	}
	if event.Request["cmd"] != "shop.GetOrder" {
		t.Fatalf("unexpected cmd: %#v", event.Request["cmd"])
	}
	req, ok := event.Request["req"].(map[string]interface{})
	if !ok || req["order_id"] != "1001" {
		t.Fatalf("unexpected req: %#v", event.Request["req"])
	}
	if event.Request["param"] != "region=sg" {
		t.Fatalf("unexpected param: %#v", event.Request["param"])
	}
}

func TestDecideUsesBuiltSPEXEvent(t *testing.T) {
	decider := &testDecider{}
	decision, err := Decide(context.Background(), decider, "", Request{Cmd: "service.Method"})
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if decision.Kind != mocksdk.DecisionKindForward {
		t.Fatalf("unexpected decision: %+v", decision)
	}
	if decider.event.Protocol != Protocol {
		t.Fatalf("unexpected protocol: %+v", decider.event)
	}
	if decider.event.Namespace != mocksdk.DefaultNamespace {
		t.Fatalf("unexpected namespace: %s", decider.event.Namespace)
	}
	if decider.event.Request["cmd"] != "service.Method" {
		t.Fatalf("unexpected cmd: %#v", decider.event.Request["cmd"])
	}
}
