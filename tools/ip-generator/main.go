package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/useryege/athena/util/ethws"
)

const (
	maxOctet            = 255
	ipv4HostSpace       = 1 << 16
	webSocketPort       = 8546
	defaultWorkerCount  = 256
	defaultProbeTimeout = 3 * time.Second
)

type probeResult struct {
	endpoint    string
	chainID     string
	blockNumber uint64
	err         error
}

type scanSummary struct {
	scanned    int
	successful int
	failed     int
}

func main() {
	first := flag.Int("first", 0, "first IPv4 octet (0-255)")
	second := flag.Int("second", 0, "second IPv4 octet (0-255)")
	workers := flag.Int("workers", defaultWorkerCount, "number of concurrent endpoint probes")
	timeout := flag.Duration("timeout", defaultProbeTimeout, "timeout for each endpoint probe")
	flag.Parse()

	if flag.NArg() != 0 {
		exitWithError("unexpected positional arguments: %v", flag.Args())
	}

	providedFlags := make(map[string]bool)
	flag.Visit(func(value *flag.Flag) {
		providedFlags[value.Name] = true
	})
	if !providedFlags["first"] {
		exitWithError("-first is required")
	}
	if !providedFlags["second"] {
		exitWithError("-second is required")
	}
	if *first < 0 || *first > maxOctet {
		exitWithError("-first must be between 0 and 255")
	}
	if *second < 0 || *second > maxOctet {
		exitWithError("-second must be between 0 and 255")
	}
	if *workers <= 0 {
		exitWithError("-workers must be greater than 0")
	}
	if *timeout <= 0 {
		exitWithError("-timeout must be greater than 0")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	output := bufio.NewWriter(os.Stdout)
	summary, scanErr := scanIPRange(ctx, output, *first, *second, *workers, *timeout)
	flushErr := output.Flush()
	fmt.Fprintf(
		os.Stderr,
		"scanned=%d successful=%d failed=%d\n",
		summary.scanned,
		summary.successful,
		summary.failed,
	)

	if scanErr != nil {
		exitWithError("scanning IP addresses: %v", scanErr)
	}
	if flushErr != nil {
		exitWithError("flushing IP addresses: %v", flushErr)
	}
}

func scanIPRange(
	ctx context.Context,
	output io.Writer,
	first int,
	second int,
	workers int,
	timeout time.Duration,
) (scanSummary, error) {
	scanCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobs := make(chan string, workers)
	results := make(chan probeResult, workers)

	var workerGroup sync.WaitGroup
	workerGroup.Add(workers)
	for worker := 0; worker < workers; worker++ {
		go func() {
			defer workerGroup.Done()
			probeEndpoints(scanCtx, jobs, results, timeout)
		}()
	}

	go generateEndpoints(scanCtx, first, second, jobs)
	go func() {
		workerGroup.Wait()
		close(results)
	}()

	var summary scanSummary
	var outputErr error
	for result := range results {
		summary.scanned++
		if result.err != nil {
			summary.failed++
			continue
		}

		summary.successful++
		if outputErr != nil {
			continue
		}
		if _, err := fmt.Fprintf(
			output,
			"%s %s %d\n",
			result.endpoint,
			result.chainID,
			result.blockNumber,
		); err != nil {
			outputErr = fmt.Errorf("writing probe result: %w", err)
			cancel()
		}
	}

	if outputErr != nil {
		return summary, outputErr
	}
	if err := ctx.Err(); err != nil {
		return summary, err
	}
	return summary, nil
}

func generateEndpoints(ctx context.Context, first, second int, jobs chan<- string) {
	defer close(jobs)

	for host := 1; host < ipv4HostSpace-1; host++ {
		third := host >> 8
		fourth := host & maxOctet
		endpoint := fmt.Sprintf("ws://%d.%d.%d.%d:%d", first, second, third, fourth, webSocketPort)

		select {
		case jobs <- endpoint:
		case <-ctx.Done():
			return
		}
	}
}

func probeEndpoints(
	ctx context.Context,
	jobs <-chan string,
	results chan<- probeResult,
	timeout time.Duration,
) {
	for endpoint := range jobs {
		results <- probeEndpoint(ctx, endpoint, timeout)
	}
}

func probeEndpoint(ctx context.Context, endpoint string, timeout time.Duration) probeResult {
	result := probeResult{endpoint: endpoint}
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client, err := ethws.DialContext(probeCtx, endpoint, "")
	if err != nil {
		result.err = fmt.Errorf("dial websocket: %w", err)
		return result
	}
	defer client.Close()

	chainID, err := client.ChainID(probeCtx)
	if err != nil {
		result.err = fmt.Errorf("get chain ID: %w", err)
		return result
	}
	if chainID == nil {
		result.err = fmt.Errorf("get chain ID: empty response")
		return result
	}

	blockNumber, err := client.BlockNumber(probeCtx)
	if err != nil {
		result.err = fmt.Errorf("get block number: %w", err)
		return result
	}

	result.chainID = chainID.String()
	result.blockNumber = blockNumber
	return result
}

func exitWithError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
