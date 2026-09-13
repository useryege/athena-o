package server

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"net"
	"testing"
)

func TestListenersCloseJoinsGatewayWhenMainAlreadyClosed(t *testing.T) {
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	conn, e := grpc.Dial(listener.Addr().String(), grpc.WithInsecure())
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close()
	listener.Close()
	l := &Listeners{Main: listener, GatewayConn: conn}
	if e = l.Close(); e != nil {
		t.Fatal("already-closed main hid gateway cleanup", e)
	}
	if conn.GetState() != connectivity.Shutdown {
		t.Fatal("gateway remains open")
	}
	if e = l.Close(); e != nil {
		t.Fatal(e)
	}
}
