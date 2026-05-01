package apiclient

import (
	fmt "fmt"
	math "math"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/util/env"
	utilio "github.com/useryege/athena/util/io"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// MaxGRPCMessageSize contains max grpc message size
var MaxGRPCMessageSize = env.ParseNumFromEnv(common.EnvGRPCMaxSizeMB, 100, 0, math.MaxInt32) * 1024 * 1024

// Clientset represents project controller api clients.
type Clientset interface {
	NewApplicationServiceClient() (utilio.Closer, ApplicationServiceClient, error)
}

type clientSet struct {
	address string
}

// NewApplicationServiceClient creates a new application client.
func (c *clientSet) NewApplicationServiceClient() (utilio.Closer, ApplicationServiceClient, error) {
	conn, err := NewConnection(c.address)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open a new connection to chain watcher: %w", err)
	}
	return conn, NewApplicationServiceClient(conn), nil
}

// NewConnection creates new connection to project controller.
func NewConnection(address string) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.NewClient(address, opts...)
	if err != nil {
		log.Errorf("Unable to connect to chain watcher service with address %s", address)
		return nil, err
	}
	return conn, nil
}

// NewApplicationClientset creates new instance of application Clientset.
func NewApplicationClientset(address string) Clientset {
	return &clientSet{address: address}
}
