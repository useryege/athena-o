package sourcecode

import (
	"reflect"
	"testing"
)

func TestNormalizeBlacklistFields(t *testing.T) {
	fields := NormalizeBlacklistFields([]string{" owner ", "blacklist", "owner", ""})
	if !reflect.DeepEqual(fields, []string{"blacklist", "owner"}) {
		t.Fatalf("fields = %v, want [blacklist owner]", fields)
	}
}

func TestAnalyzerMatchesIdentifierBoundaries(t *testing.T) {
	analyzer := NewAnalyzer()
	report := analyzer.AnalyzeSourceCode(`
contract Token {
    address ownerOfToken;
    address public owner;
    mapping(address => bool) private blacklist;
}
`, []string{"owner", "blacklist"})

	if !report.HasBlacklistFields {
		t.Fatal("report HasBlacklistFields = false, want true")
	}
	if !reflect.DeepEqual(report.BlacklistFields, []string{"blacklist", "owner"}) {
		t.Fatalf("report BlacklistFields = %v, want [blacklist owner]", report.BlacklistFields)
	}
}

func TestAnalyzerDoesNotMatchIdentifierSubstrings(t *testing.T) {
	analyzer := NewAnalyzer()
	report := analyzer.AnalyzeSourceCode(`
contract Token {
    function ownerOf(address account) external view returns (uint256) {}
}
`, []string{"owner"})

	if report.HasBlacklistFields {
		t.Fatalf("report HasBlacklistFields = true, want false: %v", report.BlacklistFields)
	}
	if len(report.BlacklistFields) != 0 {
		t.Fatalf("report BlacklistFields = %v, want empty", report.BlacklistFields)
	}
}
