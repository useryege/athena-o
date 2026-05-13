package sourcecode

import (
	"sort"
	"strings"
)

type BlacklistReport struct {
	HasBlacklistFields bool
	BlacklistFields    []string
}

func NormalizeBlacklistField(field string) string {
	return strings.TrimSpace(field)
}

func NormalizeBlacklistFields(fields []string) []string {
	seen := make(map[string]struct{}, len(fields))
	normalized := make([]string, 0, len(fields))
	for _, field := range fields {
		field = NormalizeBlacklistField(field)
		if field == "" {
			continue
		}
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		normalized = append(normalized, field)
	}
	sort.Strings(normalized)
	return normalized
}
