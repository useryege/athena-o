package types

import "time"

type RenderedPart struct {
	Index, Total  int
	Text          string
	ActivityIDs   []int64
	PayloadDigest []byte
}

type SummaryPart struct {
	RenderedPart
	DeliveryID int64
}
type SummaryBatch struct {
	ID                 int64
	OwnerID            string
	BindingRevision    uint64
	ChatID             int64
	OldestAt, FrozenAt time.Time
	Parts              []SummaryPart
}

type SummaryProgress struct {
	Total   int64
	Counts  map[string]int64
	Success bool
}
