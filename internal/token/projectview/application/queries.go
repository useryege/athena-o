package application

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/useryege/athena/internal/token/projectview"
	"github.com/useryege/athena/internal/token/research"
	"github.com/useryege/athena/internal/token/shared"
	"github.com/useryege/athena/internal/token/swap"
)

const maxTrendPoints = 500

type ReadRepository interface {
	GetProjectDetail(context.Context, int64) (*projectview.Detail, error)
	GetProjectSwapActivity(context.Context, int64) (*projectview.SwapActivity, error)
	ListProjectSwapEventsPage(context.Context, int64, swap.PairKind, uint64, int32, int32) (*projectview.SwapEventPage, error)
	ListProjectObservationsPage(context.Context, int64, string, int32, int32) (*projectview.ObservationPage, error)
	ListProjectTrendObservations(context.Context, int64, time.Time) ([]research.ProjectObservation, error)
	ListProjectWalletNormalTransactionsPage(context.Context, int64, shared.Address, string, string, int32, int32) (*projectview.WalletNormalTransactionPage, error)
}

type Queries struct {
	repository ReadRepository
	now        func() time.Time
}

func NewQueries(repository ReadRepository) *Queries {
	return &Queries{repository: repository, now: time.Now}
}

func (queries *Queries) GetProjectDetail(ctx context.Context, projectID int64) (*projectview.Detail, error) {
	return queries.repository.GetProjectDetail(ctx, projectID)
}

func (queries *Queries) GetProjectSwapActivity(ctx context.Context, projectID int64) (*projectview.SwapActivity, error) {
	activity, err := queries.repository.GetProjectSwapActivity(ctx, projectID)
	if err != nil || activity == nil {
		return activity, err
	}
	activity.GeneratedAt = queries.now().UTC()
	return activity, nil
}

func (queries *Queries) ListProjectSwapEventsPage(
	ctx context.Context,
	projectID int64,
	pairKind swap.PairKind,
	blockNumber uint64,
	page, pageSize int32,
) (*projectview.SwapEventPage, error) {
	switch pairKind {
	case swap.PairKindWETH, swap.PairKindUSDT:
	default:
		return nil, fmt.Errorf("pair kind must be weth or usdt")
	}
	if blockNumber == 0 {
		return nil, fmt.Errorf("block number must be positive")
	}
	return queries.repository.ListProjectSwapEventsPage(ctx, projectID, pairKind, blockNumber, page, pageSize)
}

func (queries *Queries) ListProjectObservationsPage(ctx context.Context, projectID int64, dataType string, page, pageSize int32) (*projectview.ObservationPage, error) {
	return queries.repository.ListProjectObservationsPage(ctx, projectID, dataType, page, pageSize)
}

func (queries *Queries) ListProjectWalletNormalTransactionsPage(
	ctx context.Context,
	projectID int64,
	wallet shared.Address,
	receiptStatus string,
	methodID string,
	page, pageSize int32,
) (*projectview.WalletNormalTransactionPage, error) {
	return queries.repository.ListProjectWalletNormalTransactionsPage(ctx, projectID, wallet, receiptStatus, methodID, page, pageSize)
}

func (queries *Queries) ListProjectTrends(ctx context.Context, projectID int64, rangeValue string) (*projectview.TrendResult, error) {
	duration, normalizedRange, err := parseTrendRange(rangeValue)
	if err != nil {
		return nil, err
	}
	generatedAt := queries.now().UTC()
	observedFrom := generatedAt.Add(-duration)
	observations, err := queries.repository.ListProjectTrendObservations(ctx, projectID, observedFrom)
	if err != nil {
		return nil, err
	}
	series, err := buildTrendSeries(observations)
	if err != nil {
		return nil, err
	}
	return &projectview.TrendResult{
		Range:        normalizedRange,
		ObservedFrom: observedFrom,
		GeneratedAt:  generatedAt,
		Series:       series,
	}, nil
}

func parseTrendRange(value string) (time.Duration, string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "24h"
	}
	switch value {
	case "1h":
		return time.Hour, value, nil
	case "6h":
		return 6 * time.Hour, value, nil
	case "24h":
		return 24 * time.Hour, value, nil
	case "7d":
		return 7 * 24 * time.Hour, value, nil
	default:
		return 0, "", fmt.Errorf("trend range must be one of 1h, 6h, 24h, 7d")
	}
}

type trendMetric struct {
	Key      string
	Label    string
	Unit     string
	DataType research.DataCollectionType
}

var trendMetrics = []trendMetric{
	{Key: "price_usd", Label: "Price USD", Unit: "usd", DataType: research.DataCollectionTypeAve},
	{Key: "market_cap", Label: "Market Cap", Unit: "usd", DataType: research.DataCollectionTypeAve},
	{Key: "fdv", Label: "FDV", Unit: "usd", DataType: research.DataCollectionTypeAve},
	{Key: "tvl", Label: "TVL", Unit: "usd", DataType: research.DataCollectionTypeAve},
	{Key: "main_pair_tvl", Label: "Main Pair TVL", Unit: "usd", DataType: research.DataCollectionTypeAve},
	{Key: "holders", Label: "Holders", Unit: "count", DataType: research.DataCollectionTypeAve},
	{Key: "risk_score", Label: "Risk Score", Unit: "score", DataType: research.DataCollectionTypeAve},
	{Key: "weth_quote_usdt", Label: "Wrapped Native Pair USDT Value", Unit: "usd", DataType: research.DataCollectionTypeChainState},
	{Key: "usdt_quote_usdt", Label: "USDT Pair USDT Value", Unit: "usd", DataType: research.DataCollectionTypeChainState},
}

type numericTrendPoint struct {
	projectview.TrendPoint
	numeric float64
}

func buildTrendSeries(observations []research.ProjectObservation) ([]projectview.TrendSeries, error) {
	points := make(map[string][]numericTrendPoint, len(trendMetrics))
	for _, observation := range observations {
		values, err := observationTrendValues(observation)
		if err != nil {
			return nil, fmt.Errorf("decode project observation %d for trends: %w", observation.ID, err)
		}
		for key, value := range values {
			numeric, err := strconv.ParseFloat(value, 64)
			if err != nil || math.IsInf(numeric, 0) || math.IsNaN(numeric) {
				continue
			}
			points[key] = append(points[key], numericTrendPoint{
				TrendPoint: projectview.TrendPoint{ObservedAt: observation.ObservedAt, Value: value},
				numeric:    numeric,
			})
		}
	}
	result := make([]projectview.TrendSeries, 0, len(trendMetrics))
	for _, metric := range trendMetrics {
		values := points[metric.Key]
		if len(values) == 0 {
			continue
		}
		sort.SliceStable(values, func(i, j int) bool {
			return values[i].ObservedAt.Before(values[j].ObservedAt)
		})
		values = downsampleLTTB(values, maxTrendPoints)
		displayPoints := make([]projectview.TrendPoint, 0, len(values))
		for _, value := range values {
			displayPoints = append(displayPoints, value.TrendPoint)
		}
		result = append(result, projectview.TrendSeries{
			Key:      metric.Key,
			Label:    metric.Label,
			Unit:     metric.Unit,
			DataType: metric.DataType,
			Points:   displayPoints,
		})
	}
	return result, nil
}

func observationTrendValues(observation research.ProjectObservation) (map[string]string, error) {
	switch observation.DataType {
	case research.DataCollectionTypeAve:
		var value research.AveObservationV1
		if err := json.Unmarshal(observation.Payload, &value); err != nil {
			return nil, err
		}
		result := map[string]string{"holders": strconv.Itoa(value.Token.Holders)}
		addDecimalTrend(result, "price_usd", value.Token.CurrentPriceUSD)
		addDecimalTrend(result, "market_cap", value.Token.MarketCap)
		addDecimalTrend(result, "fdv", value.Token.FDV)
		addDecimalTrend(result, "tvl", value.Token.TVL)
		addDecimalTrend(result, "main_pair_tvl", value.Token.MainPairTVL)
		addDecimalTrend(result, "risk_score", value.Token.RiskScore)
		return result, nil
	case research.DataCollectionTypeChainState:
		var value research.ChainStateObservationV1
		if err := json.Unmarshal(observation.Payload, &value); err != nil {
			return nil, err
		}
		result := make(map[string]string, 2)
		if value.WethPair.QuoteUsdtValueInt != nil {
			result["weth_quote_usdt"] = value.WethPair.QuoteUsdtValueInt.String()
		}
		if value.UsdtPair.QuoteUsdtValueInt != nil {
			result["usdt_quote_usdt"] = value.UsdtPair.QuoteUsdtValueInt.String()
		}
		return result, nil
	default:
		return nil, nil
	}
}

func addDecimalTrend(target map[string]string, key string, value *research.Decimal) {
	if value != nil {
		target[key] = value.String()
	}
}

func downsampleLTTB(values []numericTrendPoint, threshold int) []numericTrendPoint {
	if threshold >= len(values) || threshold < 3 {
		return values
	}
	sampled := make([]numericTrendPoint, 0, threshold)
	sampled = append(sampled, values[0])
	bucketSize := float64(len(values)-2) / float64(threshold-2)
	selectedIndex := 0
	for bucket := 0; bucket < threshold-2; bucket++ {
		averageStart := int(math.Floor(float64(bucket+1)*bucketSize)) + 1
		averageEnd := int(math.Floor(float64(bucket+2)*bucketSize)) + 1
		if averageEnd > len(values) {
			averageEnd = len(values)
		}
		if averageStart >= averageEnd {
			averageStart = averageEnd - 1
		}
		var averageX, averageY float64
		for index := averageStart; index < averageEnd; index++ {
			averageX += float64(values[index].ObservedAt.UnixMilli())
			averageY += values[index].numeric
		}
		averageRange := float64(averageEnd - averageStart)
		averageX /= averageRange
		averageY /= averageRange

		rangeStart := int(math.Floor(float64(bucket)*bucketSize)) + 1
		rangeEnd := int(math.Floor(float64(bucket+1)*bucketSize)) + 1
		if rangeEnd > len(values)-1 {
			rangeEnd = len(values) - 1
		}
		pointAX := float64(values[selectedIndex].ObservedAt.UnixMilli())
		pointAY := values[selectedIndex].numeric
		maxArea := -1.0
		nextIndex := rangeStart
		for index := rangeStart; index < rangeEnd; index++ {
			area := math.Abs((pointAX-averageX)*(values[index].numeric-pointAY)-
				(pointAX-float64(values[index].ObservedAt.UnixMilli()))*(averageY-pointAY)) * 0.5
			if area > maxArea {
				maxArea = area
				nextIndex = index
			}
		}
		sampled = append(sampled, values[nextIndex])
		selectedIndex = nextIndex
	}
	return append(sampled, values[len(values)-1])
}
