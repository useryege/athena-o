package wormtrading

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/wormtrading/apiclient"
	wormstore "github.com/useryege/athena/internal/wormtrading/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func callCatalogBudgetEntry(ctx context.Context, s *Service, entry string) error {
	items := []*apiclient.MarketCombinationItemInput{{EventConditionId: catalogConditionID(1), MarketConditionId: catalogConditionID(2), IsYes: true}}
	switch entry {
	case "catalog":
		_, err := s.GetOrderEventCatalog(ctx, &apiclient.GetOrderEventCatalogRequest{EventConditionId: catalogConditionID(1)})
		return err
	case "create":
		_, err := s.CreateMarketCombination(ctx, &apiclient.CreateMarketCombinationRequest{OwnerAccountId: catalogAccountID, Name: "Budget", Items: items})
		return err
	default:
		_, err := s.UpdateMarketCombination(ctx, &apiclient.UpdateMarketCombinationRequest{OwnerAccountId: catalogAccountID, Id: "cb9e4741-0b85-4891-a453-e13dfa4636c6", Name: "Budget", ExpectedRevision: 1, Items: items})
		return err
	}
}

func TestCatalogRequestBudgetBoundsAuthorization(t *testing.T) {
	for _, entry := range []string{"catalog", "create", "update"} {
		for _, mode := range []string{"service deadline", "earlier caller deadline", "caller cancellation"} {
			t.Run(entry+"/"+mode, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				budget := 30 * time.Millisecond
				if mode == "earlier caller deadline" {
					var deadlineCancel context.CancelFunc
					ctx, deadlineCancel = context.WithTimeout(ctx, 30*time.Millisecond)
					defer deadlineCancel()
					budget = time.Hour
				}
				entered := make(chan context.Context, 1)
				store := &combinationStoreSpy{}
				s := newCombinationService(store, accountaccess.AccessLevelReadWrite, nil)
				s.wormCatalogBudget = budget
				s.accountAccessReader = accountAccessFunc(func(ctx context.Context, _ string) (accountaccess.Access, error) {
					entered <- ctx
					<-ctx.Done()
					return accountaccess.Access{}, ctx.Err()
				})
				providerCalls := 0
				s.catalogReader = catalogReaderFunc(func(context.Context, *apiclient.GetOrderEventCatalogRequest) (*apiclient.GetOrderEventCatalogResponse, error) {
					providerCalls++
					return nil, nil
				})
				done := make(chan error, 1)
				go func() { done <- callCatalogBudgetEntry(combinationAccountContext(ctx), s, entry) }()
				var accessCtx context.Context
				select {
				case accessCtx = <-entered:
				case <-time.After(time.Second):
					t.Fatal("account reader was not reached")
				}
				deadline, bounded := accessCtx.Deadline()
				if !bounded {
					cancel()
					<-done
					t.Fatal("authorization has no finite request deadline")
				}
				if callerDeadline, ok := ctx.Deadline(); ok {
					require.Equal(t, callerDeadline, deadline)
				}
				want := codes.DeadlineExceeded
				if mode == "caller cancellation" {
					cancel()
					want = codes.Canceled
				}
				select {
				case err := <-done:
					require.Equal(t, want, status.Code(err))
				case <-time.After(time.Second):
					t.Fatal("request did not stop within its budget")
				}
				require.Zero(t, providerCalls)
				require.Empty(t, store.saved)
				require.Empty(t, store.updated)
			})
		}
	}
}

type budgetCombinationStore struct {
	wormstore.Store
	deadline chan time.Time
}

func (s *budgetCombinationStore) wait(ctx context.Context) (*wormstore.MarketCombination, error) {
	deadline, _ := ctx.Deadline()
	s.deadline <- deadline
	<-ctx.Done()
	return nil, ctx.Err()
}
func (s *budgetCombinationStore) CreateMarketCombination(ctx context.Context, _, _ string, _ []wormstore.MarketCombinationItemInput) (*wormstore.MarketCombination, error) {
	return s.wait(ctx)
}
func (s *budgetCombinationStore) UpdateMarketCombination(ctx context.Context, _, _, _ string, _ int64, _ []wormstore.MarketCombinationItemInput) (*wormstore.MarketCombination, error) {
	return s.wait(ctx)
}

func TestCatalogRequestBudgetSharedByAuthorizationCatalogAndStore(t *testing.T) {
	for _, entry := range []string{"catalog", "create", "update"} {
		t.Run(entry, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			accessEntered := make(chan context.Context, 1)
			releaseAccess := make(chan struct{})
			providerDeadline := make(chan time.Time, 1)
			store := &budgetCombinationStore{deadline: make(chan time.Time, 1)}
			s := newCombinationService(store, accountaccess.AccessLevelReadWrite, nil)
			s.wormCatalogBudget = time.Second
			s.accountAccessReader = accountAccessFunc(func(ctx context.Context, _ string) (accountaccess.Access, error) {
				accessEntered <- ctx
				select {
				case <-releaseAccess:
					return catalogAccess(accountaccess.AccessLevelReadWrite), nil
				case <-ctx.Done():
					return accountaccess.Access{}, ctx.Err()
				}
			})
			s.catalogReader = catalogReaderFunc(func(ctx context.Context, _ *apiclient.GetOrderEventCatalogRequest) (*apiclient.GetOrderEventCatalogResponse, error) {
				deadline, _ := ctx.Deadline()
				providerDeadline <- deadline
				return combinationCatalog(catalogConditionID(1), "Event", catalogConditionID(2), "Market", true, "Yes"), nil
			})
			done := make(chan error, 1)
			go func() { done <- callCatalogBudgetEntry(combinationAccountContext(ctx), s, entry) }()
			var accessCtx context.Context
			select {
			case accessCtx = <-accessEntered:
			case <-time.After(2 * time.Second):
				t.Fatal("account reader was not reached")
			}
			deadline, bounded := accessCtx.Deadline()
			close(releaseAccess)
			require.True(t, bounded, "authorization must start the overall budget")
			select {
			case got := <-providerDeadline:
				require.Equal(t, deadline, got, "catalog must retain time already spent authorizing")
			case <-time.After(2 * time.Second):
				t.Fatal("catalog was not reached")
			}
			if entry != "catalog" {
				select {
				case got := <-store.deadline:
					require.Equal(t, deadline, got, "store must retain the same remaining budget")
				case <-time.After(2 * time.Second):
					t.Fatal("store was not reached")
				}
			}
			select {
			case err := <-done:
				if entry == "catalog" {
					require.NoError(t, err)
				} else {
					require.Equal(t, codes.DeadlineExceeded, status.Code(err))
				}
			case <-time.After(2 * time.Second):
				t.Fatal("store did not stop at the overall deadline")
			}
		})
	}
}
