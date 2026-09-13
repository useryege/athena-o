package commands

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// Returning either loop must cancel and join the other, preserving its failure.
func TestRunDiscoveryJoinsScannerAndEnricher(t *testing.T) {
	started := make(chan struct{})
	stopped := make(chan struct{})
	failure := errors.New("scanner failure")
	err := runDiscovery(context.Background(), func(context.Context) error { <-started; return failure }, func(ctx context.Context) error { close(started); <-ctx.Done(); close(stopped); return nil })
	require.ErrorIs(t, err, failure)
	select {
	case <-stopped:
	default:
		t.Fatal("enricher was not joined")
	}
}
func TestRunDiscoveryCancelsBothLoops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var stopped = make(chan struct{}, 2)
	loop := func(ctx context.Context) error { <-ctx.Done(); stopped <- struct{}{}; return nil }
	done := make(chan error, 1)
	go func() { done <- runDiscovery(ctx, loop, loop) }()
	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("loops did not stop")
	}
	require.Len(t, stopped, 2)
}
