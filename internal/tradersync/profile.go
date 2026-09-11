package tradersync

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	tsmodel "github.com/useryege/athena/internal/tradersync/types"
	"github.com/useryege/athena/util/polymarket"
)

func scalar(value *string, source string, at time.Time, err error) tsmodel.Scalar {
	e := tsmodel.Evidence{Availability: "available", Source: source, QueriedAt: at}
	if err != nil {
		e.Availability, e.ReasonCode = "unavailable", "query_failed"
		value = nil
	} else if value == nil || *value == "" {
		e.Availability, e.ReasonCode = "unavailable", "missing_or_invalid"
		value = nil
	}
	return tsmodel.Scalar{Evidence: e, Value: value}
}

func pnlPoints(raw []polymarket.UserPNLPoint) []tsmodel.PnLPoint {
	out := make([]tsmodel.PnLPoint, len(raw))
	for i, p := range raw {
		out[i] = tsmodel.PnLPoint{T: p.T, P: *p.P.Value}
	}
	return out
}

func evidencePNL(period string, raw []polymarket.UserPNLPoint, age *time.Duration, interval, fidelity, source string, at time.Time, err error) tsmodel.PnLView {
	v := BuildPNL(period, pnlPoints(raw), age, PNLRules{})
	v.Interval, v.Fidelity = interval, fidelity
	if err != nil {
		v.Amount = tsmodel.Scalar{Evidence: unavailable("query_failed")}
		v.Curve = tsmodel.Curve{Evidence: unavailable("query_failed")}
	}
	v.Amount.Source, v.Curve.Source = source, source
	v.Amount.QueriedAt, v.Curve.QueriedAt = at, at
	return v
}

func (r *TargetResolver) loadProfile(ctx context.Context, input string) (tsmodel.ConfirmationCard, error) {
	p, e := r.profiles.ResolveIdentity(ctx, input)
	if e != nil {
		return tsmodel.ConfirmationCard{}, e
	}
	identity := identityFromProfile(p)
	wallet := strings.ToLower(identity.Wallet.Hex())
	source := p.PublicSource
	card := tsmodel.ConfirmationCard{Identity: identity, DefaultPeriod: "1Y", UsageNotice: "Choose low-frequency traders. High-frequency activity is outside the performance guarantees.", PnL: make(map[string]tsmodel.PnLView, 6)}
	card.Avatar = scalar(p.Public.ProfileImage, source, p.QueriedAt, nil)
	name := p.Public.Name
	if name == nil || *name == "" {
		name = p.Public.Pseudonym
	}
	card.DisplayName = scalar(name, source, p.QueriedAt, nil)
	var verified *string
	if p.Public.VerifiedBadge != nil {
		s := strconv.FormatBool(*p.Public.VerifiedBadge)
		verified = &s
	}
	card.Verified = scalar(verified, source, p.QueriedAt, nil)
	var all []polymarket.UserPNLPoint
	var allSource string
	var allErr error
	var allAt time.Time
	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		defer wg.Done()
		at := time.Now().UTC()
		stats, s, e := r.profiles.UserStats(ctx, wallet)
		join := stats.JoinDate
		if join != nil {
			if _, err := time.Parse(time.RFC3339Nano, *join); err != nil {
				join = nil
			}
		}
		card.JoinedAt = scalar(join, s, at, e)
		card.LargestWin = scalar(stats.LargestWin.Value, s, at, e)
	}()
	go func() {
		defer wg.Done()
		at := time.Now().UTC()
		v, s, e := r.profiles.Predictions(ctx, wallet)
		card.Predictions = scalar(v.Value, s, at, e)
	}()
	go func() {
		defer wg.Done()
		at := time.Now().UTC()
		v, s, e := r.profiles.PositionValue(ctx, wallet)
		card.PositionValue = scalar(v.Value, s, at, e)
	}()
	go func() {
		defer wg.Done()
		allAt = time.Now().UTC()
		all, allSource, allErr = r.profiles.UserPNL(ctx, wallet, "all", "1d")
	}()
	wg.Wait()
	// Request-clock age controls sampling only. It is never substituted for the
	// currently unknown official display reference time or its display-age evidence.
	var requestAge *time.Duration
	if allErr == nil && len(all) >= 2 {
		age := time.Since(time.Unix(all[0].T, 0))
		if age > 0 {
			requestAge = &age
		}
	}
	for _, period := range []string{"ALL", "1Y", "YTD"} {
		card.PnL[period] = evidencePNL(period, all, nil, "all", "1d", allSource, allAt, allErr)
	}
	views := make([]tsmodel.PnLView, 3)
	periods := []string{"1D", "1W", "1M"}
	wg.Add(3)
	for index, period := range periods {
		go func(index int, period string) {
			defer wg.Done()
			i, f := PlanPNL(period, requestAge)
			at := time.Now().UTC()
			raw, s, e := r.profiles.UserPNL(ctx, wallet, i, f)
			views[index] = evidencePNL(period, raw, nil, i, f, s, at, e)
		}(index, period)
	}
	wg.Wait()
	for index, period := range periods {
		card.PnL[period] = views[index]
	}
	return card, nil
}
