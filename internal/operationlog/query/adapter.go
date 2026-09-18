package query

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type TxReader interface {
	Read(context.Context, func(pgx.Tx) error) error
}
type Summary struct {
	OperationID                                                                                      string
	StartedAt                                                                                        time.Time
	ActorAccountID, ActorUsername, ActorRole, Realm, CredentialKind                                  string
	IdentityVerified, IdentitySnapshotComplete                                                       bool
	ModuleCode, ActionCode, PrimaryResourceType, PrimaryResourceID, Outcome, Observation, ReasonCode string
	DurationMS                                                                                       *int64
	BusinessState                                                                                    string
	ResourceCount                                                                                    *int64
	ResourcesComplete, ResponseWriteFailed                                                           bool
}
type Detail struct {
	Summary
	FinishedAt                                      *time.Time
	RequestID, ParentOperationID, BusinessRequestID string
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
type RuntimeStatus struct {
	ServiceEpoch            string
	CheckedAt               time.Time
	QueryReady              bool
	ProjectionState         string
	LastPublishedAt         *time.Time
	PublicationSequence     int64
	PendingEvents           int64
	QuarantinedEvents       int64
	LastProcessingErrorCode string
}
type Adapter struct {
	Reader TxReader
	Codec  *Codec
	Now    func() time.Time
}

func (a *Adapter) now() time.Time {
	if a.Now != nil {
		return a.Now().UTC()
	}
	return time.Now().UTC()
}
func (a *Adapter) List(ctx context.Context, f Filter, viewer Viewer) (Page, error) {
	now := a.now()
	nf, err := NormalizeFilter(f, now)
	if err != nil {
		return Page{}, err
	}
	var cd CursorData
	if f.Cursor != "" {
		cd, err = a.Codec.DecodeCursor(f.Cursor, viewer, now)
		if err != nil {
			return Page{}, err
		}
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
			if e := tx.QueryRow(ctx, "SELECT last_seq, COALESCE(last_published_at,clock_timestamp()) FROM operation_log.publication WHERE singleton_id=1").Scan(&seq, &at); e != nil {
				return e
			}
		}
		args := []any{seq, nf.From, nf.To}
		sql := "SELECT operation_id,started_at,actor_account_id,actor_username,actor_role,realm,credential_kind,module_code,action_code,outcome,observation,target_account_id,primary_resource_type,primary_resource_id,duration_ms,detail FROM operation_log.entry_version WHERE visible_from_seq <= $1 AND (visible_to_seq IS NULL OR visible_to_seq > $1) AND started_at >= $2 AND started_at < $3"
		n := 4
		appendCond := func(cond string, v any) { sql += " AND " + cond + fmt.Sprintf(" $%d", n); args = append(args, v); n++ }
		if nf.ActorQuery != "" {
			if u, e := uuid.Parse(nf.ActorQuery); e == nil {
				appendCond("actor_account_id =", u)
			} else {
				sql += fmt.Sprintf(" AND lower(actor_username) LIKE lower($%d)", n)
				args = append(args, nf.ActorQuery+"%")
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
		if nf.Outcome != "" {
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
			items = append(items, Summary{OperationID: uid(op), StartedAt: started.Time, ActorAccountID: uid(aid), ActorUsername: text(uname), ActorRole: arole.String, Realm: realm.String, CredentialKind: cred.String, ModuleCode: mod.String, ActionCode: act.String, Outcome: outcome.String, Observation: obs.String, PrimaryResourceType: text(rt), PrimaryResourceID: text(rid), DurationMS: duration})
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
		p.SnapshotToken, _ = a.Codec.EncodeSnapshotAt(nf, viewer, seq, at, now)
		if next {
			last := items[len(items)-1]
			p.NextCursor, _ = a.Codec.EncodeCursorAt(nf, viewer, CursorPosition{StartedAt: last.StartedAt, OperationID: last.OperationID}, seq, at, now)
		}
	}
	return p, nil
}
func (a *Adapter) Get(ctx context.Context, id string, viewer Viewer, snapshot string) (Detail, error) {
	if _, e := uuid.Parse(id); e != nil {
		return Detail{}, fmt.Errorf("invalid operation id")
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
		d = Detail{Summary: Summary{OperationID: uid(op), StartedAt: started.Time, ActorAccountID: uid(aid), ActorUsername: text(uname), ActorRole: role.String, Realm: realm.String, CredentialKind: cred.String, ModuleCode: mod.String, ActionCode: act.String, Outcome: outcome.String, Observation: obs.String, PrimaryResourceType: text(rt), PrimaryResourceID: text(rid), DurationMS: duration}, Detail: detail, RequestID: uid(req), ParentOperationID: uid(parent), BusinessRequestID: text(biz)}
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
