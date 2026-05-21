package application

const (
	projectEventTypeCreated           int16 = 1
	projectEventTypeOpenSource        int16 = 2
	projectEventTypeAutoArchiveBIN    int16 = 3
	projectEventTypeAutoArchivePolicy int16 = 4
	projectEventTypePolicyMatchAudit  int16 = 5
)

const (
	projectEventIdempotencyCreated        = "project_created"
	projectEventIdempotencyOpenSource     = "project_source_code_opened"
	projectEventIdempotencyAutoArchiveBIN = "project_auto_archived_by_bin_blacklist"
)

func projectEventIdempotencyAutoArchivePolicy(rule string) string {
	return "project_auto_archived_by_policy:" + rule
}

func projectEventIdempotencyPolicyMatch(rule string) string {
	return "project_policy_matched:" + rule
}
