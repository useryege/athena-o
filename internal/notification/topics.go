package notification

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	notificationTopicToken       = "token"
	notificationTopicPolyMover   = "poly-mover"
	notificationTopicPolyKickoff = "poly-kickoff"
	notificationTopicWorm        = "worm"

	maxTopicKeyLength   = 64
	maxTopicTitleLength = 128
)

const (
	NotificationTopicToken       = notificationTopicToken
	NotificationTopicPolyMover   = notificationTopicPolyMover
	NotificationTopicPolyKickoff = notificationTopicPolyKickoff
	NotificationTopicWorm        = notificationTopicWorm
)

type TopicConfig struct {
	Key   string
	Title string
}

func DefaultTopicConfigs() []TopicConfig {
	return []TopicConfig{
		{Key: notificationTopicToken, Title: "[TOKEN] 代币通知"},
		{Key: notificationTopicPolyMover, Title: "[POLY] 市场异动"},
		{Key: notificationTopicPolyKickoff, Title: "[POLY] 开赛通知"},
		{Key: notificationTopicWorm, Title: "[WORM] 比赛通知"},
	}
}

func NormalizeTopicKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func validateTopicKey(key string) error {
	if key == "" {
		return fmt.Errorf("notification topic key is required")
	}
	if utf8.RuneCountInString(key) > maxTopicKeyLength {
		return fmt.Errorf("notification topic key must be at most %d characters", maxTopicKeyLength)
	}
	if strings.ContainsAny(key, " \t\r\n") {
		return fmt.Errorf("notification topic key %q must not contain whitespace", key)
	}
	return nil
}

func normalizeTopicKeyValue(value string) (string, error) {
	key := NormalizeTopicKey(value)
	if err := validateTopicKey(key); err != nil {
		return "", err
	}
	return key, nil
}

func normalizeTopicConfig(config TopicConfig) (TopicConfig, error) {
	key, err := normalizeTopicKeyValue(config.Key)
	if err != nil {
		return TopicConfig{}, err
	}
	config.Key = key
	config.Title = strings.TrimSpace(config.Title)
	if config.Title == "" {
		return TopicConfig{}, fmt.Errorf("telegram topic title is required for topic %s", config.Key)
	}
	if utf8.RuneCountInString(config.Title) > maxTopicTitleLength {
		return TopicConfig{}, fmt.Errorf("telegram topic title for topic %s must be at most %d characters", config.Key, maxTopicTitleLength)
	}
	return config, nil
}

func normalizeTopicConfigs(configs []TopicConfig) ([]TopicConfig, error) {
	if len(configs) == 0 {
		configs = DefaultTopicConfigs()
	}
	normalized := make([]TopicConfig, 0, len(configs))
	seen := map[string]struct{}{}
	for _, config := range configs {
		item, err := normalizeTopicConfig(config)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[item.Key]; ok {
			return nil, fmt.Errorf("notification topic key %q is duplicated", item.Key)
		}
		seen[item.Key] = struct{}{}
		normalized = append(normalized, item)
	}
	return normalized, nil
}
