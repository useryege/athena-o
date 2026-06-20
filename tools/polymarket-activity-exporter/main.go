package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/useryege/athena/util/polymarket"
)

const defaultMarket = "0x4a2085d9d7302a081b64016124cf1981a5bcb1118392c9d645f1abb57ea14d65"

var excelHeaders = []string{
	"timestamp",
	"datetime_utc",
	"conditionId",
	"side",
	"outcome",
	"outcomeIndex",
	"price",
	"size",
	"asset",
	"proxyWallet",
	"name",
	"pseudonym",
	"transactionHash",
	"title",
	"slug",
	"eventSlug",
}

func main() {
	market := flag.String("market", defaultMarket, "Polymarket condition ID to export")
	outPath := flag.String("out", "polymarket-fifwc-tur-par-par-activity.xlsx", "Output Excel file path")
	limit := flag.Int("limit", 1000, "Page size for each API request")
	startOffset := flag.Int("start-offset", 0, "Offset to start fetching from")
	maxOffset := flag.Int("max-offset", 3000, "Maximum offset allowed by the Data API")
	filterType := flag.String("filter-type", "CASH", "Data API filterType parameter")
	filterAmount := flag.Float64("filter-amount", 3, "Data API filterAmount parameter")
	timeout := flag.Duration("timeout", 60*time.Second, "HTTP request timeout")
	flag.Parse()

	if *limit <= 0 {
		fmt.Fprintln(os.Stderr, "error: -limit must be greater than 0")
		os.Exit(1)
	}
	if *startOffset < 0 {
		fmt.Fprintln(os.Stderr, "error: -start-offset must be greater than or equal to 0")
		os.Exit(1)
	}
	if *maxOffset < 0 {
		fmt.Fprintln(os.Stderr, "error: -max-offset must be greater than or equal to 0")
		os.Exit(1)
	}
	if *startOffset > *maxOffset {
		fmt.Fprintln(os.Stderr, "error: -start-offset must be less than or equal to -max-offset")
		os.Exit(1)
	}
	if strings.TrimSpace(*filterType) != "" && *filterAmount < 0 {
		fmt.Fprintln(os.Stderr, "error: -filter-amount must be greater than or equal to 0")
		os.Exit(1)
	}

	ctx := context.Background()
	client, err := polymarket.NewDataClient(polymarket.DataConfig{Timeout: *timeout})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating data client: %v\n", err)
		os.Exit(1)
	}

	trades, err := fetchAllTrades(ctx, client, *market, *limit, *startOffset, *maxOffset, *filterType, *filterAmount)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error fetching trades: %v\n", err)
		os.Exit(1)
	}

	if err := writeExcel(*outPath, trades); err != nil {
		fmt.Fprintf(os.Stderr, "error writing excel: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("exported %d trades to %s\n", len(trades), *outPath)
}

func fetchAllTrades(
	ctx context.Context,
	client polymarket.DataClient,
	market string,
	limit int,
	startOffset int,
	maxOffset int,
	filterType string,
	filterAmount float64,
) ([]polymarket.DataTrade, error) {
	var all []polymarket.DataTrade
	offset := startOffset
	filterType = strings.TrimSpace(strings.ToUpper(filterType))

	for {
		if offset > maxOffset {
			break
		}

		limitPtr := limit
		offsetPtr := offset
		options := polymarket.ListDataTradesOptions{
			DataListOptions: polymarket.DataListOptions{
				Limit:  &limitPtr,
				Offset: &offsetPtr,
			},
			Market: []string{market},
		}
		if filterType != "" {
			options.FilterType = filterType
			options.FilterAmount = &filterAmount
		}

		rows, err := client.ListTrades(ctx, options)
		if err != nil {
			return nil, fmt.Errorf("offset %d: %w", offset, err)
		}

		fmt.Printf("fetched offset=%d count=%d total=%d\n", offset, len(rows), len(all)+len(rows))

		if len(rows) == 0 {
			break
		}

		all = append(all, rows...)
		if len(rows) < limit {
			break
		}

		offset += limit
	}

	return all, nil
}

func writeExcel(path string, trades []polymarket.DataTrade) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Activity"
	index, err := f.NewSheet(sheet)
	if err != nil {
		return err
	}
	f.SetActiveSheet(index)
	if err := f.DeleteSheet("Sheet1"); err != nil {
		return err
	}

	for col, header := range excelHeaders {
		cell, err := excelize.CoordinatesToCellName(col+1, 1)
		if err != nil {
			return err
		}
		if err := f.SetCellValue(sheet, cell, header); err != nil {
			return err
		}
	}

	for rowIdx, trade := range trades {
		values := tradeRowValues(trade)
		for col, value := range values {
			cell, err := excelize.CoordinatesToCellName(col+1, rowIdx+2)
			if err != nil {
				return err
			}
			if err := f.SetCellValue(sheet, cell, value); err != nil {
				return err
			}
		}
	}

	if err := f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	}); err != nil {
		return err
	}

	setColumnWidths(f, sheet)

	return f.SaveAs(path)
}

func tradeRowValues(trade polymarket.DataTrade) []any {
	ts := tradeInt64(trade, "timestamp")
	return []any{
		ts,
		formatTimestampUTC(ts),
		tradeString(trade, "conditionId"),
		tradeString(trade, "side"),
		tradeString(trade, "outcome"),
		tradeInt64(trade, "outcomeIndex"),
		tradeFloat64(trade, "price"),
		tradeFloat64(trade, "size"),
		tradeString(trade, "asset"),
		tradeString(trade, "proxyWallet"),
		tradeString(trade, "name"),
		tradeString(trade, "pseudonym"),
		tradeString(trade, "transactionHash"),
		tradeString(trade, "title"),
		tradeString(trade, "slug"),
		tradeString(trade, "eventSlug"),
	}
}

func tradeString(trade polymarket.DataTrade, key string) string {
	if trade == nil {
		return ""
	}
	value, ok := trade[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	default:
		return fmt.Sprint(v)
	}
}

func tradeInt64(trade polymarket.DataTrade, key string) int64 {
	if trade == nil {
		return 0
	}
	value, ok := trade[key]
	if !ok || value == nil {
		return 0
	}
	switch v := value.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case jsonNumber:
		parsed, err := v.Int64()
		if err != nil {
			return 0
		}
		return parsed
	default:
		parsed, err := strconv.ParseInt(fmt.Sprint(v), 10, 64)
		if err != nil {
			return 0
		}
		return parsed
	}
}

func tradeFloat64(trade polymarket.DataTrade, key string) float64 {
	if trade == nil {
		return 0
	}
	value, ok := trade[key]
	if !ok || value == nil {
		return 0
	}
	switch v := value.(type) {
	case float64:
		return v
	case int64:
		return float64(v)
	case int:
		return float64(v)
	case jsonNumber:
		parsed, err := v.Float64()
		if err != nil {
			return 0
		}
		return parsed
	default:
		parsed, err := strconv.ParseFloat(fmt.Sprint(v), 64)
		if err != nil {
			return 0
		}
		return parsed
	}
}

type jsonNumber interface {
	Int64() (int64, error)
	Float64() (float64, error)
}

func formatTimestampUTC(ts int64) string {
	if ts <= 0 {
		return ""
	}
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}

func setColumnWidths(f *excelize.File, sheet string) {
	widths := map[string]float64{
		"A": 14,
		"B": 22,
		"C": 70,
		"D": 8,
		"E": 12,
		"F": 14,
		"G": 10,
		"H": 12,
		"I": 24,
		"J": 44,
		"K": 24,
		"L": 24,
		"M": 70,
		"N": 40,
		"O": 32,
		"P": 32,
	}
	for col, width := range widths {
		_ = f.SetColWidth(sheet, col, col, width)
	}
}
