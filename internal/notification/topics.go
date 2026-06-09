package notification

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
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
