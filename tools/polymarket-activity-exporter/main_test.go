package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/useryege/athena/util/polymarket"
)

func TestWriteExcelRoundTrip(t *testing.T) {
	header := []string{
		"timestamp", "datetime_utc", "conditionId", "side", "outcome", "outcomeIndex",
		"price", "size", "asset", "proxyWallet", "name", "pseudonym",
		"transactionHash", "title", "slug", "eventSlug",
	}
	tests := []struct {
		name   string
		trades []polymarket.DataTrade
		rows   [][]string
	}{
		{
			name: "empty data keeps the header",
			rows: [][]string{header},
		},
		{
			name: "ordinary trade keeps every field and numeric value",
			trades: []polymarket.DataTrade{{
				"timestamp":       int64(1704067200),
				"conditionId":     "0xcondition",
				"side":            "BUY",
				"outcome":         "Yes",
				"outcomeIndex":    1,
				"price":           0.625,
				"size":            12.5,
				"asset":           "0xasset",
				"proxyWallet":     "0xwallet",
				"name":            "Alice",
				"pseudonym":       "trader-a",
				"transactionHash": "0xtransaction",
				"title":           "Will it rain?",
				"slug":            "will-it-rain",
				"eventSlug":       "weather-event",
			}},
			rows: [][]string{
				header,
				{
					"1704067200", "2024-01-01T00:00:00Z", "0xcondition", "BUY", "Yes", "1",
					"0.625", "12.5", "0xasset", "0xwallet", "Alice", "trader-a",
					"0xtransaction", "Will it rain?", "will-it-rain", "weather-event",
				},
			},
		},
		{
			name: "Unicode trade survives save and reopen",
			trades: []polymarket.DataTrade{{
				"timestamp":       int64(1704153600),
				"conditionId":     "条件-α",
				"side":            "SELL",
				"outcome":         "是",
				"outcomeIndex":    0,
				"price":           0.25,
				"size":            3.75,
				"asset":           "资产-β",
				"proxyWallet":     "钱包-γ",
				"name":            "交易员 李",
				"pseudonym":       "café☕",
				"transactionHash": "交易-hash",
				"title":           "世界杯 🏆 结果",
				"slug":            "比赛-结果",
				"eventSlug":       "事件-Δ",
			}},
			rows: [][]string{
				header,
				{
					"1704153600", "2024-01-02T00:00:00Z", "条件-α", "SELL", "是", "0",
					"0.25", "3.75", "资产-β", "钱包-γ", "交易员 李", "café☕",
					"交易-hash", "世界杯 🏆 结果", "比赛-结果", "事件-Δ",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "activity.xlsx")
			if err := writeExcel(path, tt.trades); err != nil {
				t.Fatalf("writeExcel: %v", err)
			}

			f, err := excelize.OpenFile(path)
			if err != nil {
				t.Fatalf("open exported workbook: %v", err)
			}
			defer f.Close()

			if sheets := f.GetSheetList(); !reflect.DeepEqual(sheets, []string{"Activity"}) {
				t.Errorf("sheet list = %q, want only Activity", sheets)
			}
			if active := f.GetSheetName(f.GetActiveSheetIndex()); active != "Activity" {
				t.Errorf("active sheet = %q, want Activity", active)
			}

			rows, err := f.GetRows("Activity")
			if err != nil {
				t.Fatalf("read exported rows: %v", err)
			}
			if !reflect.DeepEqual(rows, tt.rows) {
				t.Errorf("exported rows = %#v, want %#v", rows, tt.rows)
			}

			panes, err := f.GetPanes("Activity")
			if err != nil {
				t.Fatalf("read panes: %v", err)
			}
			if !panes.Freeze || panes.Split || panes.XSplit != 0 || panes.YSplit != 1 ||
				panes.TopLeftCell != "A2" || panes.ActivePane != "bottomLeft" {
				t.Errorf("panes = %+v, want first row frozen with A2 visible", panes)
			}

			widths := map[string]float64{
				"A": 14, "B": 22, "C": 70, "D": 8, "E": 12, "F": 14,
				"G": 10, "H": 12, "I": 24, "J": 44, "K": 24, "L": 24,
				"M": 70, "N": 40, "O": 32, "P": 32,
			}
			for col, want := range widths {
				got, err := f.GetColWidth("Activity", col)
				if err != nil {
					t.Fatalf("read column %s width: %v", col, err)
				}
				if got != want {
					t.Errorf("column %s width = %g, want %g", col, got, want)
				}
			}
		})
	}
}
