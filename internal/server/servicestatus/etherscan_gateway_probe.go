package servicestatus

import (
	"context"
	"strings"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/useryege/athena/internal/etherscangatewayprobe"
	operationlogrecord "github.com/useryege/athena/internal/operationlog/record"
	servicestatuspkg "github.com/useryege/athena/pkg/apiclient/servicestatus"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

const (
	defaultEtherscanGatewayProbeQueryAddress = "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"
	etherscanGatewayProbeMinIntervalMS       = int64(1)
	etherscanGatewayProbeMaxIntervalMS       = int64(1000)
	etherscanGatewayProbeMinRequestsPerKey   = int32(1)
	etherscanGatewayProbeMaxRequestsPerKey   = int32(20)
	etherscanGatewayProbeMaxStartSpread      = 10 * time.Minute
	etherscanGatewayProbeCallBudget          = 90 * time.Second
)

const (
	etherscanGatewayProbeRunStatusIdle      = "idle"
	etherscanGatewayProbeRunStatusRunning   = "running"
	etherscanGatewayProbeRunStatusCompleted = "completed"
	etherscanGatewayProbeRunStatusError     = "error"
	etherscanGatewayProbeRunResultPass      = "pass"
	etherscanGatewayProbeRunResultFail      = "fail"
	etherscanGatewayProbeRunResultError     = "error"
)

func (s *Server) RunEtherscanGatewayProbe(
	ctx context.Context,
	req *servicestatuspkg.RunEtherscanGatewayProbeRequest,
) (*servicestatuspkg.EtherscanGatewayProbeRun, error) {
	cfg, err := s.newEtherscanGatewayProbeConfig(req)
	if err != nil {
		return nil, err
	}

	runUUID, err := uuid.NewRandom()
	if err != nil {
		return nil, grpcstatus.Errorf(codes.Internal, "create probe run id: %v", err)
	}

	now := time.Now().Unix()
	total := len(cfg.APIKeys) * cfg.RequestsPerKey
	run := &servicestatuspkg.EtherscanGatewayProbeRun{
		RunId:          runUUID.String(),
		Status:         etherscanGatewayProbeRunStatusRunning,
		CreatedAt:      now,
		StartedAt:      now,
		IntervalMs:     cfg.Interval.Milliseconds(),
		RequestsPerKey: int32(cfg.RequestsPerKey),
		KeyCount:       int32(len(cfg.APIKeys)),
		GatewayCount:   int32(len(cfg.GatewayAddrs)),
		Total:          int32(total),
		RequiredSuccess: int32(
			etherscangatewayprobe.RequiredSuccess(total),
		),
		Counts: &servicestatuspkg.EtherscanGatewayProbeCategoryCounts{},
	}

	s.probeMu.Lock()
	if s.latestProbeRun != nil && s.latestProbeRun.GetStatus() == etherscanGatewayProbeRunStatusRunning {
		activeID := s.latestProbeRun.GetRunId()
		s.probeMu.Unlock()
		return nil, grpcstatus.Errorf(codes.FailedPrecondition, "Etherscan Gateway probe run %s is already running", activeID)
	}
	s.latestProbeRun = cloneEtherscanGatewayProbeRun(run)
	s.probeMu.Unlock()

	go s.executeEtherscanGatewayProbeRun(run.RunId, cfg)
	operationlogrecord.CaptureResource(ctx, "probe_run", run.GetRunId())
	operationlogrecord.CaptureString(ctx, "runId", run.GetRunId())
	operationlogrecord.CaptureInt64(ctx, "requestsPerKey", int64(run.GetRequestsPerKey()))
	operationlogrecord.CaptureInt64(ctx, "gatewayCount", int64(run.GetGatewayCount()))
	operationlogrecord.CaptureString(ctx, "state", run.GetStatus())
	operationlogrecord.Accept(ctx, "SYSTEM_GATEWAY_PROBE_CREATE")
	return cloneEtherscanGatewayProbeRun(run), nil
}

func (s *Server) GetLatestEtherscanGatewayProbeRun(
	context.Context,
	*servicestatuspkg.GetLatestEtherscanGatewayProbeRunRequest,
) (*servicestatuspkg.EtherscanGatewayProbeRun, error) {
	s.probeMu.Lock()
	defer s.probeMu.Unlock()
	if s.latestProbeRun == nil {
		return &servicestatuspkg.EtherscanGatewayProbeRun{
			Status: etherscanGatewayProbeRunStatusIdle,
			Counts: &servicestatuspkg.EtherscanGatewayProbeCategoryCounts{},
		}, nil
	}
	return cloneEtherscanGatewayProbeRun(s.latestProbeRun), nil
}

func (s *Server) GetEtherscanGatewayProbeRun(
	_ context.Context,
	req *servicestatuspkg.GetEtherscanGatewayProbeRunRequest,
) (*servicestatuspkg.EtherscanGatewayProbeRun, error) {
	runID := strings.TrimSpace(req.GetRunId())
	if runID == "" {
		return nil, grpcstatus.Error(codes.InvalidArgument, "run_id is required")
	}

	s.probeMu.Lock()
	defer s.probeMu.Unlock()
	if s.latestProbeRun == nil || s.latestProbeRun.GetRunId() != runID {
		return nil, grpcstatus.Errorf(codes.NotFound, "Etherscan Gateway probe run %s was not found", runID)
	}
	return cloneEtherscanGatewayProbeRun(s.latestProbeRun), nil
}

func (s *Server) newEtherscanGatewayProbeConfig(req *servicestatuspkg.RunEtherscanGatewayProbeRequest) (etherscangatewayprobe.Config, error) {
	if req == nil {
		req = &servicestatuspkg.RunEtherscanGatewayProbeRequest{}
	}

	intervalMS := req.GetIntervalMs()
	if intervalMS < etherscanGatewayProbeMinIntervalMS || intervalMS > etherscanGatewayProbeMaxIntervalMS {
		return etherscangatewayprobe.Config{}, grpcstatus.Errorf(
			codes.InvalidArgument,
			"interval_ms must be between %d and %d",
			etherscanGatewayProbeMinIntervalMS,
			etherscanGatewayProbeMaxIntervalMS,
		)
	}

	requestsPerKey := req.GetRequestsPerKey()
	if requestsPerKey < etherscanGatewayProbeMinRequestsPerKey || requestsPerKey > etherscanGatewayProbeMaxRequestsPerKey {
		return etherscangatewayprobe.Config{}, grpcstatus.Errorf(
			codes.InvalidArgument,
			"requests_per_key must be between %d and %d",
			etherscanGatewayProbeMinRequestsPerKey,
			etherscanGatewayProbeMaxRequestsPerKey,
		)
	}

	keys := etherscangatewayprobe.ParseAPIKeys(s.etherscanAPIKeysRaw)
	if len(keys) == 0 {
		return etherscangatewayprobe.Config{}, grpcstatus.Error(codes.FailedPrecondition, "ATHENA_ETHERSCAN_MANAGER_API_KEYS is required")
	}

	gatewayAddrs, err := etherscangatewayprobe.ParseGatewayAddrs("", strings.Join(s.etherscanGatewayIPs, " "))
	if err != nil {
		return etherscangatewayprobe.Config{}, grpcstatus.Errorf(codes.FailedPrecondition, "parse ETHERSCAN_GATEWAY_IPS: %v", err)
	}
	if len(gatewayAddrs) == 0 {
		return etherscangatewayprobe.Config{}, grpcstatus.Error(codes.FailedPrecondition, "ETHERSCAN_GATEWAY_IPS is required")
	}

	authToken := strings.TrimSpace(s.etherscanGatewayAuthTokenRaw)
	if authToken == "" {
		return etherscangatewayprobe.Config{}, grpcstatus.Error(codes.FailedPrecondition, "ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN is required")
	}

	queryAddress := strings.TrimSpace(s.etherscanGatewayProbeAddress)
	if queryAddress == "" {
		queryAddress = defaultEtherscanGatewayProbeQueryAddress
	}
	if !ethcommon.IsHexAddress(queryAddress) {
		return etherscangatewayprobe.Config{}, grpcstatus.Errorf(codes.FailedPrecondition, "ATHENA_ETHERSCAN_GATEWAY_PROBE_QUERY_ADDRESS must be a valid EVM address, got %q", queryAddress)
	}

	interval := time.Duration(intervalMS) * time.Millisecond
	total := len(keys) * int(requestsPerKey)
	if total > 0 {
		startSpread := time.Duration(total-1) * interval
		if startSpread > etherscanGatewayProbeMaxStartSpread {
			return etherscangatewayprobe.Config{}, grpcstatus.Errorf(codes.InvalidArgument, "probe start spread %s exceeds %s", startSpread.Round(time.Millisecond), etherscanGatewayProbeMaxStartSpread)
		}
	}

	return etherscangatewayprobe.Config{
		APIKeys:        keys,
		GatewayAddrs:   gatewayAddrs,
		AuthToken:      authToken,
		QueryAddress:   ethcommon.HexToAddress(queryAddress).Hex(),
		RequestsPerKey: int(requestsPerKey),
		Interval:       interval,
	}, nil
}

func (s *Server) executeEtherscanGatewayProbeRun(runID string, cfg etherscangatewayprobe.Config) {
	runTimeout := etherscanGatewayProbeCallBudget
	if cfg.Interval > 0 && cfg.RequestsPerKey > 0 && len(cfg.APIKeys) > 0 {
		runTimeout += time.Duration(len(cfg.APIKeys)*cfg.RequestsPerKey-1) * cfg.Interval
	}
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()

	result, err := etherscangatewayprobe.RunGatewayMultiKeyStaggered(ctx, cfg)
	finishedAt := time.Now().Unix()

	s.probeMu.Lock()
	defer s.probeMu.Unlock()
	if s.latestProbeRun == nil || s.latestProbeRun.GetRunId() != runID {
		return
	}

	run := cloneEtherscanGatewayProbeRun(s.latestProbeRun)
	run.FinishedAt = finishedAt
	if err != nil {
		run.Status = etherscanGatewayProbeRunStatusError
		run.Result = etherscanGatewayProbeRunResultError
		run.ErrorMessage = err.Error()
		s.latestProbeRun = run
		return
	}

	run.Status = etherscanGatewayProbeRunStatusCompleted
	run.Result = etherscanGatewayProbeRunResultFail
	if result.Counts.Success >= result.RequiredSuccess() {
		run.Result = etherscanGatewayProbeRunResultPass
	}
	run.KeyCount = int32(result.KeyCount)
	run.GatewayCount = int32(result.GatewayCount)
	run.RequestsPerKey = int32(result.RequestsPerKey)
	run.Total = int32(result.Total)
	run.RequiredSuccess = int32(result.RequiredSuccess())
	run.Counts = probeCountsToProto(result.Counts)
	run.ElapsedMs = result.Elapsed.Milliseconds()
	run.StartSpreadMs = result.StartSpread.Milliseconds()
	run.KeySummaries = probeEntitySummariesToKeyProto(result.KeySummaries)
	run.GatewaySummaries = probeEntitySummariesToGatewayProto(result.GatewaySummaries)
	run.Samples = append([]string(nil), result.Samples...)
	s.latestProbeRun = run
}

func probeCountsToProto(counts etherscangatewayprobe.Counts) *servicestatuspkg.EtherscanGatewayProbeCategoryCounts {
	return &servicestatuspkg.EtherscanGatewayProbeCategoryCounts{
		Success:        int32(counts.Success),
		RateLimit:      int32(counts.RateLimit),
		Authentication: int32(counts.Authentication),
		Plan:           int32(counts.Plan),
		InvalidRequest: int32(counts.InvalidRequest),
		Malformed:      int32(counts.Malformed),
		Upstream:       int32(counts.Upstream),
		Other:          int32(counts.Other),
	}
}

func probeEntitySummariesToKeyProto(items []etherscangatewayprobe.EntitySummary) []*servicestatuspkg.EtherscanGatewayProbeKeySummary {
	summaries := make([]*servicestatuspkg.EtherscanGatewayProbeKeySummary, 0, len(items))
	for _, item := range items {
		summaries = append(summaries, &servicestatuspkg.EtherscanGatewayProbeKeySummary{
			KeyLabel: item.Label,
			Counts:   probeCountsToProto(item.Counts),
		})
	}
	return summaries
}

func probeEntitySummariesToGatewayProto(items []etherscangatewayprobe.EntitySummary) []*servicestatuspkg.EtherscanGatewayProbeGatewaySummary {
	summaries := make([]*servicestatuspkg.EtherscanGatewayProbeGatewaySummary, 0, len(items))
	for _, item := range items {
		summaries = append(summaries, &servicestatuspkg.EtherscanGatewayProbeGatewaySummary{
			Gateway: item.Label,
			Counts:  probeCountsToProto(item.Counts),
		})
	}
	return summaries
}

func cloneEtherscanGatewayProbeRun(run *servicestatuspkg.EtherscanGatewayProbeRun) *servicestatuspkg.EtherscanGatewayProbeRun {
	if run == nil {
		return nil
	}
	clone := *run
	clone.Counts = cloneEtherscanGatewayProbeCounts(run.Counts)
	clone.KeySummaries = make([]*servicestatuspkg.EtherscanGatewayProbeKeySummary, 0, len(run.KeySummaries))
	for _, item := range run.KeySummaries {
		if item == nil {
			continue
		}
		itemClone := *item
		itemClone.Counts = cloneEtherscanGatewayProbeCounts(item.Counts)
		clone.KeySummaries = append(clone.KeySummaries, &itemClone)
	}
	clone.GatewaySummaries = make([]*servicestatuspkg.EtherscanGatewayProbeGatewaySummary, 0, len(run.GatewaySummaries))
	for _, item := range run.GatewaySummaries {
		if item == nil {
			continue
		}
		itemClone := *item
		itemClone.Counts = cloneEtherscanGatewayProbeCounts(item.Counts)
		clone.GatewaySummaries = append(clone.GatewaySummaries, &itemClone)
	}
	clone.Samples = append([]string(nil), run.Samples...)
	return &clone
}

func cloneEtherscanGatewayProbeCounts(counts *servicestatuspkg.EtherscanGatewayProbeCategoryCounts) *servicestatuspkg.EtherscanGatewayProbeCategoryCounts {
	if counts == nil {
		return &servicestatuspkg.EtherscanGatewayProbeCategoryCounts{}
	}
	clone := *counts
	return &clone
}
