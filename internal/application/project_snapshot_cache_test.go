package application

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/redis/go-redis/v9"
	"github.com/useryege/athena/internal/application/redisport"
)

func TestRedisProjectSnapshotCacheListActiveProjectsPage(t *testing.T) {
	ctx := context.Background()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cache := NewProjectSnapshotCache(redisport.NewGoRedisAdapter(client))

	for i := 1; i <= 5; i++ {
		contract := common.BigToAddress(big.NewInt(int64(i)))
		if err := cache.SetProject(ctx, &Project{Meta: ProjectMeta{
			BlockNumber: uint64(i),
			Contract:    contract,
			TxIndex:     uint64(i),
			SourceCode:  "contract Source {}",
		}}); err != nil {
			t.Fatalf("set project %d: %v", i, err)
		}
	}

	projects, total, page, pageSize, err := cache.ListActiveProjectsPage(ctx, 2, 2)
	if err != nil {
		t.Fatalf("list active projects page: %v", err)
	}
	if total != 5 || page != 2 || pageSize != 2 {
		t.Fatalf("pagination = total %d page %d pageSize %d, want 5/2/2", total, page, pageSize)
	}
	if len(projects) != 2 {
		t.Fatalf("len(projects) = %d, want 2", len(projects))
	}
	if got, want := projects[0].Meta.Contract, common.BigToAddress(big.NewInt(3)); got != want {
		t.Fatalf("projects[0].contract = %s, want %s", got, want)
	}
	if got, want := projects[1].Meta.Contract, common.BigToAddress(big.NewInt(4)); got != want {
		t.Fatalf("projects[1].contract = %s, want %s", got, want)
	}
}

func mustParseTimeForTest(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time: %v", err)
	}
	return parsed
}

func TestProjectListViewOmitsDetailOnlyFields(t *testing.T) {
	project := &Project{Meta: ProjectMeta{
		Contract:                common.BigToAddress(big.NewInt(1)),
		SourceCode:              "contract Source {}",
		SourceQualityReport:     "## Report",
		SourceQualityReportedAt: mustParseTimeForTest(t, "2026-05-22T00:00:00Z"),
	}}

	listView := projectToListView(project)
	if listView.Meta.SourceCode != "" {
		t.Fatalf("list sourceCode = %q, want empty", listView.Meta.SourceCode)
	}
	if listView.Meta.SourceQualityReport != "" {
		t.Fatalf("list sourceQualityReport = %q, want empty", listView.Meta.SourceQualityReport)
	}
	if listView.Meta.SourceQualityReportedAt != "" {
		t.Fatalf("list sourceQualityReportedAt = %q, want empty", listView.Meta.SourceQualityReportedAt)
	}
	if !listView.Meta.IsOpenSource {
		t.Fatal("list isOpenSource = false, want true")
	}

	detailView := projectToView(project, true)
	if detailView.Meta.SourceCode == "" || detailView.Meta.SourceQualityReport == "" || detailView.Meta.SourceQualityReportedAt == "" {
		t.Fatalf("detail view missing source detail fields: %#v", detailView.Meta)
	}
	if !detailView.Meta.IsOpenSource {
		t.Fatal("detail isOpenSource = false, want true")
	}
}
