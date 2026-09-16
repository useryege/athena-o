package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOptionsDefaultToReadOnlyBoundedWormRetirement(t *testing.T) {
	o, err := parseOptions(nil)
	require.NoError(t, err)
	require.False(t, o.Apply)
	require.Equal(t, 2*time.Minute, o.Timeout)
	require.EqualValues(t, 100, o.BatchSize)

	o, err = parseOptions([]string{"--apply", "--timeout=3s", "--batch-size=7"})
	require.NoError(t, err)
	require.True(t, o.Apply)
	require.Equal(t, 3*time.Second, o.Timeout)
	require.EqualValues(t, 7, o.BatchSize)
}

func TestOptionsRejectInvalidAndArbitrarySourceSelection(t *testing.T) {
	for _, args := range [][]string{
		{"--timeout=0"},
		{"--timeout=-1s"},
		{"--timeout=never"},
		{"--batch-size=0"},
		{"--batch-size=-1"},
		{"--batch-size=2147483648"},
		{"--prefix=worm"},
		{"--source=worm-markets.new-event"},
		{"surprise"},
	} {
		_, err := parseOptions(args)
		require.Error(t, err, "%v", args)
	}
}
