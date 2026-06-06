package components

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	appstore "github.com/useryege/athena/internal/application/store"
)

func MarkComponentRunning(ctx context.Context, store appstore.ProjectComponentStateStore, chainID int64, contract common.Address, component string, at time.Time) error {
	if store == nil || contract == (common.Address{}) {
		return nil
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	return store.UpsertProjectComponentState(ctx, appstore.ProjectComponentState{
		ChainID:         chainID,
		ProjectContract: contract,
		Component:       component,
		Status:          appstore.ProjectComponentStatusRunning,
		LastAttemptAt:   at,
	})
}

func MarkComponentSuccess(ctx context.Context, store appstore.ProjectComponentStateStore, chainID int64, contract common.Address, component string, at time.Time) error {
	if store == nil || contract == (common.Address{}) {
		return nil
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	return store.UpsertProjectComponentState(ctx, appstore.ProjectComponentState{
		ChainID:         chainID,
		ProjectContract: contract,
		Component:       component,
		Status:          appstore.ProjectComponentStatusSuccess,
		LastAttemptAt:   at,
		LastSuccessAt:   at,
	})
}

func MarkComponentFailed(ctx context.Context, store appstore.ProjectComponentStateStore, chainID int64, contract common.Address, component string, cause error, at time.Time) error {
	if store == nil || contract == (common.Address{}) {
		return nil
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	lastError := ""
	if cause != nil {
		lastError = cause.Error()
	}
	return store.UpsertProjectComponentState(ctx, appstore.ProjectComponentState{
		ChainID:         chainID,
		ProjectContract: contract,
		Component:       component,
		Status:          appstore.ProjectComponentStatusFailed,
		LastAttemptAt:   at,
		LastError:       lastError,
	})
}

func ComponentSucceeded(ctx context.Context, store appstore.ProjectComponentStateStore, chainID int64, contract common.Address, component string) (bool, error) {
	if store == nil || contract == (common.Address{}) {
		return false, nil
	}
	state, err := store.GetProjectComponentState(ctx, chainID, contract, component)
	if err != nil || state == nil {
		return false, err
	}
	return state.Status == appstore.ProjectComponentStatusSuccess && !state.LastSuccessAt.IsZero(), nil
}
