//go:build integration

package store

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
)

var testRuntimeOwners = struct {
	sync.Mutex
	tokens map[string]RuntimeToken
}{tokens: map[string]RuntimeToken{}}

// All pools for one isolated test database borrow a single real runtime owner.
// Explicit ownership/takeover tests construct and close their own sessions.
func runtimeTestStore(t *testing.T, pool *pgxpool.Pool) *SQLStore {
	t.Helper()
	key := pool.Config().ConnConfig.Database
	testRuntimeOwners.Lock()
	defer testRuntimeOwners.Unlock()
	token, ok := testRuntimeOwners.tokens[key]
	if !ok {
		owner, err := NewSQLStore(pool).AcquireRuntimeSession(context.Background())
		require.NoError(t, err)
		token = owner.RuntimeToken()
		testRuntimeOwners.tokens[key] = token
		t.Cleanup(func() {
			require.NoError(t, owner.CloseAfterWorkers(context.Background()))
			testRuntimeOwners.Lock()
			delete(testRuntimeOwners.tokens, key)
			testRuntimeOwners.Unlock()
		})
	}
	gate, err := NewRuntimeWriteGate(pool, token)
	require.NoError(t, err)
	result, err := NewRuntimeSQLStore(pool, gate)
	require.NoError(t, err)
	return result
}

func bindTestRuntime(t *testing.T, s *SQLStore, owner *RuntimeSession) *SQLStore {
	t.Helper()
	result, err := s.WithRuntime(owner.RuntimeToken())
	require.NoError(t, err)
	return result
}
