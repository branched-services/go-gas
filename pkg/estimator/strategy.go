package estimator

import "context"

// Strategy defines the interface for gas estimation algorithms.
// Implementations must be stateless and safe for concurrent use.
//
// Open/Closed Principle: New estimation strategies can be added
// without modifying existing code.
type Strategy interface {
	// Calculate computes a gas estimate from the provided input.
	// The implementation should be pure (no side effects) and deterministic.
	Calculate(ctx context.Context, input *CalculatorInput) (*GasEstimate, error)

	// Name returns a human-readable name for the strategy.
	// Used for logging and metrics.
	Name() string
}
