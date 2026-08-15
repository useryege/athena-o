package managedoo

import (
	"encoding/json"
	"net/url"
	"strings"
	"time"
)

const (
	polymarketEventBaseURL  = "https://polymarket.com/event/"
	polymarketMarketBaseURL = "https://polymarket.com/market/"
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

func (s *Service) polymarketNotificationLink(link string) string {
	if s == nil {
		return link
	}
	return polymarketLinkWithInviteCode(link, s.notificationInviteCode)
}

func polymarketLinkWithInviteCode(link, inviteCode string) string {
	trimmedLink := strings.TrimSpace(link)
	inviteCode = strings.TrimSpace(inviteCode)
	if trimmedLink == "" || inviteCode == "" {
		return link
	}
	parsed, err := url.Parse(trimmedLink)
	if err != nil {
		return link
	}
	query := parsed.Query()
	query.Set("r", inviteCode)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func polymarketEventMarketOrMarketLink(eventSlug, marketSlug string) string {
	eventSlug = strings.Trim(strings.TrimSpace(eventSlug), "/")
	marketSlug = strings.Trim(strings.TrimSpace(marketSlug), "/")
	if marketSlug == "" {
		return ""
	}
	if eventSlug != "" {
		return polymarketEventBaseURL + eventSlug + "/" + marketSlug
	}
	return polymarketMarketBaseURL + marketSlug
}

func truncateRunes(value string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes])
}

var (
	managedOOJSONObject = json.RawMessage(`{}`)
	managedOOJSONArray  = json.RawMessage(`[]`)
)

func marshalArrayOrFallback(value any) json.RawMessage {
	data := marshalOrFallback(value, managedOOJSONArray)
	if jsonType(data) != "array" {
		return managedOOJSONArray
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
	data := marshalOrFallback(value, managedOOJSONObject)
	if jsonType(data) != "object" {
		return managedOOJSONObject
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
