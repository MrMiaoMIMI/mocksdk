package mocksdk

import (
	"context"
	"fmt"
	"strings"
)

const (
	ScenarioIDPrefix    = "scn_"
	MinScenarioIDLength = 8
	MaxScenarioIDLength = 128
)

type scenarioContextKey struct{}

func WithScenarioID(ctx context.Context, scenarioID string) context.Context {
	return context.WithValue(ctx, scenarioContextKey{}, strings.TrimSpace(scenarioID))
}

func ScenarioIDFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	value, ok := ctx.Value(scenarioContextKey{}).(string)
	value = strings.TrimSpace(value)
	return value, ok && value != ""
}

func ValidateScenarioID(scenarioID string) error {
	if scenarioID == "" {
		return nil
	}
	if len(scenarioID) < MinScenarioIDLength || len(scenarioID) > MaxScenarioIDLength {
		return fmt.Errorf("scenario id length must be between %d and %d characters", MinScenarioIDLength, MaxScenarioIDLength)
	}
	if !strings.HasPrefix(scenarioID, ScenarioIDPrefix) {
		return fmt.Errorf("scenario id must start with %q", ScenarioIDPrefix)
	}
	for _, item := range scenarioID {
		if item >= 'a' && item <= 'z' || item >= 'A' && item <= 'Z' || item >= '0' && item <= '9' || item == '_' || item == '-' {
			continue
		}
		return fmt.Errorf("scenario id may only contain letters, numbers, underscores, and hyphens")
	}
	return nil
}
