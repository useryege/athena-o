package commands

import (
	"context"
	"errors"
	"google.golang.org/grpc"
	"net"
	"testing"
	"time"
)

func TestRunStopsScannerAndRPCOnCancellation(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	stopped := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- serve(ctx, l, grpc.NewServer(), func(c context.Context) error { close(started); <-c.Done(); close(stopped); return c.Err() })
	}()
	<-started
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown did not finish")
	}
	select {
	case <-stopped:
	default:
		t.Fatal("scanner still running")
	}
	conn, err := net.DialTimeout("tcp", l.Addr().String(), 100*time.Millisecond)
	if err == nil {
		conn.Close()
		t.Fatal("RPC listener still accepting")
	}
}

func TestScannerFatalErrorStopsRPC(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	want := errors.New("wrong network")
	err = serve(context.Background(), l, grpc.NewServer(), func(context.Context) error { return want })
	if !errors.Is(err, want) {
		t.Fatalf("got %v, want fatal scanner error", err)
	}
}

// Resource release is part of shutdown and must run after the scanner exits.
func TestServeClosesResourcesAfterCanceledScanner(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	closed := false
	cancel()
	err = serveWithCleanup(ctx, l, grpc.NewServer(), func(ctx context.Context) error { <-ctx.Done(); close(stopped); return ctx.Err() }, func() {
		select {
		case <-stopped:
			closed = true
		default:
			t.Error("resource closed before scanner stopped")
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if !closed {
		t.Fatal("resources not closed")
	}
}
