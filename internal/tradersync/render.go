package tradersync

import "github.com/useryege/athena/internal/tradersync/activity"
import tm "github.com/useryege/athena/internal/tradersync/types"

func ClassifyActivity(windowCount int64) string { return activity.Classify(windowCount) }

func RenderActivity(a tm.Activity, siteURL string) (string, error) {
	return activity.Render(a, siteURL)
}
