package apiclient

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	api "github.com/useryege/athena/pkg/apiclient/solana"
	utilgrpc "github.com/useryege/athena/util/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

const InternalAuthTokenEnv = "ATHENA_SOLANA_DISCOVERY_INTERNAL_AUTH_TOKEN"

type Clientset interface {
	Solana() api.SolanaServiceClient
	CheckHealth(context.Context) (grpc_health_v1.HealthCheckResponse_ServingStatus, error)
	Close() error
}

type clientSet struct {
	connection *utilgrpc.ClientConnection
	client     api.SolanaServiceClient
}

func NewSolanaClientset(address, internalToken string) (Clientset, error) {
	token := strings.TrimSpace(internalToken)
	if len(token) < 32 || strings.IndexFunc(token, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
		return nil, fmt.Errorf("Solana internal auth token must contain at least 32 bytes and no whitespace")
	}
	connection, err := utilgrpc.NewClientConnection(address, grpc.WithPerRPCCredentials(internalBearerCredentials{authorization: "Bearer " + token}))
	if err != nil {
		return nil, err
	}
	return &clientSet{connection: connection, client: api.NewSolanaServiceClient(connection.ClientConn())}, nil
}

func (c *clientSet) Solana() api.SolanaServiceClient { return c.client }
func (c *clientSet) CheckHealth(ctx context.Context) (grpc_health_v1.HealthCheckResponse_ServingStatus, error) {
	return c.connection.CheckHealth(ctx)
}
func (c *clientSet) Close() error { return c.connection.Close() }

type internalBearerCredentials struct{ authorization string }

func (c internalBearerCredentials) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"authorization": c.authorization}, nil
}
func (internalBearerCredentials) RequireTransportSecurity() bool { return false }
