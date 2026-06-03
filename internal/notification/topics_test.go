package notification

import "testing"

func TestDefaultTopicConfigs(t *testing.T) {
	configs := DefaultTopicConfigs()
	if len(configs) != 2 {
		t.Fatalf("configs = %#v, want token/poly", configs)
	}
	if configs[0].Key != NotificationTopicToken || configs[0].Title != "TOKEN" {
		t.Fatalf("first config = %#v, want token/TOKEN", configs[0])
	}
	if configs[1].Key != NotificationTopicPoly || configs[1].Title != "POLY" {
		t.Fatalf("second config = %#v, want poly/POLY", configs[1])
	}
}

func TestNormalizeTopicConfigsValidatesAndDeduplicates(t *testing.T) {
	configs, err := normalizeTopicConfigs([]TopicConfig{
		{Key: " Poly ", Title: " POLY Alerts "},
	})
	if err != nil {
		t.Fatalf("normalizeTopicConfigs: %v", err)
	}
	if len(configs) != 1 || configs[0].Key != "poly" || configs[0].Title != "POLY Alerts" {
		t.Fatalf("configs = %#v, want normalized key/title", configs)
	}

	_, err = normalizeTopicConfigs([]TopicConfig{
		{Key: "poly", Title: "POLY"},
		{Key: " POLY ", Title: "Duplicate"},
	})
	if err == nil {
		t.Fatalf("normalizeTopicConfigs duplicate error = nil, want error")
	}
}
