package tradersync

import (
	"context"
	tm "github.com/useryege/athena/internal/tradersync/types"
	"time"
)

// initializeFinality runs once for this object's clock, including across Run calls.
// A failed cutoff is a measurement gap, not a confirmation failure.
func (p *Projector) initializeFinality(ctx context.Context) {
	p.finalityOnce.Do(func() {
		defer p.finalityReady.Store(true)
		p.metrics.observationPoint()
		bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		cutoff, err := p.store.FinalityObservationCutoff(bounded)
		if err != nil {
			p.metrics.invalidateObservation()
			return
		}
		p.finalityCutoff = cutoff
	})
}
func (p *Projector) recordFinality(ctx context.Context, id int64, o tm.FinalityRoundObservation) {
	// A version failure cancels jobCtx, but cannot discard a confirmed result already
	// returned. This write remains owned by process workers.Wait and its source slot.
	bounded, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	_, current := p.metrics.observationPoint()
	o.Reliable = o.Reliable && current.Valid
	if err := p.store.RecordFinalityObservation(bounded, id, o); err != nil {
		// Bounded conservative state covers waiting rows too: a lost confirmed write
		// must never be replaced by a later first endpoint. Completed history survives.
		p.metrics.invalidateObservation()
	}
}
func (p *Projector) ObservationClock() tm.ObservationClock {
	_, clock := p.metrics.observationPoint()
	if p.finalityReady.Load() {
		clock.Cutoff = p.finalityCutoff
	} else {
		clock.Valid = false
	}
	return clock
}
