package sportshistory

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
)

const (
	sportsHistoryPriceHistoryBatchLimit = 20
	sportsHistoryPriceHistoryFidelity   = 1
)

var (
	sportsHistoryJSONObject = json.RawMessage("{}")
	sportsHistoryJSONArray  = json.RawMessage("[]")
)

func ptrBool(value bool) *bool {
	return &value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func boolValue(value *bool) bool {
	return value != nil && *value
}

func float64Value(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func timePtrValue(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return value.UTC()
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func uniqueNonEmptyStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		next := strings.TrimSpace(value)
		if next == "" {
			continue
		}
		if _, ok := seen[next]; ok {
			continue
		}
		seen[next] = struct{}{}
		out = append(out, next)
	}
	return out
}

func sportsHistoryTeamsFromRaw(raw json.RawMessage) []*v1alpha1.SportsTeamItem {
	type gammaTeam struct {
		Name         *string `json:"name,omitempty"`
		Logo         *string `json:"logo,omitempty"`
		Abbreviation *string `json:"abbreviation,omitempty"`
		Alias        *string `json:"alias,omitempty"`
	}
	var teams []gammaTeam
	if err := json.Unmarshal(raw, &teams); err != nil {
		return nil
	}
	items := make([]*v1alpha1.SportsTeamItem, 0, len(teams))
	for _, team := range teams {
		item := &v1alpha1.SportsTeamItem{
			Name:         strings.TrimSpace(stringValue(team.Name)),
			Logo:         strings.TrimSpace(stringValue(team.Logo)),
			Abbreviation: strings.TrimSpace(stringValue(team.Abbreviation)),
			Alias:        strings.TrimSpace(stringValue(team.Alias)),
		}
		if firstNonEmpty(item.Name, item.Abbreviation, item.Alias) == "" && item.Logo == "" {
			continue
		}
		items = append(items, item)
	}
	return items
}

func parseSportsHistoryStringList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err == nil {
		return items
	}
	return strings.Split(raw, ",")
}

func marshalArrayOrFallback(value any) json.RawMessage {
	data := marshalOrFallback(value, sportsHistoryJSONArray)
	if jsonType(data) != "array" {
		return sportsHistoryJSONArray
	}
	return data
}

func marshalOrFallback(value any, fallback json.RawMessage) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil || !json.Valid(data) {
		return fallback
	}
	return data
}

func rawObjectOrMarshal(raw json.RawMessage, value any) json.RawMessage {
	if jsonType(raw) == "object" {
		return raw
	}
	data := marshalOrFallback(value, sportsHistoryJSONObject)
	if jsonType(data) != "object" {
		return sportsHistoryJSONObject
	}
	return data
}

func jsonType(value json.RawMessage) string {
	var decoded any
	if err := json.Unmarshal(value, &decoded); err != nil {
		return ""
	}
	switch decoded.(type) {
	case map[string]any:
		return "object"
	case []any:
		return "array"
	default:
		return ""
	}
}
