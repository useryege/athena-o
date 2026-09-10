package types

type MarketRef struct {
	Evidence
	ID, Title, URL, ConditionID, PositionID, Outcome string
}
type ComboLeg struct {
	PositionID string
	Market     MarketRef
}
type TradeMetadata struct {
	Market       MarketRef
	LegsEvidence Evidence
	Legs         []ComboLeg
	Relationship string
}
