```go
package main

import (
	"context"
	"math/big"
	"os"
	"os/signal"
	"syscall"
	"time"

	"your_project/internal/application"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
)

func main() {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	registry := application.NewMemoryProjectRegistry()
	stateStore := application.NewMemoryTokenMetadataStore()
	fetcher := application.NewFakeTokenMetadataFetcher()
	publisher := application.NewLogTokenMetadataEventPublisher()

	reconciler := application.NewTokenMetadataReconciler(
		registry,
		fetcher,
		stateStore,
		publisher,
	)

	engine := application.NewTokenMetadataSyncEngine(
		registry,
		reconciler,
		10*time.Second,
		4,
		1024,
	)

	projectID := uuid.New()

	project := application.Project{
		ProjectID:   projectID,
		BlockTime:   uint64(time.Now().Unix()),
		BlockNumber: 123456,
		TokenMetadata: &application.TokenMetadata{
			Name:          "Example Token",
			Symbol:        "EXT",
			Decimals:      18,
			TotalSupply:   big.NewInt(1_000_000),
			Address:       common.HexToAddress("0x1111111111111111111111111111111111111111"),
			SourceCode:    "",
			SourceCodeABI: "",
		},
	}

	registry.AddProject(project)

	engine.Start(ctx)

	time.Sleep(3 * time.Second)

	engine.TriggerManualSync(projectID)

	<-ctx.Done()
}
```