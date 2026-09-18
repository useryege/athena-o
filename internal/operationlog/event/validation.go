package event

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const MaxEventBytes = 32 * 1024
const MaxDetailsBytes = 16 * 1024

var ErrInvalid = errors.New("invalid operation log event")
var codePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
var idPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:/-]*$`)
var walletAddressPattern = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
var codeHashPattern = regexp.MustCompile(`^0x[0-9a-f]{64}$`)
var decimalPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)$`)

func invalid(field string) error { return fmt.Errorf("%w: %s", ErrInvalid, field) }
func canonicalUUID(s string) bool {
	id, err := uuid.Parse(s)
	return err == nil && id != uuid.Nil && id.String() == s
}
func decimal(s string) bool {
	_, err := strconv.ParseUint(s, 10, 64)
	return err == nil && decimalPattern.MatchString(s)
}
func bounded(s string, n int) bool   { return len(s) <= n && utf8.ValidString(s) }
func code(s string) bool             { return bounded(s, 64) && codePattern.MatchString(s) }
func identifier(s string) bool       { return bounded(s, 128) && idPattern.MatchString(s) }
func optional(s *string, n int) bool { return s == nil || (bounded(*s, n) && *s != "") }

func Validate(e Event) error {
	if e.SchemaVersion != 1 {
		return invalid("schemaVersion")
	}
	for _, s := range []string{e.EventID, e.OperationID, e.RequestID, e.ProducerID} {
		if !canonicalUUID(s) {
			return invalid("uuid")
		}
	}
	if e.ParentOperationID != nil && (!canonicalUUID(*e.ParentOperationID) || *e.ParentOperationID == e.OperationID) {
		return invalid("parentOperationId")
	}
	for _, t := range []time.Time{e.StartedAt, e.OccurredAt} {
		_, offset := t.Zone()
		if t.IsZero() || offset != 0 {
			return invalid("timestamp")
		}
	}
	if _, err := e.StartedAt.MarshalJSON(); err != nil {
		return invalid("startedAt")
	}
	if _, err := e.OccurredAt.MarshalJSON(); err != nil {
		return invalid("occurredAt")
	}
	if e.Phase == Start {
		if e.Outcome != Unknown || e.Observation != StartOnly || e.DurationMs != nil {
			return invalid("start result")
		}
	} else if e.Phase == Finish {
		if e.Observation != FinishOnly || e.DurationMs == nil || *e.DurationMs < 0 {
			return invalid("finish result")
		}
	} else {
		return invalid("phase")
	}
	if !contains([]string{string(Unknown), string(Succeeded), string(Accepted), string(Failed), string(Denied), string(Partial), string(ActionRequired), string(Cancelled)}, string(e.Outcome)) {
		return invalid("outcome")
	}
	if err := ValidateActor(e.Actor); err != nil {
		return err
	}
	action, ok := Action(e.ActionCode)
	if !ok || action.ModuleCode != e.ModuleCode {
		return invalid("actionCode/moduleCode")
	}
	if !optional(e.BusinessRequestID, 128) {
		return invalid("businessRequestId")
	}
	if e.BusinessState != nil && !code(*e.BusinessState) {
		return invalid("businessState")
	}
	if e.ReasonCode != nil && !code(*e.ReasonCode) {
		return invalid("reasonCode")
	}
	if e.HTTPStatus != nil && (*e.HTTPStatus < 100 || *e.HTTPStatus > 599) {
		return invalid("httpStatus")
	}
	if e.GRPCCode != nil && !contains([]string{"OK", "Canceled", "Unknown", "InvalidArgument", "DeadlineExceeded", "NotFound", "AlreadyExists", "PermissionDenied", "ResourceExhausted", "FailedPrecondition", "Aborted", "OutOfRange", "Unimplemented", "Internal", "Unavailable", "DataLoss", "Unauthenticated"}, *e.GRPCCode) {
		return invalid("grpcCode")
	}
	if len(e.Effect) > 32 {
		return invalid("effect")
	}
	seenEffects := map[string]bool{}
	for _, v := range e.Effect {
		if !code(v) || seenEffects[v] {
			return invalid("effect")
		}
		seenEffects[v] = true
	}
	if len(e.Resources) > 100 {
		return invalid("resources")
	}
	primaries := 0
	for _, r := range e.Resources {
		known := false
		primaryAllowed := false
		for _, en := range entries {
			if en.ResourceType == r.Type {
				known = true
				if en.ActionCode == e.ActionCode {
					primaryAllowed = true
				}
			}
		}
		if !known || !validResourceID(r.Type, r.ID) {
			return invalid("resource")
		}
		if r.Primary {
			primaries++
			if !primaryAllowed {
				return invalid("primaryResource")
			}
		}
	}
	if primaries > 1 {
		return invalid("primaryResource")
	}
	if e.ResourceCount != nil && (*e.ResourceCount < uint64(len(e.Resources)) || (e.ResourcesComplete && *e.ResourceCount != uint64(len(e.Resources)))) {
		return invalid("resourceCount")
	}
	if e.TargetAccountID != nil {
		if !canonicalUUID(*e.TargetAccountID) || !strings.HasPrefix(e.ActionCode, "account.") {
			return invalid("targetAccountId")
		}
		found := false
		for _, r := range e.Resources {
			if r.Type == "account" && r.ID == *e.TargetAccountID {
				found = true
			}
		}
		if !found {
			return invalid("targetAccountId resource")
		}
	}
	if err := ValidateDetails(e.ActionCode, e.Details); err != nil {
		return err
	}
	b, err := marshalCanonical(e)
	if err != nil || len(b) > MaxEventBytes {
		return invalid("event size/encoding")
	}
	return nil
}

// ValidateActor permits verified identities with an unavailable username projection.
func ValidateActor(a Actor) error {
	if !contains([]string{"MEMBER", "ADMINISTRATOR", "UNKNOWN"}, a.Role) || !contains([]string{"MEMBER", "ADMIN", "UNKNOWN"}, a.Realm) || !contains([]string{"LOGIN_SESSION", "API_KEY", "DEVELOPMENT", "UNAUTHENTICATED"}, a.CredentialKind) {
		return invalid("actor enum")
	}
	if a.AccountID != nil && (!a.IdentityVerified || !canonicalUUID(*a.AccountID)) {
		return invalid("actor.accountId")
	}
	if a.UsernameSnapshot != nil && (!a.IdentityVerified || a.AccountID == nil || !optional(a.UsernameSnapshot, 256)) {
		return invalid("actor.usernameSnapshot")
	}
	if a.IdentitySnapshotComplete && (a.AccountID == nil || a.UsernameSnapshot == nil || !a.IdentityVerified || a.Role == "UNKNOWN") {
		return invalid("actor.identitySnapshotComplete")
	}
	if a.Provider != nil && !validProvider(*a.Provider) {
		return invalid("actor.provider")
	}
	return nil
}
func validResourceID(kind, s string) bool {
	switch kind {
	case "account", "binding_attempt", "subscription", "combination", "execution_plan", "execution", "execution_step", "cash_out", "cash_out_batch", "probe_run", "proposal":
		return canonicalUUID(s)
	case "wallet", "notification_delivery":
		return positiveInt64(s)
	case "wallet_address":
		return walletAddressPattern.MatchString(s)
	case "code_hash":
		return codeHashPattern.MatchString(s)
	case "block", "chain":
		return decimal(s)
	default:
		return identifier(s)
	}
}

func ValidateDetails(action string, details Details) error {
	en, ok := Action(action)
	if !ok {
		return invalid("actionCode")
	}
	for k, v := range details {
		if !contains(en.AllowedDetails, k) {
			return invalid("details field")
		}
		if err := validateValue(k, v); err != nil {
			return err
		}
	}
	b, err := json.Marshal(details)
	if err != nil || len(b) > MaxDetailsBytes {
		return invalid("details size")
	}
	return nil
}
func validateValue(k string, v Value) error {
	if len(v.raw) > MaxDetailsBytes || !utf8.Valid(v.raw) {
		return invalid("details encoding")
	}
	parsed, err := decodeUnique(v.raw)
	if err != nil || parsed == nil {
		return invalid("details value")
	}
	bools := []string{"accountCreated", "accountResolved", "beforeAvailable", "confirmedOpen", "cookieCleared", "noActiveSession", "noteChanged", "requestedOpen", "revocationConfirmed"}
	nums := []string{"addedCount", "blockNumber", "chainId", "confirmedCount", "confirmedRevision", "disputeCount", "expectedRevision", "expiresIn", "gatewayCount", "itemCount", "proposalCount", "removedCount", "requestedCount", "requestsPerKey", "revision", "stepCount"}
	ids := []string{"avatarPresetId", "ballotId", "batchId", "cashOutId", "codeHash", "combinationId", "deliveryId", "displayId", "planId", "runId", "stepId", "subscriptionId", "walletAddress", "walletId"}
	if contains(bools, k) {
		if _, ok := parsed.(bool); !ok {
			return invalid("details bool")
		}
		return nil
	}
	if contains(nums, k) {
		s, ok := parsed.(string)
		if !ok || !decimal(s) {
			return invalid("details uint64")
		}
		return nil
	}
	if contains(ids, k) {
		s, ok := parsed.(string)
		if !ok || !validDetailID(k, s) {
			return invalid("details id")
		}
		return nil
	}
	if k == "walletIds" || k == "changedFields" {
		a, ok := parsed.([]any)
		limit := 100
		if k == "changedFields" {
			limit = 32
		}
		if !ok || len(a) > limit {
			return invalid("details array")
		}
		for _, v := range a {
			s, ok := v.(string)
			if !ok || (k == "walletIds" && !positiveInt64(s)) || (k == "changedFields" && !code(s)) {
				return invalid("details array member")
			}
		}
		return nil
	}
	if k == "requestedAccess" || k == "confirmedAccess" {
		return validateAccess(v.raw, parsed)
	}
	s, ok := parsed.(string)
	if !ok {
		return invalid("details string")
	}
	if contains([]string{"slug", "oldSlug", "newSlug"}, k) {
		if !bounded(s, 256) || s == "" || strings.TrimSpace(s) != s {
			return invalid("details slug")
		}
		return nil
	}
	if !code(s) {
		return invalid("details enum")
	}
	// Domain enums retain the source service's spelling; values with several
	// action-specific domains (state/stage/warningCode) are explicitly mapped by
	// adapters, never copied from error text.
	switch k {
	case "provider":
		if !validProvider(s) {
			return invalid("provider")
		}
	case "proofKind":
		if !contains([]string{"GOOGLE", "PHANTOM", "DEVELOPMENT"}, s) {
			return invalid("proofKind")
		}
	case "walletType":
		if !contains([]string{"EVM", "SOLANA"}, s) {
			return invalid("walletType")
		}
	case "avatarKind":
		if !contains([]string{"default", "preset", "upload"}, s) {
			return invalid("avatarKind")
		}
	case "attemptStatus":
		if !contains([]string{"pending", "failed", "PENDING", "FAILED", "TELEGRAM_BINDING_ATTEMPT_STATUS_PENDING", "TELEGRAM_BINDING_ATTEMPT_STATUS_FAILED"}, s) {
			return invalid("attemptStatus")
		}
	case "bindingStatus":
		if !contains([]string{"connected", "unreachable", "CONNECTED", "UNREACHABLE", "NOT_CONNECTED", "TELEGRAM_BINDING_STATUS_CONNECTED", "TELEGRAM_BINDING_STATUS_UNREACHABLE"}, s) {
			return invalid("bindingStatus")
		}
	case "deliveryStatus":
		if !contains([]string{"pending", "sending", "sent", "failed", "cancelled", "unknown", "PENDING", "SENDING", "SENT", "FAILED", "CANCELLED", "UNKNOWN", "NOTIFICATION_DELIVERY_STATUS_PENDING", "NOTIFICATION_DELIVERY_STATUS_SENDING", "NOTIFICATION_DELIVERY_STATUS_SENT", "NOTIFICATION_DELIVERY_STATUS_FAILED", "NOTIFICATION_DELIVERY_STATUS_CANCELLED", "NOTIFICATION_DELIVERY_STATUS_UNKNOWN"}, s) {
			return invalid("deliveryStatus")
		}
	case "moduleKey":
		if !contains([]string{"trader_sync", "solana", "market_radar", "managed_oo", "profit_sharing", "worm"}, s) {
			return invalid("moduleKey")
		}
	case "requestedTier", "confirmedTier":
		if !contains([]string{"STANDARD", "PRO", "ACCOUNT_TIER_STANDARD", "ACCOUNT_TIER_PRO", "standard", "pro"}, s) {
			return invalid("tier")
		}
	}
	return nil
}
func validateAccess(raw []byte, parsed any) error {
	obj, ok := parsed.(map[string]any)
	if !ok || !exactFields(parsed, reflect.TypeOf(Access{})) {
		return invalid("access shape")
	}
	for _, k := range []string{"loginEnabled", "apiKeyEnabled", "profitSharingEnabled"} {
		if _, ok := obj[k].(bool); !ok {
			return invalid("access bool")
		}
	}
	pairs, ok := obj["moduleAccess"].([]any)
	if !ok {
		return invalid("access moduleAccess")
	}
	for _, pair := range pairs {
		if !exactFields(pair, reflect.TypeOf(ModuleAccess{})) {
			return invalid("access pair fields")
		}
	}
	var a Access
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&a); err != nil || !decimal(a.Revision) || a.ModuleAccess == nil {
		return invalid("access shape")
	}
	modules := map[string]string{"market_radar": "read", "managed_oo": "read_write", "worm_trading": "read_write", "token": "read_write", "solana": "read", "wallet": "read_write", "trader_sync": "read_write"}
	seen := map[string]bool{}
	for _, m := range a.ModuleAccess {
		max, ok := modules[m.Module]
		if !ok || seen[m.Module] || !contains([]string{"none", "read", "read_write"}, m.Access) || (m.Access == "read_write" && max == "read") {
			return invalid("access module")
		}
		seen[m.Module] = true
	}
	return nil
}

// decodeUnique rejects duplicate keys (including escaped spellings) before
// canonicalization, so conflicting payloads cannot collapse to the same hash.
func decodeUnique(b []byte) (any, error) {
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	var read func() (any, error)
	read = func() (any, error) {
		tok, err := d.Token()
		if err != nil {
			return nil, err
		}
		delim, ok := tok.(json.Delim)
		if !ok {
			return tok, nil
		}
		switch delim {
		case '{':
			m := map[string]any{}
			for d.More() {
				key, err := d.Token()
				if err != nil {
					return nil, err
				}
				k, ok := key.(string)
				if !ok {
					return nil, ErrInvalid
				}
				if _, exists := m[k]; exists {
					return nil, ErrInvalid
				}
				v, err := read()
				if err != nil {
					return nil, err
				}
				m[k] = v
			}
			_, err = d.Token()
			return m, err
		case '[':
			a := []any{}
			for d.More() {
				v, err := read()
				if err != nil {
					return nil, err
				}
				a = append(a, v)
			}
			_, err = d.Token()
			return a, err
		default:
			return nil, ErrInvalid
		}
	}
	value, err := read()
	if err != nil {
		return nil, err
	}
	if _, err = d.Token(); err != io.EOF {
		return nil, ErrInvalid
	}
	return value, nil
}
func marshalCanonical(e Event) ([]byte, error) {
	// Fix absence representations and sort details at every nesting level. Struct
	// field order is stable; list order carries source meaning and is retained.
	if e.Resources == nil {
		e.Resources = []Resource{}
	}
	if e.Effect == nil {
		e.Effect = []string{}
	}
	details := Details{}
	for k, v := range e.Details {
		parsed, err := decodeUnique(v.raw)
		if err != nil {
			return nil, err
		}
		b, err := json.Marshal(parsed)
		if err != nil {
			return nil, err
		}
		details[k] = Value{raw: b}
	}
	e.Details = details
	return json.Marshal(e)
}
func Canonical(e Event) ([]byte, error) {
	if err := Validate(e); err != nil {
		return nil, err
	}
	return marshalCanonical(e)
}

// Decode accepts only the current, strict envelope and enforces all size limits.
func Decode(b []byte) (Event, error) {
	var e Event
	if len(b) > MaxEventBytes || !utf8.Valid(b) {
		return e, invalid("encoding")
	}
	parsed, err := decodeUnique(b)
	if err != nil {
		return e, invalid("encoding")
	}
	if !exactFields(parsed, reflect.TypeOf(e)) {
		return e, invalid("envelope fields")
	}
	obj := parsed.(map[string]any)
	if !exactFields(obj["actor"], reflect.TypeOf(Actor{})) {
		return e, invalid("actor fields")
	}
	if refs, ok := obj["resources"].([]any); ok {
		for _, r := range refs {
			if !exactFields(r, reflect.TypeOf(Resource{})) {
				return e, invalid("resource fields")
			}
		}
	}
	for _, k := range []string{"durationMs", "resourceCount"} {
		if obj[k] != nil {
			s, ok := obj[k].(string)
			if !ok || !decimal(s) {
				return e, invalid(k)
			}
		}
	}

	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(&e); err != nil {
		return Event{}, invalid("encoding")
	}
	return e, Validate(e)
}

// TruncateUTF8 bounds a valid string without splitting a code point. Invalid
// UTF-8 is preserved so validation rejects it instead of silently changing it.
func TruncateUTF8(s string, n int) string {
	if !utf8.ValidString(s) {
		return s
	}
	if n < 0 {
		n = 0
	}
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}
func exactFields(v any, t reflect.Type) bool {
	obj, ok := v.(map[string]any)
	if !ok || len(obj) != t.NumField() {
		return false
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := strings.Split(field.Tag.Get("json"), ",")
		value, present := obj[tag[0]]
		if !present || !jsonFieldType(value, field.Type, contains(tag[1:], "string")) {
			return false
		}
	}
	return true
}

// jsonFieldType rejects null before encoding/json can turn it into a scalar's
// zero value. Only pointer fields have nullable semantics in the envelope.
func jsonFieldType(v any, t reflect.Type, encodedString bool) bool {
	if t.Kind() == reflect.Pointer {
		if v == nil {
			return true
		}
		return jsonFieldType(v, t.Elem(), encodedString)
	}
	if v == nil {
		return false
	}
	if t == reflect.TypeOf(time.Time{}) {
		_, ok := v.(string)
		return ok
	}
	if encodedString {
		_, ok := v.(string)
		return ok
	}
	switch t.Kind() {
	case reflect.Bool:
		_, ok := v.(bool)
		return ok
	case reflect.String:
		_, ok := v.(string)
		return ok
	case reflect.Int, reflect.Int64, reflect.Uint64:
		_, ok := v.(json.Number)
		return ok
	case reflect.Struct:
		return exactFields(v, t)
	case reflect.Slice:
		items, ok := v.([]any)
		if !ok {
			return false
		}
		for _, item := range items {
			if !jsonFieldType(item, t.Elem(), false) {
				return false
			}
		}
		return true
	case reflect.Map:
		_, ok := v.(map[string]any)
		return ok // Details has its action-specific validation.
	default:
		return false
	}
}

// BoundedDetail applies display-only truncation. Identifiers and semantic codes
// are never truncated into a different business fact; invalid values are rejected.
func BoundedDetail(key string, v Value) Value {
	if key != "slug" && key != "oldSlug" && key != "newSlug" {
		return v
	}
	parsed, err := decodeUnique(v.raw)
	if err != nil {
		return v
	}
	s, ok := parsed.(string)
	if !ok || !utf8.Valid(v.raw) {
		return v
	}
	return String(TruncateUTF8(s, 256))
}

func validProvider(s string) bool {
	return contains([]string{"google", "solana_wallet", "development", "GOOGLE", "SOLANA_WALLET", "DEVELOPMENT", "PHANTOM"}, s)
}
func positiveInt64(s string) bool {
	v, err := strconv.ParseInt(s, 10, 64)
	return err == nil && v > 0 && decimalPattern.MatchString(s)
}
func validDetailID(key, s string) bool {
	switch key {
	case "batchId", "cashOutId", "combinationId", "planId", "runId", "stepId", "subscriptionId":
		return canonicalUUID(s)
	case "walletId", "deliveryId", "ballotId":
		return positiveInt64(s)
	case "codeHash":
		return validResourceID("code_hash", s)
	case "walletAddress":
		return validResourceID("wallet_address", s)
	default:
		return identifier(s)
	}
}
