package event

import (
	_ "embed"
	"encoding/json"
)

type Entry struct {
	ActionCode     string   `json:"actionCode"`
	ModuleCode     string   `json:"moduleCode"`
	Transport      string   `json:"transport"`
	Entry          string   `json:"entry"`
	HTTPMethod     string   `json:"httpMethod"`
	Route          string   `json:"route"`
	Source         string   `json:"source"`
	Symbol         string   `json:"symbol"`
	ResourceType   string   `json:"resourceType"`
	SuccessPolicy  string   `json:"successPolicy"`
	AllowedDetails []string `json:"allowedDetails"`
	Notes          string   `json:"notes,omitempty"`
	CaptureWhen    string   `json:"captureWhen"`
}

//go:embed catalog.json
var catalogJSON []byte

var entries = func() []Entry {
	var c struct {
		Events []Entry `json:"events"`
	}
	if err := json.Unmarshal(catalogJSON, &c); err != nil {
		panic("invalid compiled operation log catalog")
	}
	return c.Events
}()

// Catalog returns independent copies of all approved capture positions.
func Catalog() []Entry {
	out := append([]Entry(nil), entries...)
	for i := range out {
		out[i].AllowedDetails = append([]string{}, out[i].AllowedDetails...)
	}
	return out
}

// Lookup returns an exact capture position, including its failure-only policy.
func Lookup(entry string) (Entry, bool) {
	for _, e := range Catalog() {
		if e.Entry == entry {
			return e, true
		}
	}
	return Entry{}, false
}

// Action returns the action's module and the union of its approved capture fields.
// BeginEntry should be used when a capture position has a narrower allowlist.
func Action(code string) (Entry, bool) {
	var out Entry
	for _, e := range entries {
		if e.ActionCode != code {
			continue
		}
		if out.ActionCode == "" {
			out = e
			out.AllowedDetails = nil
		}
		for _, k := range e.AllowedDetails {
			if !contains(out.AllowedDetails, k) {
				out.AllowedDetails = append(out.AllowedDetails, k)
			}
		}
	}
	return out, out.ActionCode != ""
}
func contains(a []string, s string) bool {
	for _, v := range a {
		if v == s {
			return true
		}
	}
	return false
}
