package retirement

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunLoopReadOnlyReportsWithoutWriting(t *testing.T) {
	retireCalls := 0
	result, err := runLoop(
		context.Background(), false, 100,
		func(context.Context) (Snapshot, error) {
			return Snapshot{RetiredTotal: 7, Pending: 2, Sending: 1}, nil
		},
		func(context.Context, int32) (Batch, error) {
			retireCalls++
			return Batch{}, nil
		},
		func(context.Context) error { return nil },
	)
	require.NoError(t, err)
	require.Equal(t, "read_only", result.Status)
	require.True(t, result.CountsVerified)
	require.EqualValues(t, 7, result.RetiredTotal)
	require.EqualValues(t, 2, result.Pending)
	require.EqualValues(t, 1, result.Sending)
	require.Zero(t, result.Cancelled)
	require.Zero(t, retireCalls)
}

func TestRunLoopAccumulatesBatchChangesAndUsesLatestSnapshot(t *testing.T) {
	snapshots := []Snapshot{{Pending: 2}, {RetiredTotal: 1, Pending: 1}, {RetiredTotal: 2}}
	countCalls := 0
	result, err := runLoop(
		context.Background(), true, 1,
		func(context.Context) (Snapshot, error) {
			r := snapshots[countCalls]
			countCalls++
			return r, nil
		},
		func(context.Context, int32) (Batch, error) { return Batch{Cancelled: 1}, nil },
		func(context.Context) error { return nil },
	)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)
	require.True(t, result.CountsVerified)
	require.EqualValues(t, 2, result.Cancelled)
	require.EqualValues(t, 2, result.RetiredTotal)
	require.Zero(t, result.Pending)
	require.Zero(t, result.Sending)
	require.Equal(t, 3, countCalls)
}

func TestRunLoopKeepsCommittedCancellationWhenRefreshFails(t *testing.T) {
	countCalls := 0
	refreshErr := errors.New("refresh failed")
	result, err := runLoop(
		context.Background(), true, 100,
		func(context.Context) (Snapshot, error) {
			countCalls++
			if countCalls == 1 {
				return Snapshot{Pending: 1}, nil
			}
			return Snapshot{}, refreshErr
		},
		func(context.Context, int32) (Batch, error) { return Batch{Cancelled: 1}, nil },
		func(context.Context) error { return nil },
	)
	require.ErrorIs(t, err, refreshErr)
	require.Equal(t, "incomplete", result.Status)
	require.EqualValues(t, 1, result.Cancelled)
	require.False(t, result.CountsVerified)
	require.EqualValues(t, 1, result.Pending)
}

func TestRunLoopTimeoutReturnsLastVerifiedRemainingCounts(t *testing.T) {
	deadlineErr := context.DeadlineExceeded
	result, err := runLoop(
		context.Background(), true, 100,
		func(context.Context) (Snapshot, error) { return Snapshot{RetiredTotal: 3, Sending: 1}, nil },
		func(context.Context, int32) (Batch, error) { return Batch{}, nil },
		func(context.Context) error { return deadlineErr },
	)
	require.ErrorIs(t, err, deadlineErr)
	require.Equal(t, "incomplete", result.Status)
	require.True(t, result.CountsVerified)
	require.EqualValues(t, 3, result.RetiredTotal)
	require.EqualValues(t, 1, result.Sending)
}
