package apiclient

import (
	"context"
	"fmt"
	"math"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/util/env"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

var MaxGRPCMessageSize = env.ParseNumFromEnv(common.EnvGRPCMaxSizeMB, 100, 0, math.MaxInt32) * 1024 * 1024

type Clientset interface {
	Catalog() (TokenCatalogServiceClient, error)
	Research() (TokenResearchServiceClient, error)
	Policy() (TokenPolicyServiceClient, error)
	Operations() (TokenOperationsServiceClient, error)
	CheckHealth(context.Context) (grpc_health_v1.HealthCheckResponse_ServingStatus, error)
	Close() error
}

type clientSet struct {
	conn *grpc.ClientConn
	err  error
}

func NewTokenAPIClientset(address string) Clientset {
	conn, err := NewConnection(address)
	return &clientSet{conn: conn, err: err}
}

func NewConnection(address string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Errorf("unable to connect to token API service with address %s", address)
		return nil, err
	}
	return conn, nil
}

func (c *clientSet) connection() (*grpc.ClientConn, error) {
	if c.err != nil {
		return nil, fmt.Errorf("open token API connection: %w", c.err)
	}
	if c.conn == nil {
		return nil, fmt.Errorf("token API connection is closed")
	}
	return c.conn, nil
}

func (c *clientSet) Catalog() (TokenCatalogServiceClient, error) {
	conn, err := c.connection()
	if err != nil {
		return nil, err
	}
	return NewTokenCatalogServiceClient(conn), nil
}

func (c *clientSet) Research() (TokenResearchServiceClient, error) {
	conn, err := c.connection()
	if err != nil {
		return nil, err
	}
	return NewTokenResearchServiceClient(conn), nil
}

func (c *clientSet) Policy() (TokenPolicyServiceClient, error) {
	conn, err := c.connection()
	if err != nil {
		return nil, err
	}
	return NewTokenPolicyServiceClient(conn), nil
}

func (c *clientSet) Operations() (TokenOperationsServiceClient, error) {
	conn, err := c.connection()
	if err != nil {
		return nil, err
	}
	return NewTokenOperationsServiceClient(conn), nil
}

func (c *clientSet) CheckHealth(ctx context.Context) (grpc_health_v1.HealthCheckResponse_ServingStatus, error) {
	conn, err := c.connection()
	if err != nil {
		return grpc_health_v1.HealthCheckResponse_UNKNOWN, err
	}
	resp, err := grpc_health_v1.NewHealthClient(conn).Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		return grpc_health_v1.HealthCheckResponse_UNKNOWN, err
	}
	return resp.GetStatus(), nil
}

func (c *clientSet) Close() error {
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}
