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

// MaxGRPCMessageSize contains max grpc message size.
var MaxGRPCMessageSize = env.ParseNumFromEnv(common.EnvGRPCMaxSizeMB, 100, 0, math.MaxInt32) * 1024 * 1024

// Clientset represents node scanner api clients.
type Clientset interface {
	NewNodeScannerClient() (utilio.Closer, NodeScannerServiceClient, error)
}

type clientSet struct {
	address string
}

func (c *clientSet) NewNodeScannerClient() (utilio.Closer, NodeScannerServiceClient, error) {
	conn, err := NewConnection(c.address)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open a new connection to node scanner: %w", err)
	}
	return conn, NewNodeScannerServiceClient(conn), nil
}

// NewConnection creates new connection to node scanner.
func NewConnection(address string) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.NewClient(address, opts...)
	if err != nil {
		log.Errorf("unable to connect to node scanner service with address %s", address)
		return nil, err
	}
	return conn, nil
}

// NewNodeScannerClientset creates new instance of node scanner Clientset.
func NewNodeScannerClientset(address string) Clientset {
	return &clientSet{address: address}
}
