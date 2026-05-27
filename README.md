# mocksdk

`mocksdk` is the lightweight Go SDK used by mockinject-style adapters to ask
mockserver for a decision without knowing mockserver internals.

The SDK intentionally has a small runtime surface:

- supports Go 1.17+
- uses only the Go standard library
- keeps protocol forwarding in the caller; the SDK only builds events, calls
  mockserver's decision endpoint, and returns response/forward decisions

## Install

```sh
go get github.com/MrMiaoMIMI/mocksdk
```

## Basic Usage

```go
client, err := mocksdk.NewClient(mocksdk.Config{
	MockServerURL: "http://127.0.0.1:8080",
	APIPrefix:     "/tenant-a",
	Namespace:     "default",
})
if err != nil {
	return err
}

decision, err := client.Decide(ctx, mocksdk.Event{
	Protocol:  "http",
	Operation: "request",
	Namespace: "default",
	Request: mocksdk.EventRequest{
		"method": "GET",
		"path":   "/api/users/1",
	},
})
```

## Scenario Context

Agent scenarios are carried through `context.Context`, not a process-wide
environment variable:

```go
ctx = mocksdk.WithScenarioID(ctx, "scn_checkout_timeout")
```

Valid scenario ids start with `scn_`, are 8 to 128 characters long, and only
contain letters, digits, underscores, and hyphens. `Client.Decide` copies the
context scenario id into `event.Meta.ScenarioID` when the event does not already
set one, so request-level callers can decide when a business request should use
a scenario overlay.

When `event.Meta.ScenarioID` is present, MockServer evaluates that scenario
overlay first. A missing, inactive, expired, or unmatched scenario uses the
event namespace fallback policy; it does not fall through to published rulesets.

Configuration can also be provided through environment variables:

- `MOCKSERVER_HOST`: mockserver base URL, for example
  `http://127.0.0.1:8080`. It may also include the deployment prefix, for
  example `http://127.0.0.1:8080/tenant-a`.
- `MOCKSERVER_API_PREFIX`: optional mockserver deployment path prefix, for
  example `/tenant-a`. This is appended before `/mockserver/api/v1/sdk/decision`
  unless `MOCKSERVER_HOST` already ends with the same prefix.
- `MOCKSERVER_NAMESPACE_ID`: namespace id, defaults to `default`
- `MOCKSERVER_TIMEOUT_MS`: optional request timeout in milliseconds

When the mockserver backend is started with `MOCKSERVER_API_PREFIX=/tenant-a`,
configure the SDK with either `MOCKSERVER_HOST=http://127.0.0.1:8080/tenant-a`
or `MOCKSERVER_HOST=http://127.0.0.1:8080` plus
`MOCKSERVER_API_PREFIX=/tenant-a`.

## Protocol Adapters

The SDK includes small standard-library adapters:

- `httpadapter`: normalizes `net/http` requests and applies response decisions
- `cacheadapter`: builds cache operation events
- `spexadapter`: builds SPEX events and decodes SPEX response decisions

Response decision payloads are stored as a single `json.RawMessage` in
`mocksdk.ResponseDecision.Payload`. Callers should use the protocol adapter
helpers instead of reading protocol fields themselves:

```go
payload, err := httpadapter.PayloadFromDecision(decision)
if err != nil {
	return err
}

fmt.Println(payload.Status, payload.Headers, string(payload.Body))
```

Available typed payload helpers:

- `httpadapter.PayloadFromDecision`: decodes `HTTPPayload` with `status`,
  `headers`, and raw `body`
- `spexadapter.PayloadFromDecision`: decodes `SPEXPayload` with `code` and raw
  JSON `resp`
- `cacheadapter.PayloadFromDecision`: decodes `CachePayload` with `hit` and
  raw `value`
