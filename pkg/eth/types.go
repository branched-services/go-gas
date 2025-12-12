// Package eth provides Ethereum node client functionality.
package eth

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/holiman/uint256"
)

// Block represents an Ethereum block with gas-relevant fields.
type Block struct {
	Number       uint64
	Hash         string
	ParentHash   string
	Timestamp    time.Time
	BaseFee      *uint256.Int // nil for pre-EIP-1559 blocks
	GasUsed      uint64
	GasLimit     uint64
	Transactions []Transaction
}

// GasUtilization returns the ratio of gas used to gas limit (0.0 to 1.0).
func (b *Block) GasUtilization() float64 {
	if b.GasLimit == 0 {
		return 0
	}
	return float64(b.GasUsed) / float64(b.GasLimit)
}

// Transaction represents an Ethereum transaction with gas-relevant fields.
type Transaction struct {
	Hash                 string
	From                 string
	To                   string // empty for contract creation
	Nonce                uint64
	GasLimit             uint64
	GasPrice             *uint256.Int // legacy transactions
	MaxFeePerGas         *uint256.Int // EIP-1559 transactions
	MaxPriorityFeePerGas *uint256.Int // EIP-1559 transactions
	Type                 uint8        // 0 = legacy, 2 = EIP-1559
}

// EffectivePriorityFee returns the priority fee that would be paid given a base fee.
// For legacy transactions, this is gasPrice - baseFee.
// For EIP-1559, this is min(maxPriorityFeePerGas, maxFeePerGas - baseFee).
func (t *Transaction) EffectivePriorityFee(baseFee *uint256.Int) *uint256.Int {
	if baseFee == nil {
		return uint256.NewInt(0)
	}

	if t.Type == 2 && t.MaxFeePerGas != nil && t.MaxPriorityFeePerGas != nil {
		// EIP-1559 transaction
		// maxMinusBase = MaxFeePerGas - BaseFee
		maxMinusBase := new(uint256.Int).Sub(t.MaxFeePerGas, baseFee)

		// if MaxPriorityFeePerGas < maxMinusBase { return MaxPriorityFeePerGas }
		if t.MaxPriorityFeePerGas.Lt(maxMinusBase) {
			return new(uint256.Int).Set(t.MaxPriorityFeePerGas)
		}
		return maxMinusBase
	}

	// Legacy transaction
	if t.GasPrice == nil {
		return uint256.NewInt(0)
	}
	// priority = GasPrice - BaseFee
	// Check for underflow (GasPrice < BaseFee)
	if t.GasPrice.Lt(baseFee) {
		return uint256.NewInt(0)
	}
	return new(uint256.Int).Sub(t.GasPrice, baseFee)
}

// IsEIP1559 returns true if this is an EIP-1559 transaction.
func (t *Transaction) IsEIP1559() bool {
	return t.Type == 2
}

// rpcBlock is the JSON-RPC representation of a block.
type rpcBlock struct {
	Number       hexUint64       `json:"number"`
	Hash         string          `json:"hash"`
	ParentHash   string          `json:"parentHash"`
	Timestamp    hexUint64       `json:"timestamp"`
	BaseFee      *hexBig         `json:"baseFeePerGas"`
	GasUsed      hexUint64       `json:"gasUsed"`
	GasLimit     hexUint64       `json:"gasLimit"`
	Transactions json.RawMessage `json:"transactions"`
}

// rpcTransaction is the JSON-RPC representation of a transaction.
type rpcTransaction struct {
	Hash                 string    `json:"hash"`
	From                 string    `json:"from"`
	To                   string    `json:"to"`
	Nonce                hexUint64 `json:"nonce"`
	Gas                  hexUint64 `json:"gas"`
	GasPrice             *hexBig   `json:"gasPrice"`
	MaxFeePerGas         *hexBig   `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *hexBig   `json:"maxPriorityFeePerGas"`
	Type                 hexUint64 `json:"type"`
}

func (r *rpcBlock) toBlock(includeTxs bool) (*Block, error) {
	block := &Block{
		Number:     uint64(r.Number),
		Hash:       r.Hash,
		ParentHash: r.ParentHash,
		Timestamp:  time.Unix(int64(r.Timestamp), 0),
		GasUsed:    uint64(r.GasUsed),
		GasLimit:   uint64(r.GasLimit),
	}

	if r.BaseFee != nil {
		block.BaseFee = r.BaseFee.Int()
	}

	if includeTxs && len(r.Transactions) > 0 && r.Transactions[0] == '{' {
		var txs []rpcTransaction
		if err := json.Unmarshal(r.Transactions, &txs); err != nil {
			return nil, fmt.Errorf("unmarshaling transactions: %w", err)
		}
		block.Transactions = make([]Transaction, len(txs))
		for i, tx := range txs {
			block.Transactions[i] = tx.toTransaction()
		}
	}

	return block, nil
}

func (r *rpcTransaction) toTransaction() Transaction {
	tx := Transaction{
		Hash:     r.Hash,
		From:     r.From,
		To:       r.To,
		Nonce:    uint64(r.Nonce),
		GasLimit: uint64(r.Gas),
		Type:     uint8(r.Type),
	}

	if r.GasPrice != nil {
		tx.GasPrice = r.GasPrice.Int()
	}
	if r.MaxFeePerGas != nil {
		tx.MaxFeePerGas = r.MaxFeePerGas.Int()
	}
	if r.MaxPriorityFeePerGas != nil {
		tx.MaxPriorityFeePerGas = r.MaxPriorityFeePerGas.Int()
	}

	return tx
}

// hexUint64 handles hex-encoded uint64 values in JSON-RPC responses.
type hexUint64 uint64

func (h *hexUint64) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	val, err := uint256.FromHex(s)
	if err != nil {
		return fmt.Errorf("invalid hex uint64: %s", s)
	}
	*h = hexUint64(val.Uint64())
	return nil
}

// hexBig handles hex-encoded big.Int values in JSON-RPC responses.
type hexBig uint256.Int

func (h *hexBig) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	val, err := uint256.FromHex(s)
	if err != nil {
		return fmt.Errorf("invalid hex big int: %s", s)
	}
	*h = hexBig(*val)
	return nil
}

func (h *hexBig) Int() *uint256.Int {
	return (*uint256.Int)(h)
}
