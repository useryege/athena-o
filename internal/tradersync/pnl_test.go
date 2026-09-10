package tradersync

import (
	"math/big"
	"testing"
	"time"

	tsmodel "github.com/useryege/athena/internal/tradersync/types"
)

func TestMonthQueryUsesStrict31DayBoundary(t *testing.T) {
	short, exact := 31*24*time.Hour-time.Nanosecond, 31*24*time.Hour
	if i, _ := PlanPNL("1M", &short); i != "all" {
		t.Fatal(i)
	}
	if i, _ := PlanPNL("1M", &exact); i != "1m" {
		t.Fatal(i)
	}
	for _, p := range []string{"1Y", "YTD", "ALL"} {
		if i, f := PlanPNL(p, nil); i != "all" || f != "1d" {
			t.Fatal(p, i, f)
		}
	}
}

func TestPNLRequests(t *testing.T) {
	for _, tt := range []struct {
		p    string
		days float64
		i, f string
	}{{"1D", 0, "1d", "1h"}, {"1W", 0, "1w", "3h"}, {"1M", 0, "1m", "18h"}, {"1W", 1, "1w", "1h"}, {"1M", 30, "all", "12h"}, {"1M", 31, "1m", "18h"}, {"ALL", 60, "all", "1d"}, {"ALL", 5, "all", "3h"}} {
		a := time.Duration(tt.days*24) * time.Hour
		if i, f := PlanPNL(tt.p, &a); i != tt.i || f != tt.f {
			t.Fatalf("%+v: %s/%s", tt, i, f)
		}
	}
}

func TestPNLCropAmountsAndEvidence(t *testing.T) {
	ref := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	round := func(r *big.Rat) string { return r.FloatString(2) }
	for _, tt := range []struct {
		p    string
		age  time.Duration
		span time.Duration
		want string
	}{{"1D", 24 * time.Hour, 24 * time.Hour, "-3.25"}, {"1D", 23 * time.Hour, 24 * time.Hour, "-5.25"}, {"1W", 7 * 24 * time.Hour, 7 * 24 * time.Hour, "-3.25"}, {"1M", 31 * 24 * time.Hour, 30 * 24 * time.Hour, "-3.25"}, {"1M", 31*24*time.Hour - time.Nanosecond, 30 * 24 * time.Hour, "-5.25"}, {"1Y", 365 * 24 * time.Hour, 365 * 24 * time.Hour, "-3.25"}, {"YTD", 24 * time.Hour, 24 * time.Hour, "-3.25"}, {"ALL", 0, 1000 * time.Hour, "-5.25"}} {
		raw := []tsmodel.PnLPoint{{T: ref.Add(-tt.span - time.Second).Unix(), P: "999"}, {T: ref.Add(-tt.span).Unix(), P: "-2.00"}, {T: ref.Unix(), P: "-5.25"}}
		v := BuildPNL(tt.p, raw, &tt.age, PNLRules{Reference: &ref, Zone: time.UTC, Round: round})
		if v.Amount.Value == nil || *v.Amount.Value != tt.want || v.Curve.Availability != "available" {
			t.Fatalf("%s %+v", tt.p, v)
		}
		if tt.p != "ALL" && (len(v.Curve.Points) != 2 || v.Curve.Points[0].P != "-2.00") {
			t.Fatal(v.Curve)
		}
	}
	raw := []tsmodel.PnLPoint{{T: ref.Unix(), P: "0"}, {T: ref.Unix(), P: "0"}}
	age := 48 * time.Hour
	v := BuildPNL("ALL", raw, nil, PNLRules{})
	if v.Curve.Availability != "available" || v.Amount.Value != nil || len(v.Curve.Points) != 2 {
		t.Fatal(v)
	}
	v = BuildPNL("1D", raw, &age, PNLRules{Reference: &ref, Round: round})
	if v.Amount.Value == nil || *v.Amount.Value != "0.00" {
		t.Fatal(v)
	}
	for _, p := range []string{"1D", "1W", "1M", "1Y", "YTD"} {
		v = BuildPNL(p, raw, &age, PNLRules{Round: round})
		if v.Curve.Availability != "unavailable" {
			t.Fatal(p, v)
		}
	}
	v = BuildPNL("YTD", raw, &age, PNLRules{Reference: &ref, Round: round})
	if v.Curve.ReasonCode != "timezone_unknown" {
		t.Fatal(v)
	}
	v = BuildPNL("1D", raw, nil, PNLRules{Reference: &ref, Round: round})
	if v.Curve.Availability != "available" || v.Amount.ReasonCode != "all_age_unknown" {
		t.Fatal(v)
	}
	for _, bad := range [][]tsmodel.PnLPoint{nil, {{T: 1, P: "0"}}, {{T: 2, P: "0"}, {T: 1, P: "0"}}, {{T: 1, P: "NaN"}, {T: 2, P: "0"}}} {
		v = BuildPNL("ALL", bad, nil, PNLRules{Round: round})
		if v.Curve.Availability != "unavailable" || v.Amount.Value != nil {
			t.Fatal(v)
		}
	}
}

func TestPNLMissingRulesRetainsRawEvidenceAndRejectsNonDecimals(t *testing.T) {
	raw := []tsmodel.PnLPoint{{T: 1, P: "-9007199254740993.00"}, {T: 2, P: "0"}}
	v := BuildPNL("1D", raw, nil, PNLRules{})
	if v.Curve.Availability != "unavailable" || len(v.Curve.Points) != 2 || v.Curve.Points[0].P != "-9007199254740993.00" {
		t.Fatal(v)
	}
	for _, bad := range []string{"1/2", "0x10", "+1"} {
		points := []tsmodel.PnLPoint{{T: 1, P: bad}, {T: 2, P: "0"}}
		if v := BuildPNL("ALL", points, nil, PNLRules{}); v.Curve.Availability != "unavailable" {
			t.Fatal(bad, v)
		}
	}
}

func TestPNLYTDLocalYearBoundaryAndStrictAge(t *testing.T) {
	zone := time.FixedZone("UTC+8", 8*3600)
	ref := time.Date(2025, 12, 31, 18, 0, 0, 0, time.UTC)
	raw := []tsmodel.PnLPoint{{T: time.Date(2025, 12, 31, 15, 59, 59, 0, time.UTC).Unix(), P: "100"}, {T: time.Date(2025, 12, 31, 16, 0, 0, 0, time.UTC).Unix(), P: "7.125"}, {T: ref.Unix(), P: "9.250"}}
	round := func(r *big.Rat) string { return r.FloatString(3) }
	age := 2 * time.Hour
	v := BuildPNL("YTD", raw, &age, PNLRules{Reference: &ref, Zone: zone, Round: round})
	if v.Amount.Value == nil || *v.Amount.Value != "2.125" || len(v.Curve.Points) != 2 {
		t.Fatal(v)
	}
	age -= time.Nanosecond
	v = BuildPNL("YTD", raw, &age, PNLRules{Reference: &ref, Zone: zone, Round: round})
	if v.Amount.Value == nil || *v.Amount.Value != "9.250" {
		t.Fatal(v)
	}
	tie := 90 * time.Hour
	if _, f := PlanPNL("ALL", &tie); f != "1h" {
		t.Fatal("tie must prefer smaller candidate", f)
	}
}
