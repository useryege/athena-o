package blocksniffer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	blocksnifferpkg "github.com/useryege/athena/pkg/apiclient/blocksniffer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestServer_SetEvmNodeWsURL(t *testing.T) {
	ctx := context.Background()

	t.Run("reject empty URL", func(t *testing.T) {
		srv := NewServer()
		req := &blocksnifferpkg.SetEvmNodeWsURLRequest{EvmNodeWsURL: ""}
		_, err := srv.SetEvmNodeWsURL(ctx, req)
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), "evm_node_ws_url is required")
	})

	t.Run("reject whitespace-only URL", func(t *testing.T) {
		srv := NewServer()
		req := &blocksnifferpkg.SetEvmNodeWsURLRequest{EvmNodeWsURL: "   \t  "}
		_, err := srv.SetEvmNodeWsURL(ctx, req)
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), "evm_node_ws_url is required")
	})

	t.Run("reject invalid URL", func(t *testing.T) {
		srv := NewServer()
		req := &blocksnifferpkg.SetEvmNodeWsURLRequest{EvmNodeWsURL: "not-a-url"}
		_, err := srv.SetEvmNodeWsURL(ctx, req)
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), "invalid evm_node_ws_url")
	})

	t.Run("accept valid URL and persist", func(t *testing.T) {
		srv := NewServer()
		want := "http://127.0.0.1:9090"
		req := &blocksnifferpkg.SetEvmNodeWsURLRequest{EvmNodeWsURL: "  " + want + "  "}
		resp, err := srv.SetEvmNodeWsURL(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, want, resp.GetEvmNodeWsURL())

		getResp, err := srv.GetEvmNodeWsURL(ctx, &blocksnifferpkg.GetEvmNodeWsURLRequest{})
		require.NoError(t, err)
		require.NotNil(t, getResp)
		assert.Equal(t, want, getResp.GetEvmNodeWsURL())
	})

	t.Run("accept ws URL with host and port", func(t *testing.T) {
		srv := NewServer()
		want := "ws://65.108.75.55:8546"
		resp, err := srv.SetEvmNodeWsURL(ctx, &blocksnifferpkg.SetEvmNodeWsURLRequest{EvmNodeWsURL: want})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, want, resp.GetEvmNodeWsURL())

		getResp, err := srv.GetEvmNodeWsURL(ctx, &blocksnifferpkg.GetEvmNodeWsURLRequest{})
		require.NoError(t, err)
		require.NotNil(t, getResp)
		assert.Equal(t, want, getResp.GetEvmNodeWsURL())
	})
}

func TestServer_GetEvmNodeWsURL(t *testing.T) {
	ctx := context.Background()

	t.Run("empty when unset", func(t *testing.T) {
		srv := NewServer()
		resp, err := srv.GetEvmNodeWsURL(ctx, &blocksnifferpkg.GetEvmNodeWsURLRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "", resp.GetEvmNodeWsURL())
	})

	t.Run("returns last set value", func(t *testing.T) {
		srv := NewServer()
		want := "https://example.com:443"
		_, err := srv.SetEvmNodeWsURL(ctx, &blocksnifferpkg.SetEvmNodeWsURLRequest{EvmNodeWsURL: want})
		require.NoError(t, err)

		resp, err := srv.GetEvmNodeWsURL(ctx, &blocksnifferpkg.GetEvmNodeWsURLRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, want, resp.GetEvmNodeWsURL())
	})
}

func TestServer_SetEvmNodeWsURL_Overwrite(t *testing.T) {
	ctx := context.Background()
	srv := NewServer()

	first := "http://127.0.0.1:9090"
	_, err := srv.SetEvmNodeWsURL(ctx, &blocksnifferpkg.SetEvmNodeWsURLRequest{EvmNodeWsURL: first})
	require.NoError(t, err)

	second := "http://192.0.2.1:50051"
	_, err = srv.SetEvmNodeWsURL(ctx, &blocksnifferpkg.SetEvmNodeWsURLRequest{EvmNodeWsURL: second})
	require.NoError(t, err)

	resp, err := srv.GetEvmNodeWsURL(ctx, &blocksnifferpkg.GetEvmNodeWsURLRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, second, resp.GetEvmNodeWsURL())
}
