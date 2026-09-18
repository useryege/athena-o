package event

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func ptr[T any](v T) *T { return &v }
func validEvent() Event {
	return Event{SchemaVersion: 1, EventID: "00000000-0000-4000-8000-000000000001", OperationID: "00000000-0000-4000-8000-000000000002", RequestID: "00000000-0000-4000-8000-000000000003", ProducerID: "00000000-0000-4000-8000-000000000004", Phase: Finish, StartedAt: time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC), OccurredAt: time.Date(2026, 9, 18, 0, 0, 1, 0, time.UTC), Actor: Actor{Role: "UNKNOWN", Realm: "UNKNOWN", CredentialKind: "UNAUTHENTICATED"}, ActionCode: "account.access.update", ModuleCode: "account", Outcome: Unknown, Observation: FinishOnly, DurationMs: ptr(int64(0)), ResourcesComplete: true}
}
func TestCatalogCoverageAndApprovedFields(t *testing.T) {
	entries := Catalog()
	actions := map[string]bool{}
	for _, e := range entries {
		actions[e.ActionCode] = true
	}
	if len(entries) != 98 || len(actions) != 74 {
		t.Fatalf("entries=%d actions=%d", len(entries), len(actions))
	}
	b, err := os.ReadFile("../../../docs/superpowers/specs/2026-09-18-key-operation-logs/event-catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	var approved struct {
		Events []Entry `json:"events"`
	}
	if err = json.Unmarshal(b, &approved); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(entries, approved.Events) {
		t.Fatal("catalog diverges from approved fields")
	}
	entries[0].AllowedDetails[0] = "secret"
	if Catalog()[0].AllowedDetails[0] == "secret" {
		t.Fatal("catalog aliases mutable caller state")
	}
}
func TestRejectInvalidEvent(t *testing.T) {
	cases := map[string]func(*Event){
		"schema": func(e *Event) { e.SchemaVersion = 2 }, "zero uuid": func(e *Event) { e.EventID = "00000000-0000-0000-0000-000000000000" }, "noncanonical uuid": func(e *Event) { e.EventID = "00000000000040008000000000000001" },
		"unverified account": func(e *Event) { e.Actor.AccountID = ptr(e.OperationID) }, "unverified name": func(e *Event) { e.Actor.UsernameSnapshot = ptr("member") }, "false snapshot": func(e *Event) { e.Actor.IdentitySnapshotComplete = true },
		"role": func(e *Event) { e.Actor.Role = "ROOT" }, "realm": func(e *Event) { e.Actor.Realm = "root" }, "credential": func(e *Event) { e.Actor.CredentialKind = "COOKIE" },
		"module": func(e *Event) { e.ModuleCode = "other" }, "action": func(e *Event) { e.ActionCode = "arbitrary" }, "outcome": func(e *Event) { e.Outcome = "OK" }, "phase": func(e *Event) { e.Phase = "DONE" },
		"start result": func(e *Event) { e.Phase = Start; e.Observation = StartOnly; e.Outcome = Succeeded }, "missing duration": func(e *Event) { e.DurationMs = nil }, "negative duration": func(e *Event) { e.DurationMs = ptr(int64(-1)) },
		"timestamp": func(e *Event) { e.StartedAt = time.Time{} }, "nonUTC": func(e *Event) { e.OccurredAt = e.OccurredAt.In(time.FixedZone("offset", 3600)) },
		"forbidden detail": func(e *Event) { e.Details = Details{"bearer": String("secret")} }, "wrong action detail": func(e *Event) { e.Details = Details{"walletId": String("1")} },
		"number type": func(e *Event) {
			var v Value
			_ = json.Unmarshal([]byte("1"), &v)
			e.Details = Details{"expectedRevision": v}
		}, "overflow": func(e *Event) { e.Details = Details{"expectedRevision": String("18446744073709551616")} }, "leading zero": func(e *Event) { e.Details = Details{"expectedRevision": String("01")} },
		"null": func(e *Event) {
			var v Value
			_ = json.Unmarshal([]byte("null"), &v)
			e.Details = Details{"beforeAvailable": v}
		},
		"access extra": func(e *Event) {
			var v Value
			_ = json.Unmarshal([]byte(`{"loginEnabled":true,"apiKeyEnabled":false,"profitSharingEnabled":false,"revision":"1","moduleAccess":[],"bearer":"secret"}`), &v)
			e.Details = Details{"confirmedAccess": v}
		},
		"access missing": func(e *Event) {
			var v Value
			_ = json.Unmarshal([]byte(`{"revision":"1"}`), &v)
			e.Details = Details{"confirmedAccess": v}
		},
		"access enum": func(e *Event) {
			e.Details = Details{"confirmedAccess": AccessValue(Access{Revision: "1", ModuleAccess: []ModuleAccess{{"token", "root"}}})}
		},
		"resource type": func(e *Event) { e.Resources = []Resource{{Type: "secret", ID: "1"}} }, "resource length": func(e *Event) { e.Resources = make([]Resource, 101) },
		"bad account resource": func(e *Event) { e.Resources = []Resource{{Type: "account", ID: "arbitrary", Primary: true}} },
		"resource count":       func(e *Event) { e.ResourceCount = ptr(uint64(1)) }, "effect length": func(e *Event) { e.Effect = make([]string, 33) }, "effect prose": func(e *Event) { e.Effect = []string{"raw error text"} },
		"business id length": func(e *Event) { e.BusinessRequestID = ptr(strings.Repeat("x", 129)) }, "invalid utf8": func(e *Event) { e.BusinessRequestID = ptr(string([]byte{255})) },
		"http": func(e *Event) { e.HTTPStatus = ptr(999) }, "grpc": func(e *Event) { e.GRPCCode = ptr("MadeUp") }, "reason prose": func(e *Event) { e.ReasonCode = ptr("secret error body") },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			e := validEvent()
			mutate(&e)
			if Validate(e) == nil {
				t.Fatal("accepted invalid event")
			}
		})
	}
}
func TestCanonicalPresenceAndPrecision(t *testing.T) {
	e := validEvent()
	e.Details = Details{"expectedRevision": String("18446744073709551615"), "beforeAvailable": Bool(false)}
	b, err := Canonical(e)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte(`"beforeAvailable":false,"expectedRevision":"18446744073709551615"`)) {
		t.Fatalf("lost ordering or values: %s", b)
	}
	if !bytes.Contains(b, []byte(`"durationMs":"0"`)) {
		t.Fatal("duration must remain decimal string")
	}
	var parsed Event
	if err = json.Unmarshal(b, &parsed); err != nil {
		t.Fatal(err)
	}
	again, err := Canonical(parsed)
	if err != nil || !bytes.Equal(b, again) {
		t.Fatalf("unstable canonical: %v", err)
	}
	delete(e.Details, "beforeAvailable")
	absent, _ := Canonical(e)
	if bytes.Equal(absent, b) {
		t.Fatal("absent differs from false")
	}
}

func TestStrictDecodeAndNestedCanonicalOrdering(t *testing.T) {
	e := validEvent()
	e.Details = Details{"confirmedAccess": AccessValue(Access{Revision: "18446744073709551615", ModuleAccess: []ModuleAccess{{"token", "read_write"}, {"solana", "read"}}})}
	b, err := Canonical(e)
	if err != nil {
		t.Fatal(err)
	}
	for name, raw := range map[string][]byte{
		"duplicate":              bytes.Replace(b, []byte(`"schemaVersion":1`), []byte(`"schemaVersion":1,"schemaVersion":1`), 1),
		"unknown":                bytes.Replace(b, []byte(`"schemaVersion":1`), []byte(`"schemaVersion":1,"secret":"token"`), 1),
		"case alias":             bytes.Replace(b, []byte(`"schemaVersion"`), []byte(`"SchemaVersion"`), 1),
		"missing explicit false": bytes.Replace(b, []byte(`"resourcesComplete":true`), []byte(`"ignored":true`), 1),
		"trailing":               append(append([]byte{}, b...), []byte(` {}`)...),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Decode(raw); err == nil {
				t.Fatal("accepted malformed envelope")
			}
		})
	}
	var raw Value
	_ = json.Unmarshal([]byte(`{"revision":"18446744073709551615","moduleAccess":[{"access":"read_write","module":"token"},{"access":"read","module":"solana"}],"profitSharingEnabled":false,"apiKeyEnabled":false,"loginEnabled":false}`), &raw)
	e.Details["confirmedAccess"] = raw
	reordered, err := Canonical(e)
	if err != nil || !bytes.Equal(b, reordered) {
		t.Fatal("nested object order changed canonical payload", err)
	}
	_ = json.Unmarshal([]byte(`{"revision":"1","revision":"2","moduleAccess":[],"profitSharingEnabled":false,"apiKeyEnabled":false,"loginEnabled":false}`), &raw)
	e.Details["confirmedAccess"] = raw
	if Validate(e) == nil {
		t.Fatal("duplicate access key accepted")
	}
}
func TestDetailDomainEnumsAndUTF8(t *testing.T) {
	e := validEvent()
	e.ActionCode = "account.tier.update"
	for _, tier := range []string{"standard", "pro", "ACCOUNT_TIER_STANDARD", "ACCOUNT_TIER_PRO", "STANDARD", "PRO"} {
		e.Details = Details{"confirmedTier": String(tier)}
		if err := Validate(e); err != nil {
			t.Fatalf("source tier %s: %v", tier, err)
		}
	}
	for _, tier := range []string{"FREE", "ENTERPRISE", "arbitrary"} {
		e.Details = Details{"confirmedTier": String(tier)}
		if Validate(e) == nil {
			t.Fatalf("invented tier %s", tier)
		}
	}
	e.ActionCode = "profit_sharing.round.create"
	e.ModuleCode = "profit_sharing"
	e.Details = Details{"slug": String(string([]byte{255}))}
	if Validate(e) == nil {
		t.Fatal("invalid UTF8 replaced silently")
	}
	e = validEvent()
	e.Details = Details{"confirmedAccess": AccessValue(Access{Revision: "1", ModuleAccess: []ModuleAccess{{"solana", "read_write"}}})}
	if Validate(e) == nil {
		t.Fatal("read-only module granted write")
	}
}
func TestSizeAndResourceBounds(t *testing.T) {
	e := validEvent()
	b, err := Canonical(e)
	if err != nil {
		t.Fatal(err)
	}
	limit := append(append([]byte{}, b...), bytes.Repeat([]byte(" "), 32*1024-len(b))...)
	if _, err := Decode(limit); err != nil {
		t.Fatalf("at-boundary envelope rejected: %v", err)
	}
	if _, err := Decode(append(limit, ' ')); err == nil {
		t.Fatal("accepted envelope beyond 32 KiB")
	}
	e.Details = Details{"beforeAvailable": Value{raw: json.RawMessage(strings.Repeat(" ", 16*1024-4) + "true")}}
	if err := Validate(e); err != nil {
		t.Fatalf("at-boundary raw detail rejected: %v", err)
	}
	e.Details = Details{"beforeAvailable": Value{raw: json.RawMessage(strings.Repeat(" ", 16*1024-3) + "true")}}
	if Validate(e) == nil {
		t.Fatal("accepted excessive raw details")
	}
}

func TestTruncateAtUTF8Boundary(t *testing.T) {
	if got := TruncateUTF8("ab钱包", 6); got != "ab钱" {
		t.Fatalf("got %q", got)
	}
	if got := TruncateUTF8("钱包", 2); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestDomainResourceAndDetailIdentifiers(t *testing.T) {
	cases := []struct{ action, module, key, value string }{
		{"wallet.batch.create", "wallet", "walletIds", `["not-a-number"]`},
		{"worm.execution.create", "worm", "runId", `"not-a-uuid"`},
		{"trader_sync.subscription.create", "trader_sync", "subscriptionId", `"not-a-uuid"`},
		{"identity.login", "identity", "provider", `"opaqueBearer"`},
		{"wallet.batch.create", "wallet", "walletType", `"MADE_UP"`},
		{"account.avatar.upload", "account", "avatarKind", `"MADE_UP"`},
	}
	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			e := validEvent()
			e.ActionCode = tc.action
			e.ModuleCode = tc.module
			var v Value
			_ = json.Unmarshal([]byte(tc.value), &v)
			e.Details = Details{tc.key: v}
			if Validate(e) == nil {
				t.Fatal("accepted invalid source-domain value")
			}
		})
	}
	e := validEvent()
	e.Actor.Provider = ptr("opaqueBearer")
	if Validate(e) == nil {
		t.Fatal("unknown identity provider")
	}
	e = validEvent()
	e.Resources = []Resource{{Type: "wallet", ID: "uuid-not-wallet-number"}}
	if Validate(e) == nil {
		t.Fatal("invalid wallet resource")
	}
}
func TestValueCopiesDoNotAliasOnDecode(t *testing.T) {
	v := String("1")
	e := validEvent()
	e.Details = Details{"expectedRevision": v}
	_ = json.Unmarshal([]byte(`"2"`), &v)
	b, _ := Canonical(e)
	if !bytes.Contains(b, []byte(`"expectedRevision":"1"`)) {
		t.Fatal("value mutated stored detail")
	}
}

func TestPermissionPairRejectsAliasAndMissingFields(t *testing.T) {
	for _, pair := range []string{`{"Module":"token","access":"read"}`, `{"module":"token","Access":"read"}`, `{"module":"token","access":"read","extra":true}`, `{"module":"token"}`} {
		e := validEvent()
		var v Value
		_ = json.Unmarshal([]byte(`{"loginEnabled":false,"apiKeyEnabled":false,"profitSharingEnabled":false,"revision":"1","moduleAccess":[`+pair+`]}`), &v)
		e.Details = Details{"confirmedAccess": v}
		if Validate(e) == nil {
			t.Errorf("accepted pair %s", pair)
		}
	}
}
