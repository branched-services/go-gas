package estimator

import (
	"context"

	"github.com/branched-services/go-gas/pkg/eth"
	"github.com/holiman/uint256"
)

type mockBlockReader struct {
	blockByNumberFunc func(ctx context.Context, number *uint256.Int) (*eth.Block, error)
	latestBlockFunc   func(ctx context.Context) (*eth.Block, error)
	chainIDFunc       func(ctx context.Context) (uint64, error)
}

func (m *mockBlockReader) BlockByNumber(ctx context.Context, number *uint256.Int) (*eth.Block, error) {
	if m.blockByNumberFunc != nil {
		return m.blockByNumberFunc(ctx, number)
	}
	return nil, nil
}

func (m *mockBlockReader) LatestBlock(ctx context.Context) (*eth.Block, error) {
	if m.latestBlockFunc != nil {
		return m.latestBlockFunc(ctx)
	}
	return nil, nil
}

func (m *mockBlockReader) ChainID(ctx context.Context) (uint64, error) {
	if m.chainIDFunc != nil {
		return m.chainIDFunc(ctx)
	}
	return 0, nil
}

type mockTxReader struct {
	txByHashFunc func(ctx context.Context, hash string) (*eth.Transaction, error)
}

func (m *mockTxReader) TransactionByHash(ctx context.Context, hash string) (*eth.Transaction, error) {
	if m.txByHashFunc != nil {
		return m.txByHashFunc(ctx, hash)
	}
	return nil, nil
}

type mockSubscriber struct {
	subHeadsFunc   func(ctx context.Context) (<-chan *eth.Block, error)
	subPendingFunc func(ctx context.Context) (<-chan string, error)
	closeFunc      func() error
}

func (m *mockSubscriber) SubscribeNewHeads(ctx context.Context) (<-chan *eth.Block, error) {
	if m.subHeadsFunc != nil {
		return m.subHeadsFunc(ctx)
	}
	return nil, nil
}

func (m *mockSubscriber) SubscribeNewPendingTransactions(ctx context.Context) (<-chan string, error) {
	if m.subPendingFunc != nil {
		return m.subPendingFunc(ctx)
	}
	return nil, nil
}

func (m *mockSubscriber) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}
