package notification

import (
	"github.com/jackc/pgx/v5/pgxpool"
	notificationstore "github.com/useryege/athena/internal/notification/store"
	"strings"
	"testing"
)

func TestServerConfiguresSummariesOnBorrowedPoolBeforeStart(t *testing.T) {
	pool := &pgxpool.Pool{}
	s, err := NewServer(ServerOpts{Store: notificationstore.NewSQLStore(pool), InternalAuthToken: strings.Repeat("a", 32), SiteURL: "https://athena.test"})
	if err != nil {
		t.Fatal(err)
	}
	if s.service.summarySource == nil {
		t.Fatal("production server omitted summary source")
	}
	if s.service.summarySource.pool != pool || len(s.service.workSources()) != 4 {
		t.Fatal("summary does not share the existing pool and dispatcher")
	}
	if err := s.Stop(); err != nil {
		t.Fatal(err)
	} // Borrowed pool must not be closed.
	for _, opts := range []ServerOpts{
		{InternalAuthToken: strings.Repeat("a", 32), SiteURL: "https://athena.test"},
		{Store: notificationstore.NewSQLStore(pool), InternalAuthToken: strings.Repeat("a", 32)},
	} {
		if _, err := NewServer(opts); err == nil {
			t.Fatal("missing composition dependency accepted")
		}
	}
}
