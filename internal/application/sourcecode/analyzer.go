package sourcecode

import "time"

type Analyzer interface {
	AnalyzeSourceCode(sourceCode string, blacklistFields []string) BlacklistReport
}

var _ Analyzer = &analyzerImpl{}

type analyzerImpl struct{}

func NewAnalyzer() Analyzer {
	return &analyzerImpl{}
}

func (a *analyzerImpl) AnalyzeSourceCode(sourceCode string, blacklistFields []string) BlacklistReport {
	now := time.Now()
	blacklist := blacklistSet(blacklistFields)
	if len(blacklist) == 0 || sourceCode == "" {
		return BlacklistReport{ResolvedAt: now}
	}

	matched := make([]string, 0, len(blacklist))
	seen := make(map[string]struct{}, len(blacklist))
	for i := 0; i < len(sourceCode); {
		if !isSolidityIdentifierChar(sourceCode[i]) {
			i++
			continue
		}

		start := i
		for i < len(sourceCode) && isSolidityIdentifierChar(sourceCode[i]) {
			i++
		}
		field := sourceCode[start:i]
		if _, ok := blacklist[field]; !ok {
			continue
		}
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		matched = append(matched, field)
	}

	matched = NormalizeBlacklistFields(matched)
	return BlacklistReport{
		HasBlacklistFields: len(matched) > 0,
		BlacklistFields:    matched,
		ResolvedAt:         now,
	}
}

func blacklistSet(fields []string) map[string]struct{} {
	fields = NormalizeBlacklistFields(fields)
	blacklist := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		blacklist[field] = struct{}{}
	}
	return blacklist
}

func isSolidityIdentifierChar(value byte) bool {
	return value >= 'a' && value <= 'z' ||
		value >= 'A' && value <= 'Z' ||
		value >= '0' && value <= '9' ||
		value == '_' ||
		value == '$'
}
