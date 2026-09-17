package moduleaccess

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/status"
	"testing"
	"time"
)

type testStore struct {
	Store
	open     bool
	err      error
	reads    int
	deadline time.Time
}

func (s *testStore) GetModuleAccessSetting(ctx context.Context, k Key) (Setting, error) {
	s.reads++
	s.deadline, _ = ctx.Deadline()
	return Setting{Key: k, Open: s.open}, s.err
}
func TestCheckReadsEveryRequestAndFailsClosed(t *testing.T) {
	s := &testStore{}
	err := Check(context.Background(), s, Worm)
	require.Equal(t, ClosedReason, status.Convert(err).Message())
	info := status.Convert(err).Details()[0].(*errdetails.ErrorInfo)
	require.Equal(t, Domain, info.Domain)
	require.Equal(t, "worm", info.Metadata["module_key"])
	s.open = true
	require.NoError(t, Check(context.Background(), s, Worm))
	s.err = errors.New("database unavailable")
	require.Equal(t, UnavailableReason, status.Convert(Check(context.Background(), s, Worm)).Message())
	require.Equal(t, 3, s.reads)
	require.WithinDuration(t, time.Now().Add(2*time.Second), s.deadline, time.Second)
	require.Error(t, Check(context.Background(), s, Key("token")))
	require.Equal(t, 3, s.reads)
}
func TestCheckPreservesShorterDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()
	s := &testStore{open: true}
	require.NoError(t, Check(ctx, s, Worm))
	expected, _ := ctx.Deadline()
	require.Equal(t, expected, s.deadline)
}
