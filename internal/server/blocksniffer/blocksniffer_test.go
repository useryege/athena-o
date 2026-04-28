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

func TestServer_SetNodeGrpcURL(t *testing.T) {
	ctx := context.Background()

	t.Run("reject empty URL", func(t *testing.T) {
		srv := NewServer()
		req := &blocksnifferpkg.SetNodeGrpcURLRequest{NodeGrpcUrl: ""}
		_, err := srv.SetNodeGrpcURL(ctx, req)
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), "node_grpc_url is required")
	})

	t.Run("reject whitespace-only URL", func(t *testing.T) {
		srv := NewServer()
		req := &blocksnifferpkg.SetNodeGrpcURLRequest{NodeGrpcUrl: "   \t  "}
		_, err := srv.SetNodeGrpcURL(ctx, req)
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), "node_grpc_url is required")
	})

	t.Run("reject invalid URL", func(t *testing.T) {
		srv := NewServer()
		req := &blocksnifferpkg.SetNodeGrpcURLRequest{NodeGrpcUrl: "not-a-url"}
		_, err := srv.SetNodeGrpcURL(ctx, req)
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), "invalid node_grpc_url")
	})

	t.Run("accept valid URL and persist", func(t *testing.T) {
		srv := NewServer()
		want := "http://127.0.0.1:9090"
		req := &blocksnifferpkg.SetNodeGrpcURLRequest{NodeGrpcUrl: "  " + want + "  "}
		resp, err := srv.SetNodeGrpcURL(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, want, resp.GetNodeGrpcUrl())

		getResp, err := srv.GetNodeGrpcURL(ctx, &blocksnifferpkg.GetNodeGrpcURLRequest{})
		require.NoError(t, err)
		require.NotNil(t, getResp)
		assert.Equal(t, want, getResp.GetNodeGrpcUrl())
	})

	t.Run("accept ws URL with host and port", func(t *testing.T) {
		srv := NewServer()
		want := "ws://65.108.75.55:8546"
		resp, err := srv.SetNodeGrpcURL(ctx, &blocksnifferpkg.SetNodeGrpcURLRequest{NodeGrpcUrl: want})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, want, resp.GetNodeGrpcUrl())

		getResp, err := srv.GetNodeGrpcURL(ctx, &blocksnifferpkg.GetNodeGrpcURLRequest{})
		require.NoError(t, err)
		require.NotNil(t, getResp)
		assert.Equal(t, want, getResp.GetNodeGrpcUrl())
	})
}

func TestServer_GetNodeGrpcURL(t *testing.T) {
	ctx := context.Background()

	t.Run("empty when unset", func(t *testing.T) {
		srv := NewServer()
		resp, err := srv.GetNodeGrpcURL(ctx, &blocksnifferpkg.GetNodeGrpcURLRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "", resp.GetNodeGrpcUrl())
	})

	t.Run("returns last set value", func(t *testing.T) {
		srv := NewServer()
		want := "https://example.com:443"
		_, err := srv.SetNodeGrpcURL(ctx, &blocksnifferpkg.SetNodeGrpcURLRequest{NodeGrpcUrl: want})
		require.NoError(t, err)

		resp, err := srv.GetNodeGrpcURL(ctx, &blocksnifferpkg.GetNodeGrpcURLRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, want, resp.GetNodeGrpcUrl())
	})
}

func TestServer_SetNodeGrpcURL_Overwrite(t *testing.T) {
	ctx := context.Background()
	srv := NewServer()

	first := "http://127.0.0.1:9090"
	_, err := srv.SetNodeGrpcURL(ctx, &blocksnifferpkg.SetNodeGrpcURLRequest{NodeGrpcUrl: first})
	require.NoError(t, err)

	second := "http://192.0.2.1:50051"
	_, err = srv.SetNodeGrpcURL(ctx, &blocksnifferpkg.SetNodeGrpcURLRequest{NodeGrpcUrl: second})
	require.NoError(t, err)

	resp, err := srv.GetNodeGrpcURL(ctx, &blocksnifferpkg.GetNodeGrpcURLRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, second, resp.GetNodeGrpcUrl())
}
