package estimator

import (
	"context"
	"errors"
	"sync/atomic"
)

// ErrNotReady indicates the estimator has not produced its first estimate.
var ErrNotReady = errors.New("estimator not ready")

// EstimateReader provides read-only access to gas estimates.
// Implemented by Provider; consumers should depend on this interface.
type EstimateReader interface {
	Current(ctx context.Context) (*GasEstimate, error)
}

// ReadinessChecker provides health check functionality.
// Implemented by Provider; used by health probes.
type ReadinessChecker interface {
	Ready() bool
}

// Provider serves pre-computed gas estimates.
//
// Design:
// - Writes happen when a new estimate is computed (~every block or recalc interval)
// - Reads happen on every API request (potentially thousands per second)
// - atomic.Pointer provides lock-free reads with zero allocations
//
// Thread safety: All methods are safe for concurrent use.
type Provider struct {
	current atomic.Pointer[GasEstimate]
	updates atomic.Uint64 // total number of updates (for metrics)
}

// NewProvider creates a new Provider.
func NewProvider() *Provider {
	return &Provider{}
}

// Update atomically replaces the current estimate.
// The provided estimate should be treated as immutable after this call.
func (p *Provider) Update(est *GasEstimate) {
	p.current.Store(est)
	p.updates.Add(1)
}

// Current returns the latest gas estimate.
// Returns ErrNotReady if no estimate has been computed yet.
//
// This is the hot path - must be as fast as possible.
// Single atomic load, no allocations, no locks.
func (p *Provider) Current(ctx context.Context) (*GasEstimate, error) {
	// Check context first to support request cancellation
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	est := p.current.Load()
	if est == nil {
		return nil, ErrNotReady
	}
	return est, nil
}

// Ready returns true if at least one estimate has been computed.
// Used for health/readiness checks.
func (p *Provider) Ready() bool {
	return p.current.Load() != nil
}

// UpdateCount returns the total number of estimate updates.
// Useful for metrics and debugging.
func (p *Provider) UpdateCount() uint64 {
	return p.updates.Load()
}

// Verify interface compliance at compile time.
var (
	_ EstimateReader   = (*Provider)(nil)
	_ ReadinessChecker = (*Provider)(nil)
)
