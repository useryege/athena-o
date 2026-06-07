package runtime

import (
	"context"
	stderrors "errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/application"
	"github.com/useryege/athena/util/ave"
	"github.com/useryege/athena/util/errors"
	"github.com/useryege/athena/util/ethereumapi"
	"github.com/useryege/athena/util/ethws"
	"google.golang.org/grpc"
)

func runAPIMode(ctx context.Context, opts Options) error {
	nodeClient, err := ethws.DialContext(ctx, opts.NodeWSURL, opts.NodeWSUseProxy)
	if err != nil {
		log.Fatalf("failed to connect to node websocket: %v", err)
	}
	defer nodeClient.Close()

	nodeChainID, err := nodeClient.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch node chain id: %w", err)
	}
	log.Infof("node chain id: %d", nodeChainID.Int64())

	var apiFetcher ethereumapi.EthereumAPI
	if opts.EtherscanAPIBaseURL != "" && opts.EtherscanAPIKey != "" {
		apiFetcher = ethereumapi.NewEthereumAPI(opts.EtherscanAPIBaseURL, opts.EtherscanAPIKey)
	}

	athenaContractAddress, v2FactoryContractAddress, wethContractAddress, usdtContractAddress, wethDecimals, usdtDecimals, err := loadAthenaContractOptions(ctx, nodeClient, opts.AthenaContract)
	if err != nil {
		return err
	}

	liquidityLockerAddresses, err := parseLiquidityLockerAddresses(opts.LiquidityLockers)
	if err != nil {
		return err
	}

	server, err := application.NewServer(application.ApplicationServerOpts{
		NodeClient:     nodeClient,
		AthenaContract: athenaContractAddress,
		ChainID:        nodeChainID.Int64(),
		AveConfig: ave.Config{
			BaseURL: opts.AveAPIBaseURL,
			APIKey:  opts.AveAPIKey,
		},
		Store:           opts.Store,
		LiquidityLocker: liquidityLockerAddresses,
		APIFetcher:      apiFetcher,

		// Fetch from Athena contract
		V2FactoryContract: v2FactoryContractAddress,
		WethContract:      wethContractAddress,
		UsdtContract:      usdtContractAddress,
		WethDecimals:      wethDecimals,
		UsdtDecimals:      usdtDecimals,
	})
	if err != nil {
		return err
	}

	applicationGrpc := server.CreateGRPC()

	lc := &net.ListenConfig{}
	listener, err := lc.Listen(ctx, "tcp", fmt.Sprintf("%s:%d", opts.ListenHost, opts.ListenPort))
	errors.CheckError(err)

	if err := server.Start(); err != nil {
		return err
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		s := <-sigCh
		log.Printf("got signal %v, attempting graceful shutdown", s)
		applicationGrpc.GracefulStop()
		if err := server.Stop(); err != nil {
			log.Printf("failed to stop application server cleanly: %v", err)
		}
		wg.Done()
	}()

	log.Println("starting grpc server")
	err = applicationGrpc.Serve(listener)
	if err != nil && !stderrors.Is(err, grpc.ErrServerStopped) {
		errors.CheckError(err)
	}

	wg.Wait()
	log.Println("clean shutdown")
	return nil
}
