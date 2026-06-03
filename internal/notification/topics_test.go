package notification

import "testing"

func TestDefaultTopicConfigs(t *testing.T) {
	configs := DefaultTopicConfigs()
	if len(configs) != 3 {
		t.Fatalf("configs = %#v, want token/poly-mover/poly-kickoff", configs)
	}
	if configs[0].Key != NotificationTopicToken || configs[0].Title != "[TOKEN] 代币通知" {
		t.Fatalf("first config = %#v, want token title", configs[0])
	}
	if configs[1].Key != NotificationTopicPolyMover || configs[1].Title != "[POLY] 市场异动" {
		t.Fatalf("second config = %#v, want poly mover title", configs[1])
	}
	if configs[2].Key != NotificationTopicPolyKickoff || configs[2].Title != "[POLY] 开赛通知" {
		t.Fatalf("third config = %#v, want poly kickoff title", configs[2])
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
