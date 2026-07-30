package tokenchain

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/useryege/athena/internal/token/chainregistry"
	"github.com/useryege/athena/util/env"
)

const (
	ethEnvironmentPrefix = "ATHENA_TOKEN_ETH"
	bscEnvironmentPrefix = "ATHENA_TOKEN_BSC"
)

type chainFlags struct {
	enabled                          string
	nodeWSURLs                       string
	athenaContract                   string
	processorInitialLookbackDuration string
	processorPollInterval            string
	swapPollInterval                 string
}

type Flags struct {
	ethereum chainFlags
	bsc      chainFlags
}

func (f *Flags) Bind(command *cobra.Command) {
	bindChain(command, "eth", ethEnvironmentPrefix, "Ethereum Mainnet", &f.ethereum)
	bindChain(command, "bsc", bscEnvironmentPrefix, "BSC Mainnet", &f.bsc)
}

func (f *Flags) Registry() (*chainregistry.Registry, error) {
	ethereum, err := parseChain("eth", ethEnvironmentPrefix, f.ethereum)
	if err != nil {
		return nil, err
	}
	bsc, err := parseChain("bsc", bscEnvironmentPrefix, f.bsc)
	if err != nil {
		return nil, err
	}
	return chainregistry.New(chainregistry.Config{Ethereum: ethereum, BSC: bsc})
}

func bindChain(command *cobra.Command, flagPrefix, environmentPrefix, name string, values *chainFlags) {
	command.Flags().StringVar(&values.enabled, flagPrefix+"-enabled", env.StringFromEnv(environmentPrefix+"_ENABLED", ""), "Enable "+name)
	command.Flags().StringVar(&values.nodeWSURLs, flagPrefix+"-node-ws-urls", env.StringFromEnv(environmentPrefix+"_NODE_WS_URLS", ""), "Comma, space, or newline-separated "+name+" WebSocket node URLs")
	command.Flags().StringVar(&values.athenaContract, flagPrefix+"-athena-contract", env.StringFromEnv(environmentPrefix+"_ATHENA_CONTRACT", ""), name+" ATHENA contract address")
	command.Flags().StringVar(&values.processorInitialLookbackDuration, flagPrefix+"-processor-initial-lookback-duration", env.StringFromEnv(environmentPrefix+"_PROCESSOR_INITIAL_LOOKBACK_DURATION", ""), name+" processor initial lookback duration")
	command.Flags().StringVar(&values.processorPollInterval, flagPrefix+"-processor-poll-interval", env.StringFromEnv(environmentPrefix+"_PROCESSOR_POLL_INTERVAL", ""), name+" processor poll interval")
	command.Flags().StringVar(&values.swapPollInterval, flagPrefix+"-swap-poll-interval", env.StringFromEnv(environmentPrefix+"_SWAP_POLL_INTERVAL", ""), name+" Swap processor poll interval")
}

func parseChain(flagPrefix, environmentPrefix string, values chainFlags) (chainregistry.ChainConfig, error) {
	enabled, err := parseBool(values.enabled, environmentPrefix+"_ENABLED", "--"+flagPrefix+"-enabled")
	if err != nil {
		return chainregistry.ChainConfig{}, err
	}
	nodeWSURLs := parseList(values.nodeWSURLs)
	if len(nodeWSURLs) == 0 {
		return chainregistry.ChainConfig{}, requiredError(environmentPrefix+"_NODE_WS_URLS", "--"+flagPrefix+"-node-ws-urls")
	}
	athenaContract := strings.TrimSpace(values.athenaContract)
	if athenaContract == "" {
		return chainregistry.ChainConfig{}, requiredError(environmentPrefix+"_ATHENA_CONTRACT", "--"+flagPrefix+"-athena-contract")
	}
	initialLookback, err := parseDuration(values.processorInitialLookbackDuration, environmentPrefix+"_PROCESSOR_INITIAL_LOOKBACK_DURATION", "--"+flagPrefix+"-processor-initial-lookback-duration")
	if err != nil {
		return chainregistry.ChainConfig{}, err
	}
	pollInterval, err := parseDuration(values.processorPollInterval, environmentPrefix+"_PROCESSOR_POLL_INTERVAL", "--"+flagPrefix+"-processor-poll-interval")
	if err != nil {
		return chainregistry.ChainConfig{}, err
	}
	swapPollInterval, err := parseDuration(values.swapPollInterval, environmentPrefix+"_SWAP_POLL_INTERVAL", "--"+flagPrefix+"-swap-poll-interval")
	if err != nil {
		return chainregistry.ChainConfig{}, err
	}
	return chainregistry.ChainConfig{
		Enabled:                          enabled,
		NodeWSURLs:                       nodeWSURLs,
		AthenaContract:                   athenaContract,
		ProcessorInitialLookbackDuration: initialLookback,
		ProcessorPollInterval:            pollInterval,
		SwapPollInterval:                 swapPollInterval,
	}, nil
}

func parseBool(raw, environmentVariable, flag string) (bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false, requiredError(environmentVariable, flag)
	}
	if !strings.EqualFold(raw, "true") && !strings.EqualFold(raw, "false") {
		return false, fmt.Errorf("%s / %s must be true or false", environmentVariable, flag)
	}
	value, _ := strconv.ParseBool(raw)
	return value, nil
}

func parseDuration(raw, environmentVariable, flag string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, requiredError(environmentVariable, flag)
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("parse %s / %s: %w", environmentVariable, flag, err)
	}
	return value, nil
}

func parseList(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	})
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(strings.Trim(part, `"'`))
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}

func requiredError(environmentVariable, flag string) error {
	return fmt.Errorf("%s / %s is required", environmentVariable, flag)
}
