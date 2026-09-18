package store

import (
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
