package application

const (
	projectEventTypeCreated        int16 = 1
	projectEventTypeOpenSource     int16 = 2
	projectEventTypeAutoArchiveBIN int16 = 3
)

const (
	projectEventIdempotencyCreated        = "project_created"
	projectEventIdempotencyOpenSource     = "project_source_code_opened"
	projectEventIdempotencyAutoArchiveBIN = "project_auto_archived_by_bin_blacklist"
)
