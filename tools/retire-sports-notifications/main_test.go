package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOptionsDefaultToReadOnly(t *testing.T) {
	o, err := parseOptions(nil)
	require.NoError(t, err)
	require.False(t, o.Apply)
	require.Equal(t, 5*time.Minute, o.Timeout)
	o, err = parseOptions([]string{"--apply", "--timeout=2s"})
	require.NoError(t, err)
	require.True(t, o.Apply)
	require.Equal(t, 2*time.Second, o.Timeout)
}

func TestOptionsRejectInvalidOrUnexpectedArguments(t *testing.T) {
	for _, args := range [][]string{{"--timeout=0"}, {"--timeout=-1s"}, {"--timeout=never"}, {"--prefix=polymarket"}, {"surprise"}} {
		_, err := parseOptions(args)
		require.Error(t, err, "%v", args)
	}
}
