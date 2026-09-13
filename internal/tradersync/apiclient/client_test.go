package apiclient

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"github.com/useryege/athena/internal/tradersync/rpcconfig"
	"google.golang.org/grpc/metadata"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// This compile assertion also guards the targeted gogo generator adaptation.
var _ func(grpc.ClientConnInterface) TraderSyncServiceClient = NewTraderSyncServiceClient

func TestUnavailableClientCoversEveryRPCWithoutConnection(t *testing.T) {
	client := NewUnavailableClient("configuration_invalid")
	typ := reflect.TypeOf((*TraderSyncServiceClient)(nil)).Elem()
	require.Equal(t, 16, typ.NumMethod())
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		t.Run(method.Name, func(t *testing.T) {
			request := reflect.New(method.Type.In(1).Elem())
			result := reflect.ValueOf(client).MethodByName(method.Name).Call([]reflect.Value{reflect.ValueOf(context.Background()), request})
			require.True(t, result[0].IsNil())
			err := result[1].Interface().(error)
			require.Equal(t, codes.Unavailable, status.Code(err))
		})
	}
}

func TestNewClientRejectsConfigurationBeforeConnecting(t *testing.T) {
	_, _, err := NewClient(rpcconfig.Client{})
	require.Error(t, err)
	start := time.Now()
	_, closer, err := NewClient(rpcconfig.Client{Address: "127.0.0.1:1", Transport: "loopback-insecure", Token: clientToken, MaxMessageBytes: 1024})
	require.NoError(t, err)
	require.Less(t, time.Since(start), time.Second)
	require.NoError(t, closer.Close())
}

const clientToken = "0123456789abcdef0123456789abcdef"

type boundaryServer struct {
	UnimplementedTraderSyncServiceServer
	get    func(context.Context, *GetActivityRequest) (*GetActivityResponse, error)
	create func(context.Context, *CreateSubscriptionRequest) (*CreateSubscriptionResponse, error)
}

func (s *boundaryServer) GetActivity(ctx context.Context, r *GetActivityRequest) (*GetActivityResponse, error) {
	return s.get(ctx, r)
}
func (s *boundaryServer) CreateSubscription(ctx context.Context, r *CreateSubscriptionRequest) (*CreateSubscriptionResponse, error) {
	return s.create(ctx, r)
}
func serveClientFixture(t *testing.T, service *boundaryServer, options ...grpc.ServerOption) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer(options...)
	RegisterTraderSyncServiceServer(server, service)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	return listener.Addr().String()
}
func TestClientBoundsBudgetsAndSendsServiceCredential(t *testing.T) {
	service := &boundaryServer{}
	check := func(ctx context.Context, budget time.Duration) {
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.InDelta(t, budget.Seconds(), time.Until(deadline).Seconds(), .2)
		require.Equal(t, []string{"Bearer " + clientToken}, metadata.ValueFromIncomingContext(ctx, "authorization"))
	}
	service.get = func(ctx context.Context, _ *GetActivityRequest) (*GetActivityResponse, error) {
		check(ctx, 5*time.Second)
		return &GetActivityResponse{Activity: &Activity{Id: "9007199254740993"}}, nil
	}
	service.create = func(ctx context.Context, _ *CreateSubscriptionRequest) (*CreateSubscriptionResponse, error) {
		check(ctx, 15*time.Second)
		return &CreateSubscriptionResponse{}, nil
	}
	address := serveClientFixture(t, service)
	client, closer, err := NewClient(rpcconfig.Client{Address: address, Transport: "loopback-insecure", Token: clientToken, MaxMessageBytes: 1024})
	require.NoError(t, err)
	defer closer.Close()
	got, err := client.GetActivity(context.Background(), &GetActivityRequest{})
	require.NoError(t, err)
	require.Equal(t, "9007199254740993", got.Activity.Id)
	_, err = client.CreateSubscription(context.Background(), &CreateSubscriptionRequest{})
	require.NoError(t, err)
}
func TestClientPropagatesShortParentDeadlineAndCancellation(t *testing.T) {
	for _, mode := range []string{"deadline", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			arrived := make(chan time.Time, 1)
			done := make(chan error, 1)
			service := &boundaryServer{get: func(ctx context.Context, _ *GetActivityRequest) (*GetActivityResponse, error) {
				deadline, _ := ctx.Deadline()
				arrived <- deadline
				<-ctx.Done()
				done <- ctx.Err()
				return nil, status.FromContextError(ctx.Err()).Err()
			}}
			client, closer, err := NewClient(rpcconfig.Client{Address: serveClientFixture(t, service), Transport: "loopback-insecure", Token: clientToken, MaxMessageBytes: 1024})
			require.NoError(t, err)
			defer closer.Close()
			duration := time.Second
			if mode == "deadline" {
				duration = 50 * time.Millisecond
			}
			ctx, cancel := context.WithTimeout(context.Background(), duration)
			defer cancel()
			want, _ := ctx.Deadline()
			result := make(chan error, 1)
			go func() { _, err := client.GetActivity(ctx, &GetActivityRequest{}); result <- err }()
			select {
			case got := <-arrived:
				require.WithinDuration(t, want, got, 10*time.Millisecond)
			case <-time.After(time.Second):
				t.Fatal("request did not reach server")
			}
			if mode == "cancel" {
				cancel()
			}
			select {
			case err := <-result:
				if mode == "cancel" {
					require.Equal(t, codes.Canceled, status.Code(err))
				} else {
					require.Equal(t, codes.DeadlineExceeded, status.Code(err))
				}
			case <-time.After(time.Second):
				t.Fatal("client cancellation blocked")
			}
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("handler did not receive cancellation")
			}
		})
	}
}
func TestClientDoesNotRetryWritesAndEnforcesMessageSize(t *testing.T) {
	var calls atomic.Int32
	service := &boundaryServer{create: func(context.Context, *CreateSubscriptionRequest) (*CreateSubscriptionResponse, error) {
		calls.Add(1)
		return nil, status.Error(codes.Unavailable, "reply unavailable")
	}, get: func(context.Context, *GetActivityRequest) (*GetActivityResponse, error) {
		return &GetActivityResponse{Activity: &Activity{PositionId: strings.Repeat("9", 2048)}}, nil
	}}
	client, closer, err := NewClient(rpcconfig.Client{Address: serveClientFixture(t, service), Transport: "loopback-insecure", Token: clientToken, MaxMessageBytes: 1024})
	require.NoError(t, err)
	defer closer.Close()
	_, err = client.CreateSubscription(context.Background(), &CreateSubscriptionRequest{})
	require.Equal(t, codes.Unavailable, status.Code(err))
	require.Equal(t, int32(1), calls.Load())
	_, err = client.GetActivity(context.Background(), &GetActivityRequest{})
	require.Equal(t, codes.ResourceExhausted, status.Code(err))
	_, err = client.CreateSubscription(context.Background(), &CreateSubscriptionRequest{ConfirmationToken: strings.Repeat("x", 2048)})
	require.Equal(t, codes.ResourceExhausted, status.Code(err))
	require.Equal(t, int32(1), calls.Load())
}

func tlsFiles(t *testing.T) (certFile, keyFile string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "trader-sync.test"}, DNSNames: []string{"trader-sync.test"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, IsCA: true, BasicConstraintsValid: true}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	private, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	dir := t.TempDir()
	certFile = filepath.Join(dir, "certificate.pem")
	keyFile = filepath.Join(dir, "key.pem")
	require.NoError(t, os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0600))
	require.NoError(t, os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: private}), 0600))
	return
}
func TestClientTLSVerifiesCAAndServerNameWithoutPlaintextFallback(t *testing.T) {
	certificate, key := tlsFiles(t)
	credentials, err := rpcconfig.ServerCredentials(rpcconfig.Server{ListenAddress: "127.0.0.1:8122", Transport: "tls", CertFile: certificate, KeyFile: key})
	require.NoError(t, err)
	var calls atomic.Int32
	service := &boundaryServer{get: func(context.Context, *GetActivityRequest) (*GetActivityResponse, error) {
		calls.Add(1)
		return &GetActivityResponse{Activity: &Activity{Id: "1"}}, nil
	}}
	address := serveClientFixture(t, service, grpc.Creds(credentials))
	otherCA, _ := tlsFiles(t)
	for _, tc := range []struct {
		name, ca, serverName string
		ok                   bool
	}{{"trusted", certificate, "trader-sync.test", true}, {"wrong CA", otherCA, "trader-sync.test", false}, {"wrong name", certificate, "other.test", false}, {"system trust", "", "trader-sync.test", false}} {
		t.Run(tc.name, func(t *testing.T) {
			client, closer, err := NewClient(rpcconfig.Client{Address: address, Transport: "tls", Token: clientToken, CAFile: tc.ca, ServerName: tc.serverName, MaxMessageBytes: 1024})
			require.NoError(t, err)
			defer closer.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
			defer cancel()
			_, err = client.GetActivity(ctx, &GetActivityRequest{})
			if tc.ok {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
	require.Equal(t, int32(1), calls.Load())
	plainAddress := serveClientFixture(t, service)
	cfg, err := rpcconfig.LoadClient(func(k string) (string, bool) {
		v, ok := map[string]string{"ATHENA_TRADER_SYNC_SERVER_ADDRESS": plainAddress, "ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN": clientToken, "ATHENA_TRADER_SYNC_TLS_CA_FILE": certificate, "ATHENA_TRADER_SYNC_TLS_SERVER_NAME": "trader-sync.test"}[k]
		return v, ok
	})
	require.NoError(t, err)
	client, closer, err := NewClient(cfg)
	require.NoError(t, err)
	defer closer.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	_, err = client.GetActivity(ctx, &GetActivityRequest{})
	require.Error(t, err)
	require.Equal(t, int32(1), calls.Load())
}
