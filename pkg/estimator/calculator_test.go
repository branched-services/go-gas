package estimator

import (
	"context"
	"testing"
	"time"

	"github.com/holiman/uint256"
)

func TestHybridStrategy_Calculate(t *testing.T) {
	// Helper to create uint256.Int
	u256 := func(v uint64) *uint256.Int {
		return uint256.NewInt(v)
	}

	// Helper to create a block
	makeBlock := func(number uint64, baseFee uint64, gasUsed, gasLimit uint64, priorityFees []uint64) *BlockData {
		fees := make([]*uint256.Int, len(priorityFees))
		for i, f := range priorityFees {
			fees[i] = u256(f)
		}
		return &BlockData{
			Number:       number,
			Timestamp:    time.Now(),
			BaseFee:      u256(baseFee),
			GasUsed:      gasUsed,
			GasLimit:     gasLimit,
			PriorityFees: fees,
		}
	}

	defaultStrategy := DefaultStrategy()

	tests := []struct {
		name        string
		strategy    *HybridStrategy
		input       *CalculatorInput
		wantBaseFee *uint256.Int
		wantUrgent  *uint256.Int // Expected Urgent Priority Fee
		wantErr     bool
	}{
		{
			name:     "Not ready (no current block)",
			strategy: defaultStrategy,
			input:    &CalculatorInput{},
			wantErr:  true,
		},
		{
			name:     "Base fee prediction - target usage",
			strategy: defaultStrategy,
			input: &CalculatorInput{
				ChainID:      1,
				CurrentBlock: makeBlock(100, 1000000000, 15000000, 30000000, nil), // 50% usage
			},
			wantBaseFee: u256(1000000000), // Should stay same
		},
		{
			name:     "Base fee prediction - full block",
			strategy: defaultStrategy,
			input: &CalculatorInput{
				ChainID:      1,
				CurrentBlock: makeBlock(100, 1000000000, 30000000, 30000000, nil), // 100% usage
			},
			// Delta = 1000000000 * (30000000 - 15000000) / 15000000 / 8 = 1000000000 * 1 / 8 = 125000000
			// New BaseFee = 1000000000 + 125000000 = 1125000000
			wantBaseFee: u256(1125000000),
		},
		{
			name:     "Base fee prediction - empty block",
			strategy: defaultStrategy,
			input: &CalculatorInput{
				ChainID:      1,
				CurrentBlock: makeBlock(100, 1000000000, 0, 30000000, nil), // 0% usage
			},
			// Delta = 1000000000 * (15000000 - 0) / 15000000 / 8 = 125000000
			// New BaseFee = 1000000000 - 125000000 = 875000000
			wantBaseFee: u256(875000000),
		},
		{
			name:     "No data - defaults",
			strategy: defaultStrategy,
			input: &CalculatorInput{
				ChainID:      1,
				CurrentBlock: makeBlock(100, 1000000000, 15000000, 30000000, nil),
			},
			wantBaseFee: u256(1000000000),
			// Default Urgent (99%)
			// Min: 1 gwei, Max: 500 gwei
			// Diff: 499 gwei
			// Scaled: 499 * 99 / 100 = 494.01 gwei
			// Result: 1 + 494.01 = 495.01 gwei = 495010000000
			wantUrgent: u256(495010000000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.strategy.Calculate(context.Background(), tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Calculate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if !got.BaseFee.Eq(tt.wantBaseFee) {
				t.Errorf("Calculate() BaseFee = %v, want %v", got.BaseFee, tt.wantBaseFee)
			}

			if tt.wantUrgent != nil {
				if !got.Urgent.MaxPriorityFeePerGas.Eq(tt.wantUrgent) {
					t.Errorf("Calculate() Urgent Priority = %v, want %v", got.Urgent.MaxPriorityFeePerGas, tt.wantUrgent)
				}
			}
		})
	}
}

func TestHybridStrategy_Blend(t *testing.T) {
	s := DefaultStrategy()
	u256 := func(v uint64) *uint256.Int { return uint256.NewInt(v) }

	tests := []struct {
		name    string
		a, b    *uint256.Int
		weightA float64
		want    *uint256.Int
	}{
		{
			name:    "Equal weights",
			a:       u256(100),
			b:       u256(200),
			weightA: 0.5,
			want:    u256(150),
		},
		{
			name:    "Full weight A",
			a:       u256(100),
			b:       u256(200),
			weightA: 1.0,
			want:    u256(100),
		},
		{
			name:    "Full weight B",
			a:       u256(100),
			b:       u256(200),
			weightA: 0.0,
			want:    u256(200),
		},
		{
			name:    "75-25 split",
			a:       u256(100),
			b:       u256(200),
			weightA: 0.75,
			// 100 * 0.75 + 200 * 0.25 = 75 + 50 = 125
			want: u256(125),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.blend(tt.a, tt.b, tt.weightA)
			if !got.Eq(tt.want) {
				t.Errorf("blend() = %v, want %v", got, tt.want)
			}
		})
	}
}
