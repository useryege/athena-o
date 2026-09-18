package query

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/useryege/athena/internal/operationlog/event"
)

type TxReader interface {
	Read(context.Context, func(pgx.Tx) error) error
}
type Summary struct {
	OperationID                                                                                      string
	StartedAt                                                                                        time.Time
	ActorAccountID, ActorUsername, ActorRole, Realm, CredentialKind, TargetAccountID                 string
	IdentityVerified, IdentitySnapshotComplete                                                       bool
	ModuleCode, ActionCode, PrimaryResourceType, PrimaryResourceID, Outcome, Observation, ReasonCode string
	DurationMS                                                                                       *int64
	BusinessState                                                                                    string
	ResourceCount                                                                                    *int64
	ResourcesComplete, ResponseWriteFailed                                                           bool
	Provider                                                                                         string
}
type ResourceFact struct {
	Type, ID, Relation string
	ReferenceVerified  bool
}
type ChangeFact struct {
	FieldCode               string
	BeforeValue, AfterValue *string
	BeforeAvailable         bool
	ValueKind               string
}
type CountsFact struct{ Requested, Confirmed, Failed, Unknown *int64 }
type ProtocolFact struct {
	GRPCCode, HTTPStatus, ReasonCode string
	ResponseWriteFailed              bool
}
type SourceFacts struct {
	ProducerID                      string
	FirstReceivedAt, LastReceivedAt time.Time
	PhasesReceived                  []string
	SnapshotSequence                int64
}
type Detail struct {
	Summary
	FinishedAt                                      *time.Time
	RequestID, ParentOperationID, BusinessRequestID string
	Effect                                          []string
	Resources                                       []ResourceFact
	Changes                                         []ChangeFact
	Counts                                          CountsFact
	Protocol                                        ProtocolFact
	Source                                          SourceFacts
	Detail                                          json.RawMessage
	SourceEventIDs                                  []string
}
type Page struct {
	Items                     []Summary
	NextCursor, SnapshotToken string
	SnapshotSequence          int64
	SnapshotAt                time.Time
	PageSize                  int
	AppliedFilters            NormalizedFilter
}
type ProducerStatus struct {
	ProducerID                                                                                 string
	StartedAt, LastSeenAt, StoppedAt                                                           *time.Time
	AttemptedEvents, ConfirmedEvents, UnconfirmedEvents, InvalidEvents, CapacityRejectedEvents int64
	LastFailureAt, LastRecoveredAt                                                             *time.Time
	LastFailureCode                                                                            *string
	PersistenceReachable                                                                       *bool
}
type RuntimeStatus struct {
	ServiceEpoch            string
	CheckedAt               time.Time
	QueryReady              bool
	ProjectionState         string
	LastPublishedAt         *time.Time
	PublicationSequence     int64
	PendingEvents           int64
	OldestPendingReceivedAt *time.Time
	QuarantinedEvents       int64
	LastProcessingErrorCode *string
	Producers               []ProducerStatus
	TotalObserved           int64
	ProducersComplete       bool
}
type RuntimeReader interface {
	RuntimeStatus(context.Context) (RuntimeStatus, error)
}
type Adapter struct {
	Reader        TxReader
	Codec         *Codec
	Now           func() time.Time
	RuntimeSource RuntimeReader
}

func (a *Adapter) now() time.Time {
	if a.Now != nil {
		return a.Now().UTC()
	}
	return time.Now().UTC()
}
func (a *Adapter) List(ctx context.Context, f Filter, viewer Viewer) (Page, error) {
	now := a.now()
	var cd CursorData
	if f.Cursor != "" {
		if a.Codec == nil {
			return Page{}, ErrCursorInvalid
		}
		var err error
		cd, err = a.Codec.DecodeCursor(f.Cursor, viewer, now)
		if err != nil {
			return Page{}, err
		}
		// A follow-up page inherits the signed range and filters. Explicit
		// values are normalized below and then compared to the token.
		if f.From == nil {
			v := cd.Filters.From
			f.From = &v
		}
		if f.To == nil {
			v := cd.Filters.To
			f.To = &v
		}
		if f.PageSize == 0 {
			f.PageSize = cd.Filters.PageSize
		}
		if f.ActorQuery == "" {
			f.ActorQuery = cd.Filters.ActorQuery
		}
		if f.ActorRole == "" {
			f.ActorRole = cd.Filters.ActorRole
		}
		if f.CredentialKind == "" {
			f.CredentialKind = cd.Filters.CredentialKind
		}
		if f.ModuleCode == "" {
			f.ModuleCode = cd.Filters.ModuleCode
		}
		if f.ActionCode == "" {
			f.ActionCode = cd.Filters.ActionCode
		}
		if f.Outcome == "" {
			f.Outcome = cd.Filters.Outcome
		}
		if f.TargetAccountID == "" {
			f.TargetAccountID = cd.Filters.TargetAccountID
		}
		if f.ResourceType == "" {
			f.ResourceType = cd.Filters.ResourceType
		}
		if f.ResourceID == "" {
			f.ResourceID = cd.Filters.ResourceID
		}
	}
	nf, err := NormalizeFilter(f, now)
	if err != nil {
		return Page{}, err
	}
	if f.Cursor != "" {
		if err = a.Codec.ValidateCursorParameters(f.Cursor, nf, viewer, now); err != nil {
			return Page{}, err
		}
	}
	seq := int64(0)
	at := now
	pos := CursorPosition{}
	if f.Cursor != "" {
		seq, at, pos = cd.SnapshotSequence, cd.SnapshotAt, cd.Position
	}
	if a.Reader == nil {
		return Page{}, fmt.Errorf("query reader unavailable")
	}
	var items []Summary
	var next bool
	err = a.Reader.Read(ctx, func(tx pgx.Tx) error {
		if _, e := tx.Exec(ctx, "SET LOCAL statement_timeout = '2500ms'"); e != nil {
			return e
		}
		if f.Cursor == "" {
			if e := tx.QueryRow(ctx, "SELECT last_seq FROM operation_log.publication WHERE singleton_id=1").Scan(&seq); e != nil {
				return e
			}
			// snapshotAt identifies when this query snapshot was created, not
			// when the last publication happened.
			at = now
		}
		args := []any{seq, nf.From, nf.To}
		sql := "SELECT operation_id,started_at,actor_account_id,actor_username,actor_role,realm,credential_kind,module_code,action_code,outcome,observation,target_account_id,primary_resource_type,primary_resource_id,duration_ms,detail FROM operation_log.entry_version WHERE visible_from_seq <= $1 AND (visible_to_seq IS NULL OR visible_to_seq > $1) AND started_at >= $2 AND started_at < $3"
		n := 4
		appendCond := func(cond string, v any) { sql += " AND " + cond + fmt.Sprintf(" $%d", n); args = append(args, v); n++ }
		if nf.ActorQuery != "" {
			if u, e := uuid.Parse(nf.ActorQuery); e == nil {
				appendCond("actor_account_id =", u)
			} else {
				sql += fmt.Sprintf(" AND lower(actor_username) LIKE lower($%d) ESCAPE '\\'", n)
				args = append(args, escapeLike(nf.ActorQuery)+"%")
				n++
			}
		}
		// Filters with stable bind numbering.
		if nf.ActorRole != "" && nf.ActorRole != "ALL" {
			appendCond("actor_role =", nf.ActorRole)
		}
		if nf.CredentialKind != "" && nf.CredentialKind != "ALL" {
			appendCond("credential_kind =", nf.CredentialKind)
		}
		if nf.ModuleCode != "" {
			appendCond("module_code =", nf.ModuleCode)
		}
		if nf.ActionCode != "" {
			appendCond("action_code =", nf.ActionCode)
		}
		if nf.Outcome != "" && nf.Outcome != "ALL" {
			appendCond("outcome =", nf.Outcome)
		}
		if nf.TargetAccountID != "" {
			u, _ := uuid.Parse(nf.TargetAccountID)
			appendCond("target_account_id =", u)
		}
		if nf.ResourceType != "" {
			appendCond("primary_resource_type =", nf.ResourceType)
			appendCond("primary_resource_id =", nf.ResourceID)
		}
		if f.Cursor != "" {
			sql += fmt.Sprintf(" AND (started_at, operation_id) < ($%d, $%d)", n, n+1)
			args = append(args, pos.StartedAt, pos.OperationID)
			n += 2
		}
		sql += " ORDER BY started_at DESC, operation_id DESC LIMIT " + fmt.Sprintf("%d", nf.PageSize+1)
		rows, e := tx.Query(ctx, sql, args...)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var op, aid pgtype.UUID
			var started pgtype.Timestamptz
			var uname, arole, realm, cred, mod, act, outcome, obs, tid, rt, rid pgtype.Text
			var dur pgtype.Int8
			var detail []byte
			if e := rows.Scan(&op, &started, &aid, &uname, &arole, &realm, &cred, &mod, &act, &outcome, &obs, &tid, &rt, &rid, &dur, &detail); e != nil {
				return e
			}
			uid := func(v pgtype.UUID) string {
				if v.Valid {
					return uuid.UUID(v.Bytes).String()
				}
				return ""
			}
			text := func(v pgtype.Text) string {
				if v.Valid {
					return v.String
				}
				return ""
			}
			var duration *int64
			if dur.Valid {
				x := dur.Int64
				duration = &x
			}
			sm := Summary{OperationID: uid(op), StartedAt: started.Time, ActorAccountID: uid(aid), ActorUsername: text(uname), ActorRole: arole.String, Realm: realm.String, CredentialKind: cred.String, TargetAccountID: text(tid), ModuleCode: mod.String, ActionCode: act.String, Outcome: outcome.String, Observation: obs.String, PrimaryResourceType: text(rt), PrimaryResourceID: text(rid), DurationMS: duration}
			enrichSummary(&sm, detail)
			items = append(items, sm)
		}
		if e := rows.Err(); e != nil {
			return e
		}
		if len(items) > nf.PageSize {
			items = items[:nf.PageSize]
			next = true
		}
		return nil
	})
	if err != nil {
		return Page{}, err
	}
	p := Page{Items: items, PageSize: nf.PageSize, AppliedFilters: nf, SnapshotSequence: seq, SnapshotAt: at}
	if a.Codec != nil {
		p.SnapshotToken, _ = a.Codec.EncodeSnapshotPreserving(nf, viewer, seq, at, now, now.Add(CursorTTL), now)
		if next {
			last := items[len(items)-1]
			p.NextCursor, _ = a.Codec.EncodeCursorPreserving(nf, viewer, CursorPosition{StartedAt: last.StartedAt, OperationID: last.OperationID}, seq, at, cd.IssuedAt, cd.ExpiresAt, now)
		}
	}
	return p, nil
}
func (a *Adapter) Get(ctx context.Context, id string, viewer Viewer, snapshot string) (Detail, error) {
	u, e := uuid.Parse(id)
	if e != nil || u == uuid.Nil || u.String() != id {
		return Detail{}, fmt.Errorf("%w: %s", ErrInvalidOperationID, id)
	}
	if a.Reader == nil {
		return Detail{}, fmt.Errorf("query reader unavailable")
	}
	now := a.now()
	seq := int64(0)
	if snapshot != "" {
		if a.Codec == nil {
			return Detail{}, ErrCursorInvalid
		}
		s, e := a.Codec.DecodeSnapshot(snapshot, viewer, now)
		if e != nil {
			return Detail{}, e
		}
		seq = s.SnapshotSequence
	}
	var d Detail
	err := a.Reader.Read(ctx, func(tx pgx.Tx) error {
		if _, e := tx.Exec(ctx, "SET LOCAL statement_timeout = '1500ms'"); e != nil {
			return e
		}
		q := "SELECT operation_id,started_at,finished_at,actor_account_id,actor_username,actor_role,realm,credential_kind,module_code,action_code,outcome,observation,target_account_id,primary_resource_type,primary_resource_id,duration_ms,detail,request_id,parent_operation_id,business_request_id,source_event_ids,visible_from_seq FROM operation_log.entry_version WHERE operation_id=$1"
		args := []any{id}
		if snapshot != "" {
			q += " AND visible_from_seq <= $2 AND (visible_to_seq IS NULL OR visible_to_seq > $2)"
			args = append(args, seq)
		} else {
			q += " AND visible_to_seq IS NULL"
		}
		var op, req, parent pgtype.UUID
		var started, finished pgtype.Timestamptz
		var aid pgtype.UUID
		var uname, role, realm, cred, mod, act, outcome, obs, tid, rt, rid pgtype.Text
		var dur pgtype.Int8
		var detail []byte
		var biz pgtype.Text
		var src []pgtype.UUID
		var from int64
		if e := tx.QueryRow(ctx, q, args...).Scan(&op, &started, &finished, &aid, &uname, &role, &realm, &cred, &mod, &act, &outcome, &obs, &tid, &rt, &rid, &dur, &detail, &req, &parent, &biz, &src, &from); e != nil {
			if errors.Is(e, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return e
		}
		text := func(v pgtype.Text) string {
			if v.Valid {
				return v.String
			}
			return ""
		}
		uid := func(v pgtype.UUID) string {
			if v.Valid {
				return uuid.UUID(v.Bytes).String()
			}
			return ""
		}
		var duration *int64
		if dur.Valid {
			x := dur.Int64
			duration = &x
		}
		d = Detail{Summary: Summary{OperationID: uid(op), StartedAt: started.Time, ActorAccountID: uid(aid), ActorUsername: text(uname), ActorRole: role.String, Realm: realm.String, CredentialKind: cred.String, TargetAccountID: text(tid), ModuleCode: mod.String, ActionCode: text(act), Outcome: outcome.String, Observation: obs.String, PrimaryResourceType: text(rt), PrimaryResourceID: text(rid), DurationMS: duration}, Detail: detail, RequestID: uid(req), ParentOperationID: uid(parent), BusinessRequestID: text(biz)}
		enrichDetail(&d, detail)
		d.Source.SnapshotSequence = from
		for _, u := range src {
			if u.Valid {
				d.SourceEventIDs = append(d.SourceEventIDs, uuid.UUID(u.Bytes).String())
			}
		}
		if finished.Valid {
			t := finished.Time
			d.FinishedAt = &t
		}
		return nil
	})
	return d, err
}

func escapeLike(v string) string {
	v = strings.ReplaceAll(v, "\\", "\\\\")
	v = strings.ReplaceAll(v, "%", "\\%")
	return strings.ReplaceAll(v, "_", "\\_")
}

func enrichSummary(s *Summary, b []byte) {
	var e event.Event
	if json.Unmarshal(b, &e) != nil {
		return
	}
	s.IdentityVerified = e.Actor.IdentityVerified
	s.IdentitySnapshotComplete = e.Actor.IdentitySnapshotComplete
	s.BusinessState = valueString(e.BusinessState)
	s.ReasonCode = valueString(e.ReasonCode)
	s.ResponseWriteFailed = e.ResponseWriteFailed
	s.Provider = valueString(e.Actor.Provider)
	if e.ResourceCount != nil && *e.ResourceCount <= math.MaxInt64 {
		x := int64(*e.ResourceCount)
		s.ResourceCount = &x
	}
	s.ResourcesComplete = e.ResourcesComplete
}
func enrichDetail(d *Detail, b []byte) {
	var v struct {
		event.Event
		FirstReceivedAt time.Time `json:"firstReceivedAt"`
		LastReceivedAt  time.Time `json:"lastReceivedAt"`
		PhasesReceived  []string  `json:"phasesReceived"`
	}
	if json.Unmarshal(b, &v) != nil {
		return
	}
	enrichSummary(&d.Summary, b)
	d.Effect = append([]string(nil), v.Effect...)
	for _, r := range v.Resources {
		rel := "RELATED"
		if r.Primary {
			rel = "PRIMARY"
		}
		d.Resources = append(d.Resources, ResourceFact{Type: r.Type, ID: r.ID, Relation: rel, ReferenceVerified: r.ReferenceVerified})
	}
	d.Source.ProducerID = v.ProducerID
	d.Source.FirstReceivedAt = v.FirstReceivedAt
	d.Source.LastReceivedAt = v.LastReceivedAt
	if len(v.PhasesReceived) > 0 {
		d.Source.PhasesReceived = append([]string(nil), v.PhasesReceived...)
	} else if v.Phase != "" {
		d.Source.PhasesReceived = []string{string(v.Phase)}
	}
	if v.GRPCCode != nil {
		d.Protocol.GRPCCode = *v.GRPCCode
	}
	if v.HTTPStatus != nil {
		d.Protocol.HTTPStatus = fmt.Sprintf("%d", *v.HTTPStatus)
	}
	if v.ReasonCode != nil {
		d.Protocol.ReasonCode = *v.ReasonCode
	}
	d.Protocol.ResponseWriteFailed = v.ResponseWriteFailed
	keys := make([]string, 0, len(v.Details))
	for k := range v.Details {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		raw, err := json.Marshal(v.Details[k])
		if err != nil {
			continue
		}
		value := string(raw)
		d.Changes = append(d.Changes, ChangeFact{FieldCode: k, AfterValue: &value, BeforeAvailable: false, ValueKind: "JSON"})
		switch k {
		case "requested", "requestedCount":
			d.Counts.Requested = jsonInt(raw)
		case "confirmed", "confirmedCount":
			d.Counts.Confirmed = jsonInt(raw)
		case "failed", "failedCount":
			d.Counts.Failed = jsonInt(raw)
		case "unknown", "unknownCount":
			d.Counts.Unknown = jsonInt(raw)
		}
	}
}
func jsonInt(raw []byte) *int64 {
	var x int64
	if json.Unmarshal(raw, &x) == nil {
		return &x
	}
	var f float64
	if json.Unmarshal(raw, &f) == nil && f >= math.MinInt64 && f <= math.MaxInt64 {
		y := int64(f)
		return &y
	}
	return nil
}

func valueString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
func (a *Adapter) Runtime(ctx context.Context) (RuntimeStatus, error) {
	if a.RuntimeSource == nil {
		return RuntimeStatus{}, fmt.Errorf("runtime status unavailable")
	}
	return a.RuntimeSource.RuntimeStatus(ctx)
}
