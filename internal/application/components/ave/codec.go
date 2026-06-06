package ave

import (
	"time"

	appstore "github.com/useryege/athena/internal/application/store"
	utilave "github.com/useryege/athena/util/ave"
)

func DetailFromResponse(resp *utilave.TokenDetailResponse, fetchedAt time.Time) (*appstore.ProjectAveDetail, error) {
	return appstore.ProjectAveDetailFromAveResponse(resp, fetchedAt)
}
