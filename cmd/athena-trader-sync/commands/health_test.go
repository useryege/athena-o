package commands

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/tradersync"
	"github.com/useryege/athena/internal/tradersync/rpcconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func healthTLS(t *testing.T) (string, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	cert := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "athena-trader-sync"}, DNSNames: []string{"athena-trader-sync"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, IsCA: true, BasicConstraintsValid: true}
	der, err := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	require.NoError(t, err)
	private, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	dir := t.TempDir()
	certFile, keyFile := filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem")
	require.NoError(t, os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0600))
	require.NoError(t, os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: private}), 0600))
	return certFile, keyFile
}
func TestHealthTLSUsesCertificateNameAndOnlyHealthConfiguration(t *testing.T) {
	cert, key := healthTLS(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	creds, err := rpcconfig.ServerCredentials(rpcconfig.Server{ListenAddress: listener.Addr().String(), Transport: "tls", CertFile: cert, KeyFile: key})
	require.NoError(t, err)
	server := grpc.NewServer(grpc.Creds(creds))
	healthServer := health.NewServer()
	healthServer.SetServingStatus(tradersync.HealthServiceName, healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(server, healthServer)
	go server.Serve(listener)
	defer server.Stop()
	for _, env := range []string{"ATHENA_ACCOUNT_STATE_POSTGRES_DSN", "ATHENA_TRADER_SYNC_HTTP_URL", "ATHENA_TRADER_SYNC_WSS_URL", "ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN", "ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY"} {
		t.Setenv(env, "invalid")
	}
	execute := func(name string) error {
		cmd := NewCommand()
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetArgs([]string{"health", "--target", listener.Addr().String(), "--transport", "tls", "--tls-ca-file", cert, "--tls-server-name", name, "--timeout", "200ms"})
		return cmd.Execute()
	}
	require.NoError(t, execute("athena-trader-sync"))
	require.Error(t, execute("127.0.0.1"))
	healthServer.SetServingStatus(tradersync.HealthServiceName, healthpb.HealthCheckResponse_NOT_SERVING)
	require.ErrorContains(t, execute("athena-trader-sync"), "NOT_SERVING")
}
func TestHealthDeadlineIsBounded(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	cmd := NewCommand()
	cmd.SetArgs([]string{"health", "--target", listener.Addr().String(), "--transport", "loopback-insecure", "--timeout", "40ms"})
	started := time.Now()
	require.Error(t, cmd.Execute())
	require.Less(t, time.Since(started), time.Second)
}
