package server

import (
	"net"

	"google.golang.org/grpc"
)

type Listeners struct {
	Main        net.Listener
	Metrics     net.Listener
	GatewayConn *grpc.ClientConn
}

func (l *Listeners) Close() error {
	if l.Main != nil {
		if err := l.Main.Close(); err != nil {
			return err
		}
		l.Main = nil
	}
	if l.Metrics != nil {
		if err := l.Metrics.Close(); err != nil {
			return err
		}
		l.Metrics = nil
	}
	if l.GatewayConn != nil {
		if err := l.GatewayConn.Close(); err != nil {
			return err
		}
		l.GatewayConn = nil
	}
	return nil
}
