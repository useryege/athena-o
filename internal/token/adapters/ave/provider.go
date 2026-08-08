package aveadapter

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/token/research"
	researchapp "github.com/useryege/athena/internal/token/research/application"
	"github.com/useryege/athena/internal/token/shared"
	"github.com/useryege/athena/util/ave"
)

const httpClientTimeout = 60 * time.Second

type Config struct {
	APIKey  string
	BaseURL string
}

type Provider struct {
	client ave.Client
}

func New(config Config) (*Provider, error) {
	if strings.TrimSpace(config.APIKey) == "" {
		return nil, fmt.Errorf("Ave API key is required")
	}
	client, err := ave.NewClient(ave.Config{BaseURL: config.BaseURL, APIKey: config.APIKey, Timeout: httpClientTimeout})
	if err != nil {
		return nil, err
	}
	return &Provider{client: client}, nil
}

func (p *Provider) GetMarketData(ctx context.Context, request researchapp.AveMarketDataRequest) (research.AveObservationV1, error) {
	commonContract := ethcommon.Address(request.Contract)
	response, err := p.client.GetTokenDetail(ctx, commonContract, request.ChainID)
	if err != nil {
		return research.AveObservationV1{}, fmt.Errorf("fetch Ave token detail chain_id=%d contract=%s: %w", request.ChainID, request.Contract.Hex(), err)
	}
	return normalizeObservation(response, commonContract, request.ChainID, ethcommon.Address(request.WethPair), ethcommon.Address(request.UsdtPair))
}

func normalizeObservation(
	resp *ave.TokenDetailResponse,
	expectedContract ethcommon.Address,
	chainID int64,
	expectedWethPair ethcommon.Address,
	expectedUsdtPair ethcommon.Address,
) (research.AveObservationV1, error) {
	if resp == nil {
		return research.AveObservationV1{}, fmt.Errorf("response is nil")
	}
	expectedChain, ok := ave.ChainNameForChainID(chainID)
	if !ok {
		return research.AveObservationV1{}, fmt.Errorf("unsupported chain id %d", chainID)
	}
	token := resp.Data.Token
	address, err := requiredAddress("token address", token.Token)
	if err != nil {
		return research.AveObservationV1{}, err
	}
	if address != expectedContract {
		return research.AveObservationV1{}, fmt.Errorf("token address %s does not match requested contract %s", address.Hex(), expectedContract.Hex())
	}
	if !strings.EqualFold(strings.TrimSpace(token.Chain), expectedChain) {
		return research.AveObservationV1{}, fmt.Errorf("token chain %q does not match chain id %d", token.Chain, chainID)
	}
	totalSupply, err := aveDecimal("total supply", token.Total)
	if err != nil {
		return research.AveObservationV1{}, err
	}
	currentPriceUSD, err := aveDecimal("current price usd", token.CurrentPriceUSD)
	if err != nil {
		return research.AveObservationV1{}, err
	}
	currentPriceETH, err := aveDecimal("current price eth", token.CurrentPriceETH)
	if err != nil {
		return research.AveObservationV1{}, err
	}
	marketCap, err := aveDecimal("market cap", token.MarketCap)
	if err != nil {
		return research.AveObservationV1{}, err
	}
	fdv, err := aveDecimal("fdv", token.FDV)
	if err != nil {
		return research.AveObservationV1{}, err
	}
	tvl, err := aveDecimal("tvl", token.TVL)
	if err != nil {
		return research.AveObservationV1{}, err
	}
	mainPairTVL, err := aveDecimal("main pair tvl", token.MainPairTVL)
	if err != nil {
		return research.AveObservationV1{}, err
	}
	riskScore, err := aveDecimal("risk score", token.RiskScore)
	if err != nil {
		return research.AveObservationV1{}, err
	}
	isMintable, err := optionalBool("is mintable", token.IsMintable)
	if err != nil {
		return research.AveObservationV1{}, err
	}
	result := research.AveObservationV1{
		ChainID: chainID, IsAudited: resp.Data.IsAudited,
		Token: research.AveTokenV1{
			Address: shared.Address(address), Name: token.Name, Symbol: token.Symbol, LogoURL: strings.TrimSpace(token.LogoURL), Decimals: token.Decimal,
			TotalSupply: totalSupply, CurrentPriceUSD: currentPriceUSD, CurrentPriceETH: currentPriceETH,
			MarketCap: marketCap, FDV: fdv, TVL: tvl, MainPairTVL: mainPairTVL,
			Holders: token.Holders, RiskLevel: token.RiskLevel, RiskScore: riskScore, RiskInfo: token.RiskInfo,
			IsMintable: isMintable, HasMintMethod: token.HasMintMethod, IsLPNotLocked: token.IsLPNotLocked,
			HasNotRenounced: token.HasNotRenounced, HasNotAudited: token.HasNotAudited, HasNotOpenSource: token.HasNotOpenSource,
			IsInBlacklist: token.IsInBlacklist, IsHoneypot: token.IsHoneypot, LaunchAt: unixTime(token.LaunchAt), UpdatedAt: unixTime(token.UpdatedAt),
		},
		Pairs: make([]research.AvePairV1, 0, 2),
	}
	var wethPair, usdtPair *research.AvePairV1
	for index, pair := range resp.Data.Pairs {
		pairValue := strings.TrimSpace(pair.Pair)
		if !ethcommon.IsHexAddress(pairValue) {
			continue
		}
		pairAddress := ethcommon.HexToAddress(pairValue)
		if pairAddress == (ethcommon.Address{}) {
			continue
		}
		switch {
		case expectedWethPair != (ethcommon.Address{}) && pairAddress == expectedWethPair && wethPair == nil:
			value, err := normalizePair(pair, index, pairAddress, expectedChain, chainID)
			if err != nil {
				return research.AveObservationV1{}, err
			}
			wethPair = &value
		case expectedUsdtPair != (ethcommon.Address{}) && pairAddress == expectedUsdtPair && usdtPair == nil:
			value, err := normalizePair(pair, index, pairAddress, expectedChain, chainID)
			if err != nil {
				return research.AveObservationV1{}, err
			}
			usdtPair = &value
		}
	}
	if wethPair != nil {
		result.Pairs = append(result.Pairs, *wethPair)
	}
	if usdtPair != nil {
		result.Pairs = append(result.Pairs, *usdtPair)
	}
	return result, nil
}

func normalizePair(pair ave.Pair, index int, pairAddress ethcommon.Address, expectedChain string, chainID int64) (research.AvePairV1, error) {
	if !strings.EqualFold(strings.TrimSpace(pair.Chain), expectedChain) {
		return research.AvePairV1{}, fmt.Errorf("pair %d chain %q does not match chain id %d", index, pair.Chain, chainID)
	}
	token0Address, err := requiredAddress(fmt.Sprintf("pair %d token0 address", index), pair.Token0Address)
	if err != nil {
		return research.AvePairV1{}, err
	}
	token1Address, err := requiredAddress(fmt.Sprintf("pair %d token1 address", index), pair.Token1Address)
	if err != nil {
		return research.AvePairV1{}, err
	}
	reserve0, err := aveDecimal(fmt.Sprintf("pair %d reserve0", index), pair.Reserve0)
	if err != nil {
		return research.AvePairV1{}, err
	}
	reserve1, err := aveDecimal(fmt.Sprintf("pair %d reserve1", index), pair.Reserve1)
	if err != nil {
		return research.AvePairV1{}, err
	}
	volumeUSD, err := aveDecimal(fmt.Sprintf("pair %d volume usd", index), pair.VolumeU)
	if err != nil {
		return research.AvePairV1{}, err
	}
	pairMarketCap, err := aveDecimal(fmt.Sprintf("pair %d market cap", index), pair.MarketCap)
	if err != nil {
		return research.AvePairV1{}, err
	}
	pairFDV, err := aveDecimal(fmt.Sprintf("pair %d fdv", index), pair.FDV)
	if err != nil {
		return research.AvePairV1{}, err
	}
	return research.AvePairV1{
		Pair: shared.Address(pairAddress), ChainID: chainID, AMM: pair.AMM, Token0Address: shared.Address(token0Address), Token0Symbol: pair.Token0Symbol,
		Token1Address: shared.Address(token1Address), Token1Symbol: pair.Token1Symbol, Reserve0: reserve0, Reserve1: reserve1,
		VolumeUSD: volumeUSD, MarketCap: pairMarketCap, FDV: pairFDV, IsFake: pair.IsFake, CreatedAt: unixTime(pair.CreatedAt), UpdatedAt: unixTime(pair.UpdatedAt),
	}, nil
}

func aveDecimal(field, value string) (*research.Decimal, error) {
	parsed, err := research.ParseOptionalDecimal(value)
	if err != nil {
		return nil, fmt.Errorf("normalize ave %s: %w", field, err)
	}
	return parsed, nil
}

func requiredAddress(field, value string) (ethcommon.Address, error) {
	value = strings.TrimSpace(value)
	if !ethcommon.IsHexAddress(value) {
		return ethcommon.Address{}, fmt.Errorf("ave %s %q is invalid", field, value)
	}
	address := ethcommon.HexToAddress(value)
	if address == (ethcommon.Address{}) {
		return ethcommon.Address{}, fmt.Errorf("ave %s is empty", field)
	}
	return address, nil
}

func optionalBool(field, value string) (*bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return nil, fmt.Errorf("ave %s %q is invalid: %w", field, value, err)
	}
	return &parsed, nil
}

func unixTime(value int64) *time.Time {
	if value <= 0 {
		return nil
	}
	parsed := time.Unix(value, 0).UTC()
	return &parsed
}
