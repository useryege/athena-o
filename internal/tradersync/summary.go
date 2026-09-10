package tradersync

import (
	"github.com/useryege/athena/internal/tradersync/activity"
	tm "github.com/useryege/athena/internal/tradersync/types"
	"time"
)

func SummaryWindow(oldest time.Time, previousStart *time.Time) (time.Time, time.Time) {
	earliest := oldest
	if previousStart != nil && previousStart.Add(time.Minute).After(earliest) {
		earliest = previousStart.Add(time.Minute)
	}
	return earliest, oldest.Add(time.Minute)
}
func RenderSummary(items []tm.Activity, batchID string, siteURL string) ([]tm.RenderedPart, error) {
	return activity.RenderSummary(items, batchID, siteURL)
}
