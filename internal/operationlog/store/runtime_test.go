package store

import (
	"context"
	"errors"
	"testing"
)

func TestProjectionErrorDoesNotMakeReadableSnapshotsUnready(t *testing.T) {
	s := &Store{}
	s.SetQueryReady(true)
	s.setProcessingError(errors.New("temporary projector write failure"))
	if !s.QueryReady() {
		t.Fatal("projector failure made query readiness false")
	}
	if s.LastProcessingError() == nil {
		t.Fatal("projector failure was not retained for runtime status")
	}
	s.setProcessingError(nil)
	if s.LastProcessingError() != nil {
		t.Fatal("projector recovery did not clear processing error")
	}
}

func TestProjectorReadinessRecoversAfterVerification(t *testing.T) {
	s := &Store{}
	s.SetQueryReady(false)
	verified := false
	p := &Projector{Store: s, Verify: func(context.Context) error {
		verified = true
		return nil
	}}
	if err := p.refreshQueryReadiness(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !verified || !s.QueryReady() {
		t.Fatalf("verified=%v queryReady=%v", verified, s.QueryReady())
	}
}
