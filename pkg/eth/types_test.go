package eth

import (
	"testing"

	"github.com/holiman/uint256"
)

func TestTransaction_EffectivePriorityFee(t *testing.T) {
	u256 := func(v uint64) *uint256.Int { return uint256.NewInt(v) }

	tests := []struct {
		name    string
		tx      *Transaction
		baseFee *uint256.Int
		want    *uint256.Int
	}{
		{
			name: "EIP-1559: MaxFee > BaseFee + Priority",
			tx: &Transaction{
				Type:                 2,
				MaxFeePerGas:         u256(100),
				MaxPriorityFeePerGas: u256(10),
			},
			baseFee: u256(50),
			// Cap = 100 - 50 = 50. Priority = 10. Min(10, 50) = 10.
			want: u256(10),
		},
		{
			name: "EIP-1559: MaxFee < BaseFee + Priority",
			tx: &Transaction{
				Type:                 2,
				MaxFeePerGas:         u256(60),
				MaxPriorityFeePerGas: u256(20),
			},
			baseFee: u256(50),
			// Cap = 60 - 50 = 10. Priority = 20. Min(20, 10) = 10.
			want: u256(10),
		},
		{
			name: "EIP-1559: MaxFee < BaseFee",
			tx: &Transaction{
				Type:                 2,
				MaxFeePerGas:         u256(40),
				MaxPriorityFeePerGas: u256(10),
			},
			baseFee: u256(50),
			want:    u256(0),
		},
		{
			name: "Legacy: GasPrice > BaseFee",
			tx: &Transaction{
				Type:     0,
				GasPrice: u256(100),
			},
			baseFee: u256(50),
			// 100 - 50 = 50
			want: u256(50),
		},
		{
			name: "Legacy: GasPrice < BaseFee",
			tx: &Transaction{
				Type:     0,
				GasPrice: u256(40),
			},
			baseFee: u256(50),
			want:    u256(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.tx.EffectivePriorityFee(tt.baseFee)
			if !got.Eq(tt.want) {
				t.Errorf("EffectivePriorityFee() = %v, want %v", got, tt.want)
			}
		})
	}
}
