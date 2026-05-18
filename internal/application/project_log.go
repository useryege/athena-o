package application

const (
	projectEventTypeCreated    int16 = 1
	projectEventTypeOpenSource int16 = 2
)

const (
	projectEventIdempotencyCreated    = "project_created"
	projectEventIdempotencyOpenSource = "project_source_code_opened"
)
