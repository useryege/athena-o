// Package query contains stable operation-log query contracts and adapters.
package query

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/useryege/athena/internal/operationlog/event"
)

var (
	ErrInvalidFilter   = errors.New("invalid operation log filter")
	ErrCursorInvalid   = errors.New("cursor invalid")
	ErrCursorExpired   = errors.New("cursor expired")
	ErrSnapshotExpired = errors.New("snapshot expired")
	ErrNotFound        = errors.New("operation log not found")
)

const (
	DefaultPageSize = 50
	MaxPageSize     = 100
	MaxRange        = 90 * 24 * time.Hour
	CursorTTL       = 30 * time.Minute
)

type Filter struct {
	From            *time.Time
	To              *time.Time
	ActorQuery      string
	ActorRole       string
	CredentialKind  string
	ModuleCode      string
	ActionCode      string
	Outcome         string
	TargetAccountID string
	ResourceType    string
	ResourceID      string
	PageSize        int
	Cursor          string
}

type NormalizedFilter struct {
	From            time.Time `json:"from"`
	To              time.Time `json:"to"`
	ActorQuery      string    `json:"actorQuery,omitempty"`
	ActorRole       string    `json:"actorRole,omitempty"`
	CredentialKind  string    `json:"credentialKind,omitempty"`
	ModuleCode      string    `json:"moduleCode,omitempty"`
	ActionCode      string    `json:"actionCode,omitempty"`
	Outcome         string    `json:"outcome,omitempty"`
	TargetAccountID string    `json:"targetAccountId,omitempty"`
	ResourceType    string    `json:"resourceType,omitempty"`
	ResourceID      string    `json:"resourceId,omitempty"`
	PageSize        int       `json:"pageSize"`
}

func NormalizeFilter(in Filter, now time.Time) (NormalizedFilter, error) {
	now = now.UTC()
	out := NormalizedFilter{ActorQuery: strings.TrimSpace(in.ActorQuery), ActorRole: strings.TrimSpace(in.ActorRole), CredentialKind: strings.TrimSpace(in.CredentialKind), ModuleCode: strings.TrimSpace(in.ModuleCode), ActionCode: strings.TrimSpace(in.ActionCode), Outcome: strings.TrimSpace(in.Outcome), TargetAccountID: strings.TrimSpace(in.TargetAccountID), ResourceType: strings.TrimSpace(in.ResourceType), ResourceID: strings.TrimSpace(in.ResourceID), PageSize: in.PageSize}
	if out.ActorRole == "" {
		out.ActorRole = "ALL"
	}
	if out.CredentialKind == "" {
		out.CredentialKind = "ALL"
	}
	if out.Outcome == "" {
		out.Outcome = "ALL"
	}
	if in.From == nil && in.To == nil {
		out.From, out.To = now.Add(-7*24*time.Hour), now
	} else if in.From == nil || in.To == nil {
		return NormalizedFilter{}, fmt.Errorf("%w: from and to must be supplied together", ErrInvalidFilter)
	} else {
		out.From, out.To = in.From.UTC(), in.To.UTC()
	}
	if !out.From.Before(out.To) || out.To.Sub(out.From) > MaxRange {
		return NormalizedFilter{}, fmt.Errorf("%w: invalid time range", ErrInvalidFilter)
	}
	if out.ActorRole != "" && out.ActorRole != "ALL" && out.ActorRole != "MEMBER" && out.ActorRole != "ADMINISTRATOR" && out.ActorRole != "UNKNOWN" {
		return NormalizedFilter{}, fmt.Errorf("%w: actor_role", ErrInvalidFilter)
	}
	if out.CredentialKind != "" && out.CredentialKind != "ALL" && out.CredentialKind != "LOGIN_SESSION" && out.CredentialKind != "DEVELOPMENT" && out.CredentialKind != "API_KEY" && out.CredentialKind != "UNAUTHENTICATED" {
		return NormalizedFilter{}, fmt.Errorf("%w: credential_kind", ErrInvalidFilter)
	}
	if out.Outcome != "" && out.Outcome != "ALL" && out.Outcome != "UNKNOWN" && out.Outcome != "SUCCEEDED" && out.Outcome != "ACCEPTED" && out.Outcome != "FAILED" && out.Outcome != "DENIED" && out.Outcome != "PARTIAL" && out.Outcome != "ACTION_REQUIRED" && out.Outcome != "CANCELLED" {
		return NormalizedFilter{}, fmt.Errorf("%w: outcome", ErrInvalidFilter)
	}
	if out.PageSize == 0 {
		out.PageSize = DefaultPageSize
	}
	if out.PageSize < 1 || out.PageSize > MaxPageSize {
		return NormalizedFilter{}, fmt.Errorf("%w: page_size must be 1..100", ErrInvalidFilter)
	}
	if len([]byte(out.ActorQuery)) > 128 {
		return NormalizedFilter{}, fmt.Errorf("%w: actor_query too long", ErrInvalidFilter)
	}
	if out.TargetAccountID != "" {
		u, err := uuid.Parse(out.TargetAccountID)
		if err != nil || u.String() != out.TargetAccountID {
			return NormalizedFilter{}, fmt.Errorf("%w: target_account_id", ErrInvalidFilter)
		}
	}
	if out.ModuleCode != "" || out.ActionCode != "" {
		if out.ActionCode != "" {
			e, ok := event.Action(out.ActionCode)
			if !ok || (out.ModuleCode != "" && e.ModuleCode != out.ModuleCode) {
				return NormalizedFilter{}, fmt.Errorf("%w: action/module", ErrInvalidFilter)
			}
		} else {
			found := false
			for _, e := range event.Catalog() {
				if e.ModuleCode == out.ModuleCode {
					found = true
					break
				}
			}
			if !found {
				return NormalizedFilter{}, fmt.Errorf("%w: module_code", ErrInvalidFilter)
			}
		}
	}
	if (out.ResourceType == "") != (out.ResourceID == "") {
		return NormalizedFilter{}, fmt.Errorf("%w: resource_type and resource_id are paired", ErrInvalidFilter)
	}
	if len([]byte(out.ResourceID)) > 128 {
		return NormalizedFilter{}, fmt.Errorf("%w: resource_id too long", ErrInvalidFilter)
	}
	return out, nil
}
