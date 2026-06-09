package notification

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	NotificationTopicLabelToken       = "[TOKEN] 代币通知"
	NotificationTopicLabelPolyMover   = "[POLY] 市场异动"
	NotificationTopicLabelPolyKickoff = "[POLY] 开赛通知"
	NotificationTopicLabelWorm        = "[WORM] 比赛通知"

	maxTopicLabelLength = 128
)

func normalizeTopicLabel(value string) (string, error) {
	label := strings.TrimSpace(value)
	if label == "" {
		return "", fmt.Errorf("notification topic label is required")
	}
	if utf8.RuneCountInString(label) > maxTopicLabelLength {
		return "", fmt.Errorf("notification topic label must be at most %d characters", maxTopicLabelLength)
	}
	return label, nil
}
