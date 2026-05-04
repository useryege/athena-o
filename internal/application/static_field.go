package application

import "time"

type StaticField string

const (
	StaticFieldName          StaticField = "name"
	StaticFieldSymbol        StaticField = "symbol"
	StaticFieldDecimals      StaticField = "decimals"
	StaticFieldSourceCode    StaticField = "source_code"
	StaticFieldSourceCodeABI StaticField = "source_code_abi"
)

type StaticFieldPolicy struct {
	Field StaticField
	// Required  bool
	BaseDelay time.Duration
	Source    string
}

var DefaultStaticFieldPolicies = map[StaticField]StaticFieldPolicy{
	StaticFieldName: {
		Field: StaticFieldName,
		// Required:  true,
		BaseDelay: 10 * time.Second,
		Source:    "rpc",
	},
	StaticFieldSymbol: {
		Field: StaticFieldSymbol,
		// Required:  true,
		BaseDelay: 10 * time.Second,
		Source:    "rpc",
	},
	StaticFieldDecimals: {
		Field: StaticFieldDecimals,
		// Required:  true,
		BaseDelay: 10 * time.Second,
		Source:    "rpc",
	},
	StaticFieldSourceCode: {
		Field: StaticFieldSourceCode,
		// Required:  false,
		BaseDelay: 5 * time.Minute,
		Source:    "explorer",
	},
	StaticFieldSourceCodeABI: {
		Field: StaticFieldSourceCodeABI,
		// Required:  false,
		BaseDelay: 5 * time.Minute,
		Source:    "explorer",
	},
}
