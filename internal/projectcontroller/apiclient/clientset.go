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
	NewProjectControllerClient() (utilio.Closer, ProjectControllerServiceClient, error)
}

type clientSet struct {
	address string
}

// NewProjectControllerClient creates a new project controller client.
func (c *clientSet) NewProjectControllerClient() (utilio.Closer, ProjectControllerServiceClient, error) {
	conn, err := NewConnection(c.address)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open a new connection to project controller: %w", err)
	}
	return conn, NewProjectControllerServiceClient(conn), nil
}

// NewConnection creates new connection to project controller.
func NewConnection(address string) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.NewClient(address, opts...)
	if err != nil {
		log.Errorf("Unable to connect to project controller service with address %s", address)
		return nil, err
	}
	return conn, nil
}

// NewProjectControllerClientset creates new instance of project controller Clientset.
func NewProjectControllerClientset(address string) Clientset {
	return &clientSet{address: address}
}
