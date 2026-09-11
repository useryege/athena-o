package tradersync

import (
	"encoding/json"
	"math"
	"math/big"
	"strconv"
	"time"

	tsmodel "github.com/useryege/athena/internal/tradersync/types"
)

type PNLRules struct {
	Reference *time.Time
	Zone      *time.Location
	Round     func(*big.Rat) string
}

func PlanPNL(period string, allAge *time.Duration) (interval, fidelity string) {
	switch period {
	case "1D":
		interval, fidelity = "1d", "1h"
	case "1W":
		interval, fidelity = "1w", "3h"
	case "1M":
		interval, fidelity = "1m", "18h"
	case "1Y", "YTD", "ALL":
		interval, fidelity = "all", "1d"
	default:
		return "", ""
	}
	if allAge == nil || *allAge <= 0 {
		return
	}
	h, n := allAge.Hours(), 60.0
	switch period {
	case "1D":
		h, n = math.Min(24, h), 24
	case "1W":
		h, n = math.Min(168, h), 56
	case "1M":
		h, n = math.Min(720, h), 48
		if *allAge < 31*24*time.Hour {
			interval, n = "all", 60
		}
	}
	best, err := 1, math.Inf(1)
	for _, v := range []int{1, 3, 12, 18, 24} {
		if float64(v) > h {
			continue
		}
		if d := math.Abs(h/float64(v) - n); d < err {
			best, err = v, d
		}
	}
	fidelity = strconv.Itoa(best) + "h"
	if best == 24 {
		fidelity = "1d"
	}
	return
}

func unavailable(reason string) tsmodel.Evidence {
	return tsmodel.Evidence{Availability: "unavailable", ReasonCode: reason}
}

func BuildPNL(period string, raw []tsmodel.PnLPoint, allAge *time.Duration, rules PNLRules) tsmodel.PnLView {
	i, f := PlanPNL(period, allAge)
	v := tsmodel.PnLView{Interval: i, Fidelity: f, Reference: rules.Reference, Amount: tsmodel.Scalar{Evidence: unavailable("invalid_series")}, Curve: tsmodel.Curve{Evidence: unavailable("invalid_series")}}
	if rules.Zone != nil {
		v.Timezone = rules.Zone.String()
	}
	fail := func(reason string) tsmodel.PnLView {
		v.Curve.Evidence = unavailable(reason)
		v.Amount.Evidence = unavailable(reason)
		return v
	}
	if i == "" {
		return fail("invalid_period")
	}
	if len(raw) < 2 {
		return fail("insufficient_points")
	}
	for j, p := range raw {
		if len(p.P) == 0 || !json.Valid([]byte(p.P)) || (p.P[0] != '-' && (p.P[0] < '0' || p.P[0] > '9')) {
			return fail("invalid_number")
		}
		if _, ok := new(big.Rat).SetString(p.P); !ok {
			return fail("invalid_number")
		}
		if j > 0 && p.T < raw[j-1].T {
			return fail("time_reversed")
		}
	}
	points := append([]tsmodel.PnLPoint(nil), raw...)
	v.Curve.Points = append([]tsmodel.PnLPoint(nil), raw...)
	threshold := time.Duration(0)
	if period != "ALL" {
		if rules.Reference == nil {
			return fail("reference_unknown")
		}
		switch period {
		case "1D":
			threshold = 24 * time.Hour
		case "1W":
			threshold = 7 * 24 * time.Hour
		case "1M":
			threshold = 30 * 24 * time.Hour
		case "1Y":
			threshold = 365 * 24 * time.Hour
		case "YTD":
			if rules.Zone == nil {
				return fail("timezone_unknown")
			}
			ref := rules.Reference.In(rules.Zone)
			threshold = ref.Sub(time.Date(ref.Year(), 1, 1, 0, 0, 0, 0, rules.Zone))
		}
		lower := rules.Reference.Add(-threshold)
		points = nil
		for _, p := range raw {
			if !time.Unix(p.T, 0).Before(lower) {
				points = append(points, p)
			}
		}
		if len(points) < 2 {
			return fail("insufficient_cropped_points")
		}
		if period == "1M" {
			threshold = 31 * 24 * time.Hour
		}
	}
	v.Curve = tsmodel.Curve{Evidence: tsmodel.Evidence{Availability: "available"}, Points: points}
	if period != "ALL" && (allAge == nil || *allAge < 0) {
		v.Amount.Evidence = unavailable("all_age_unknown")
		return v
	}
	if rules.Round == nil {
		v.Amount.Evidence = unavailable("rounding_unknown")
		return v
	}
	end, _ := new(big.Rat).SetString(points[len(points)-1].P)
	amount := new(big.Rat).Set(end)
	if period != "ALL" && *allAge >= threshold {
		begin, _ := new(big.Rat).SetString(points[0].P)
		amount.Sub(end, begin)
	}
	value := rules.Round(amount)
	v.Amount = tsmodel.Scalar{Evidence: tsmodel.Evidence{Availability: "available"}, Value: &value}
	return v
}
