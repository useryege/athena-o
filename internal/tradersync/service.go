package tradersync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/useryege/athena/internal/tradersync/store"
	tm "github.com/useryege/athena/internal/tradersync/types"
	"github.com/useryege/athena/util/ethws"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/url"
	"strconv"
	"time"
)

type Dependencies struct {
	Pool          *pgxpool.Pool
	Resolver      *TargetResolver
	Subscriptions *SubscriptionService
	Collector     *Collector
	Projector     *Projector
	Directory     *DirectoryRefresher
}
type Service struct {
	deps  Dependencies
	reads *store.SQLStore
	key   []byte
}

func NewService(cfg Config, deps Dependencies) (*Service, error) {
	if e := cfg.Validate(); e != nil {
		return nil, e
	}
	if deps.Pool == nil || deps.Resolver == nil || deps.Subscriptions == nil || deps.Collector == nil || deps.Projector == nil || deps.Directory == nil {
		return nil, fmt.Errorf("Trader Sync requires pool, resolver, subscriptions, collector, projector and directory")
	}
	if deps.Subscriptions.baseline != deps.Collector {
		return nil, fmt.Errorf("subscription registrar must be the running collector instance")
	}
	return &Service{deps: deps, reads: store.NewSQLStore(deps.Pool), key: []byte(cfg.CursorHMACKey)}, nil
}
func (c Config) Validate() error {
	for _, item := range []struct {
		name, value string
		schemes     []string
	}{{"HTTP", c.HTTPURL, []string{"http", "https"}}, {"WSS", c.WebSocketURL, []string{"ws", "wss"}}, {"site", c.SiteURL, []string{"http", "https"}}} {
		u, e := url.Parse(item.value)
		allowed := false
		if e == nil {
			for _, scheme := range item.schemes {
				allowed = allowed || u.Scheme == scheme
			}
		}
		if e != nil || !allowed || u.Host == "" || u.Fragment != "" || u.User != nil || (item.name == "site" && u.RawQuery != "") {
			return fmt.Errorf("Trader Sync absolute %s URL required", item.name)
		}
	}
	if c.CursorHMACKey == "" {
		return fmt.Errorf("Trader Sync stable cursor HMAC key required")
	}
	if e := ethws.ValidateProxyURL(c.ProxyURL); e != nil {
		return e
	}
	return nil
}

func (s *Service) ResolveTarget(ctx context.Context, owner, input string) (tm.ResolvedTarget, error) {
	return s.deps.Resolver.Resolve(ctx, owner, input)
}
func (s *Service) CreateSubscription(ctx context.Context, owner string, in tm.CreateInput) (tm.SubscriptionDetails, error) {
	v, e := s.deps.Subscriptions.Create(ctx, owner, in)
	if e != nil {
		return tm.SubscriptionDetails{}, e
	}
	return s.GetSubscription(ctx, owner, v.ID)
}
func (s *Service) ChangeSubscription(ctx context.Context, owner, action string, in tm.ChangeInput) (tm.SubscriptionDetails, error) {
	v, e := s.deps.Subscriptions.Change(ctx, owner, action, in)
	if e != nil {
		return tm.SubscriptionDetails{}, e
	}
	return s.GetSubscription(ctx, owner, v.ID)
}
func (s *Service) UpdateTargetNote(ctx context.Context, owner string, in tm.NoteInput) (tm.TargetNote, error) {
	return s.deps.Subscriptions.UpdateNote(ctx, owner, in)
}
func (s *Service) GetSubscription(ctx context.Context, owner, id string) (tm.SubscriptionDetails, error) {
	if id == "" {
		return tm.SubscriptionDetails{}, status.Error(codes.InvalidArgument, "subscription required")
	}
	p, e := s.reads.ReadSubscriptions(ctx, owner, tm.SubscriptionFilter{ID: id}, tm.ReadPage{Limit: 1})
	if e != nil {
		return tm.SubscriptionDetails{}, e
	}
	return p.Subscriptions[0], nil
}
func positiveResource(v string) (int64, error) {
	n, e := cursorNumber(v)
	if e != nil || n == 0 {
		return 0, status.Error(codes.InvalidArgument, "positive decimal resource ID required")
	}
	return n, nil
}
func (s *Service) GetActivity(ctx context.Context, owner, id string) (tm.ActivityDetails, error) {
	n, e := positiveResource(id)
	if e != nil {
		return tm.ActivityDetails{}, e
	}
	p, e := s.reads.ReadActivities(ctx, owner, tm.ActivityReadInput{Limit: 1, ActivityID: n})
	if e != nil {
		return tm.ActivityDetails{}, e
	}
	return p.Activities[0], nil
}
func (s *Service) GetSummaryBatch(ctx context.Context, owner, id string) (tm.SummaryBatchDetails, error) {
	n, e := positiveResource(id)
	if e != nil {
		return tm.SummaryBatchDetails{}, e
	}
	return s.reads.ReadSummaryBatch(ctx, owner, n)
}
func (s *Service) GetSubscriptionSummary(ctx context.Context, admin, id string) (tm.SubscriptionSummary, error) {
	if id == "" {
		return tm.SubscriptionSummary{}, status.Error(codes.InvalidArgument, "subscription required")
	}
	p, e := s.reads.ReadSubscriptionSummaries(ctx, admin, tm.AdminSubscriptionFilter{ID: id}, tm.ReadPage{Limit: 1})
	if e != nil {
		return tm.SubscriptionSummary{}, e
	}
	return p.Summaries[0], nil
}
func (s *Service) GetRuntimeStatus(ctx context.Context, admin string) (tm.RuntimeStatus, error) {
	var clock tm.ObservationClock
	if s.deps.Projector != nil {
		clock = s.deps.Projector.ObservationClock()
	}
	result, err := s.reads.ReadRuntimeStatus(ctx, admin, clock)
	if err != nil {
		return tm.RuntimeStatus{}, err
	}
	epoch, queued, persisting, alive := s.deps.Collector.RawSnapshot()
	available := alive && result.CollectorConnected && strconv.FormatUint(epoch, 10) == result.CollectorEpoch
	flag := "0"
	if available {
		flag = "1"
		scope := result.CollectorEpoch
		result.Metrics = append(result.Metrics,
			tm.RuntimeMetric{Name: "raw_queue_depth", Value: strconv.Itoa(queued), Unit: "raw_logs", Kind: "gauge", ServiceEpoch: &scope},
			tm.RuntimeMetric{Name: "raw_persist_in_flight", Value: strconv.Itoa(persisting), Unit: "raw_logs", Kind: "gauge", ServiceEpoch: &scope})
	}
	result.Metrics = append(result.Metrics, tm.RuntimeMetric{Name: "raw_observation_available", Value: flag, Unit: "boolean", Kind: "gauge"})
	if s.deps.Projector != nil {
		result.Metrics = append(result.Metrics, s.deps.Projector.MetricsSnapshot()...)
	}
	return result, nil
}
func filterDigest(v any) string {
	raw, _ := json.Marshal(v)
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:])
}
func validState(v string) bool {
	switch v {
	case "", "pending_baseline", "healthy", "interrupted", "paused", "permission_disabled", "cancelled":
		return true
	}
	return false
}
func (s *Service) pageCursor(owner, kind string, size int32, filter any, token string) (tm.ReadPage, Cursor, error) {
	n, e := normalizePageSize(size)
	if e != nil {
		return tm.ReadPage{}, Cursor{}, e
	}
	c := Cursor{Version: 1, Kind: kind, PrincipalID: owner, FilterDigest: filterDigest(filter), Direction: "desc", PageSize: n}
	if kind == "part" {
		c.Direction = "asc"
	}
	p := tm.ReadPage{Limit: n + 1}
	if token != "" {
		c, e = DecodeCursor(token, s.key, owner, c.FilterDigest, kind, n)
		if e != nil {
			return p, c, e
		}
		if kind == "part" {
			number, e := positiveResource(c.ID)
			if e != nil || number > 2147483647 {
				return p, c, invalidCursor()
			}
			p.AfterIndex = int32(number)
		} else {
			at, e := time.Parse(time.RFC3339Nano, c.Time)
			if e != nil {
				return p, c, invalidCursor()
			}
			p.AfterTime = &at
			p.AfterID = c.ID
		}
	}
	return p, c, nil
}

type SubscriptionList struct {
	tm.SubscriptionReadPage
	NextCursor string
}

func (s *Service) ListSubscriptions(ctx context.Context, owner string, size int32, token string, filter tm.SubscriptionFilter) (SubscriptionList, error) {
	if filter.View == "" {
		filter.View = "current"
	}
	if (filter.View != "current" && filter.View != "cancelled") || !validState(filter.State) || filter.ID != "" {
		return SubscriptionList{}, status.Error(codes.InvalidArgument, "invalid subscription filter")
	}
	page, c, e := s.pageCursor(owner, "subscription", size, filter, token)
	if e != nil {
		return SubscriptionList{}, e
	}
	result, e := s.reads.ReadSubscriptions(ctx, owner, filter, page)
	if e != nil {
		return SubscriptionList{}, e
	}
	out := SubscriptionList{SubscriptionReadPage: result}
	if len(out.Subscriptions) > int(c.PageSize) {
		out.Subscriptions = out.Subscriptions[:c.PageSize]
		last := out.Subscriptions[len(out.Subscriptions)-1]
		c.Time = last.CreatedAt.UTC().Format(time.RFC3339Nano)
		c.ID = last.ID
		out.NextCursor, e = EncodeCursor(c, s.key)
	}
	return out, e
}

type HistoryList struct {
	tm.HistoryReadPage
	NextCursor string
}

func (s *Service) ListHistory(ctx context.Context, owner, sub string, size int32, token string) (HistoryList, error) {
	page, c, e := s.pageCursor(owner, "history", size, sub, token)
	if e != nil {
		return HistoryList{}, e
	}
	result, e := s.reads.ReadHistory(ctx, owner, sub, page)
	if e != nil {
		return HistoryList{}, e
	}
	out := HistoryList{HistoryReadPage: result}
	if len(out.Entries) > int(c.PageSize) {
		out.Entries = out.Entries[:c.PageSize]
		last := out.Entries[len(out.Entries)-1]
		c.Time = last.SortAt.UTC().Format(time.RFC3339Nano)
		c.ID = last.ID
		out.NextCursor, e = EncodeCursor(c, s.key)
	}
	return out, e
}

type SummaryPartList struct {
	tm.SummaryPartReadPage
	NextCursor string
}

func (s *Service) ListSummaryParts(ctx context.Context, owner, batch, activity string, size int32, token string) (SummaryPartList, error) {
	batchID, e := positiveResource(batch)
	if e != nil {
		return SummaryPartList{}, e
	}
	var activityID int64
	if activity != "" {
		activityID, e = positiveResource(activity)
		if e != nil {
			return SummaryPartList{}, e
		}
	}
	page, c, e := s.pageCursor(owner, "part", size, []string{batch, activity}, token)
	if e != nil {
		return SummaryPartList{}, e
	}
	result, e := s.reads.ReadSummaryParts(ctx, owner, batchID, activityID, page)
	if e != nil {
		return SummaryPartList{}, e
	}
	out := SummaryPartList{SummaryPartReadPage: result}
	if len(out.Parts) > int(c.PageSize) {
		out.Parts = out.Parts[:c.PageSize]
		c.ID = strconv.FormatInt(int64(out.Parts[len(out.Parts)-1].Index), 10)
		out.NextCursor, e = EncodeCursor(c, s.key)
	}
	return out, e
}

type SubscriptionSummaryList struct {
	tm.AdminSubscriptionReadPage
	NextCursor string
}

func (s *Service) ListSubscriptionSummaries(ctx context.Context, admin string, size int32, token string, filter tm.AdminSubscriptionFilter) (SubscriptionSummaryList, error) {
	if !validState(filter.State) || filter.ID != "" {
		return SubscriptionSummaryList{}, status.Error(codes.InvalidArgument, "invalid summary filter")
	}
	page, c, e := s.pageCursor(admin, "admin_subscription", size, filter, token)
	if e != nil {
		return SubscriptionSummaryList{}, e
	}
	result, e := s.reads.ReadSubscriptionSummaries(ctx, admin, filter, page)
	if e != nil {
		return SubscriptionSummaryList{}, e
	}
	out := SubscriptionSummaryList{AdminSubscriptionReadPage: result}
	if len(out.Summaries) > int(c.PageSize) {
		out.Summaries = out.Summaries[:c.PageSize]
		last := out.Summaries[len(out.Summaries)-1]
		c.Time = last.CreatedAt.UTC().Format(time.RFC3339Nano)
		c.ID = last.SubscriptionID
		out.NextCursor, e = EncodeCursor(c, s.key)
	}
	return out, e
}

type ActivityList struct {
	tm.ActivityReadPage
	NextCursor, RefreshCursor, SnapshotToken string
}

func (s *Service) ListActivities(ctx context.Context, owner string, size int32, token, refresh string, filter tm.ActivityFilter) (ActivityList, error) {
	if token != "" && refresh != "" {
		return ActivityList{}, invalidCursor()
	}
	n, e := normalizePageSize(size)
	if e != nil {
		return ActivityList{}, e
	}
	if filter.From != nil {
		v := filter.From.UTC()
		filter.From = &v
	}
	if filter.To != nil {
		v := filter.To.UTC()
		filter.To = &v
	}
	if filter.From != nil && filter.To != nil && !filter.From.Before(*filter.To) {
		return ActivityList{}, status.Error(codes.InvalidArgument, "from must precede to")
	}
	digest := filterDigest(filter)
	base := Cursor{Version: 1, PrincipalID: owner, FilterDigest: digest, Direction: "desc", PageSize: n}
	input := tm.ActivityReadInput{Filter: filter, Limit: n + 1}
	if token != "" || refresh != "" {
		kind, value := "activity_next", token
		if refresh != "" {
			kind, value = "activity_refresh", refresh
		}
		c, e := DecodeCursor(value, s.key, owner, digest, kind, n)
		if e != nil {
			return ActivityList{}, e
		}
		snapshot, _ := cursorNumber(c.SnapshotID)
		input.Snapshot = &snapshot
		if refresh != "" {
			input.Refresh = true
			input.Empty = c.Empty
			if !c.Empty {
				lower, _ := cursorNumber(c.LowerID)
				upper, _ := cursorNumber(c.UpperID)
				input.Lower = &lower
				input.Upper = &upper
			}
		} else {
			after, _ := cursorNumber(c.AfterID)
			input.After = &after
		}
	}
	result, e := s.reads.ReadActivities(ctx, owner, input)
	if e != nil {
		return ActivityList{}, e
	}
	out := ActivityList{ActivityReadPage: result}
	hasNext := len(out.Activities) > int(n) || result.HasOlder
	if len(out.Activities) > int(n) {
		out.Activities = out.Activities[:n]
	}
	base.SnapshotID = strconv.FormatInt(result.Snapshot, 10)
	snapshot := base
	snapshot.Kind = "activity_snapshot"
	out.SnapshotToken, e = EncodeCursor(snapshot, s.key)
	if e != nil {
		return ActivityList{}, e
	}
	refreshed := base
	refreshed.Kind = "activity_refresh"
	refreshed.Empty = len(out.Activities) == 0
	if !refreshed.Empty {
		refreshed.UpperID = strconv.FormatInt(out.Activities[0].ID, 10)
		refreshed.LowerID = strconv.FormatInt(out.Activities[len(out.Activities)-1].ID, 10)
	}
	out.RefreshCursor, e = EncodeCursor(refreshed, s.key)
	if e != nil {
		return ActivityList{}, e
	}
	if hasNext && !refreshed.Empty {
		next := base
		next.Kind = "activity_next"
		next.AfterID = refreshed.LowerID
		out.NextCursor, e = EncodeCursor(next, s.key)
	}
	return out, e
}
