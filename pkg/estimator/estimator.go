package estimator

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/branched-services/go-gas/pkg/eth"
	"github.com/holiman/uint256"
)

// Estimator orchestrates gas estimation by:
// 1. Subscribing to new blocks
// 2. Sampling the mempool
// 3. Triggering recalculation
// 4. Updating the provider
type Estimator struct {
	// Dependencies (injected)
	client     eth.BlockReader
	txReader   eth.TransactionReader
	subscriber eth.Subscriber
	provider   *Provider
	strategy   Strategy
	logger     *slog.Logger

	// Configuration
	historySize    int
	mempoolSamples int
	recalcInterval time.Duration

	// Internal state
	history   *History
	localPool *LocalTxPool
	chainID   uint64

	// Lifecycle
	mu      sync.Mutex
	running bool
}

// Option configures an Estimator.
type Option func(*Estimator)

// WithHistorySize sets the number of historical blocks to track.
func WithHistorySize(size int) Option {
	return func(e *Estimator) {
		e.historySize = size
	}
}

// WithMempoolSamples sets the maximum number of pending transactions to sample.
func WithMempoolSamples(samples int) Option {
	return func(e *Estimator) {
		e.mempoolSamples = samples
	}
}

// WithRecalcInterval sets how often to recalculate estimates.
func WithRecalcInterval(d time.Duration) Option {
	return func(e *Estimator) {
		e.recalcInterval = d
	}
}

// WithStrategy sets the estimation strategy.
func WithStrategy(s Strategy) Option {
	return func(e *Estimator) {
		e.strategy = s
	}
}

// WithLogger sets the logger.
func WithLogger(l *slog.Logger) Option {
	return func(e *Estimator) {
		e.logger = l
	}
}

// New creates a new Estimator with the given dependencies and options.
func New(
	client eth.BlockReader,
	txReader eth.TransactionReader,
	subscriber eth.Subscriber,
	provider *Provider,
	opts ...Option,
) *Estimator {
	e := &Estimator{
		client:         client,
		txReader:       txReader,
		subscriber:     subscriber,
		provider:       provider,
		strategy:       DefaultStrategy(),
		logger:         slog.Default(),
		historySize:    20,
		mempoolSamples: 500,
		recalcInterval: 200 * time.Millisecond,
	}

	for _, opt := range opts {
		opt(e)
	}

	e.history = NewHistory(e.historySize)
	e.localPool = NewLocalTxPool(e.mempoolSamples * 2)
	e.logger = e.logger.With("component", "estimator")

	return e
}

// Run starts the estimator. Blocks until context is canceled.
func (e *Estimator) Run(ctx context.Context) error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return fmt.Errorf("estimator already running")
	}
	e.running = true
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		e.running = false
		e.mu.Unlock()
	}()

	// Get chain ID
	chainID, err := e.client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("getting chain ID: %w", err)
	}
	e.chainID = chainID
	e.logger.Info("connected to chain", "chain_id", chainID)

	// Bootstrap with recent blocks
	if err := e.bootstrap(ctx); err != nil {
		return fmt.Errorf("bootstrapping: %w", err)
	}

	// Subscribe to new blocks
	blockCh, err := e.subscriber.SubscribeNewHeads(ctx)
	if err != nil {
		return fmt.Errorf("subscribing to new heads: %w", err)
	}

	// Subscribe to pending transactions
	txHashCh, err := e.subscriber.SubscribeNewPendingTransactions(ctx)
	if err != nil {
		return fmt.Errorf("subscribing to pending txs: %w", err)
	}

	// Periodic recalculation ticker
	ticker := time.NewTicker(e.recalcInterval)
	defer ticker.Stop()

	e.logger.Info("estimator running",
		"strategy", e.strategy.Name(),
		"history_size", e.historySize,
		"mempool_samples", e.mempoolSamples,
		"recalc_interval", e.recalcInterval,
	)

	for {
		select {
		case <-ctx.Done():
			e.logger.Info("estimator stopping")
			return nil

		case block, ok := <-blockCh:
			if !ok {
				return fmt.Errorf("block subscription closed")
			}
			e.handleNewBlock(ctx, block)

		case txHash, ok := <-txHashCh:
			if !ok {
				return fmt.Errorf("tx subscription closed")
			}
			// Process in background to avoid blocking main loop
			go e.processPendingTx(ctx, txHash)

		case <-ticker.C:
			e.recalculate(ctx)
		}
	}
}

// bootstrap loads recent blocks to warm up the history.
func (e *Estimator) bootstrap(ctx context.Context) error {
	latest, err := e.client.LatestBlock(ctx)
	if err != nil {
		return fmt.Errorf("getting latest block: %w", err)
	}

	e.logger.Info("bootstrapping history", "latest_block", latest.Number)

	// Load last N blocks
	for i := 0; i < e.historySize && latest.Number > uint64(i); i++ {
		blockNum := latest.Number - uint64(i)
		block, err := e.client.BlockByNumber(ctx, uint256.NewInt(blockNum))
		if err != nil {
			e.logger.Warn("failed to fetch historical block",
				"block", blockNum,
				"error", err,
			)
			continue
		}
		e.history.Push(block)
	}

	e.logger.Info("bootstrap complete", "blocks_loaded", e.history.Len())

	// Trigger initial calculation
	e.recalculate(ctx)

	return nil
}

// handleNewBlock processes a new block notification.
func (e *Estimator) handleNewBlock(ctx context.Context, block *eth.Block) {
	start := time.Now()

	// Fetch full block with transactions
	fullBlock, err := e.client.BlockByNumber(ctx, uint256.NewInt(block.Number))
	if err != nil {
		e.logger.Error("failed to fetch full block",
			"block", block.Number,
			"error", err,
		)
		return
	}

	e.history.Push(fullBlock)
	e.recalculate(ctx)

	lag := time.Since(block.Timestamp)
	e.logger.Info("processed new block",
		"block", block.Number,
		"base_fee_gwei", weiToGwei(block.BaseFee),
		"chain_lag_ms", lag.Milliseconds(),
		"processing_time_ms", time.Since(start).Milliseconds(),
	)
}

// recalculate computes a new estimate and updates the provider.
func (e *Estimator) recalculate(ctx context.Context) {
	start := time.Now()

	// Build calculator input
	input, err := e.buildInput(ctx)
	if err != nil {
		e.logger.Error("failed to build calculator input", "error", err)
		return
	}

	// Calculate new estimate
	estimate, err := e.strategy.Calculate(ctx, input)
	if err != nil {
		e.logger.Error("calculation failed", "error", err)
		return
	}

	// Update provider
	e.provider.Update(estimate)

	e.logger.Debug("estimate updated",
		"block", estimate.BlockNumber,
		"base_fee_gwei", weiToGwei(estimate.BaseFee),
		"urgent_priority_gwei", weiToGwei(estimate.Urgent.MaxPriorityFeePerGas),
		"standard_priority_gwei", weiToGwei(estimate.Standard.MaxPriorityFeePerGas),
		"duration_us", time.Since(start).Microseconds(),
	)
}

// buildInput constructs the calculator input from current state.
func (e *Estimator) buildInput(ctx context.Context) (*CalculatorInput, error) {
	blocks := e.history.Snapshot()
	if len(blocks) == 0 {
		return nil, fmt.Errorf("no blocks in history")
	}

	// Convert blocks to BlockData
	recentBlocks := make([]*BlockData, len(blocks))
	for i, block := range blocks {
		recentBlocks[i] = e.convertBlock(block)
	}

	// Sample pending transactions from local pool
	pendingTxs := e.localPool.Snapshot()

	// Get previous estimate for smoothing
	var prevEstimate *GasEstimate
	if est, err := e.provider.Current(ctx); err == nil {
		prevEstimate = est
	}

	return &CalculatorInput{
		ChainID:          e.chainID,
		CurrentBlock:     recentBlocks[0],
		RecentBlocks:     recentBlocks,
		PendingTxs:       pendingTxs,
		PreviousEstimate: prevEstimate,
	}, nil
}

func (e *Estimator) convertBlock(block *eth.Block) *BlockData {
	bd := &BlockData{
		Number:    block.Number,
		Timestamp: block.Timestamp,
		BaseFee:   block.BaseFee,
		GasUsed:   block.GasUsed,
		GasLimit:  block.GasLimit,
	}

	// Extract priority fees from transactions
	for _, tx := range block.Transactions {
		fee := tx.EffectivePriorityFee(block.BaseFee)
		if !fee.IsZero() {
			bd.PriorityFees = append(bd.PriorityFees, fee)
		}
	}

	return bd
}

func (e *Estimator) convertTx(tx *eth.Transaction) *TxData {
	return &TxData{
		MaxPriorityFeePerGas: tx.MaxPriorityFeePerGas,
		MaxFeePerGas:         tx.MaxFeePerGas,
		GasPrice:             tx.GasPrice,
		IsEIP1559:            tx.IsEIP1559(),
	}
}

// processPendingTx fetches and adds a pending transaction to the local pool.
func (e *Estimator) processPendingTx(ctx context.Context, hash string) {
	// Fetch transaction details
	// Use a short timeout to avoid hanging on slow nodes
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	tx, err := e.txReader.TransactionByHash(ctx, hash)
	if err != nil {
		// Ignore errors (tx might be gone or node busy)
		return
	}
	e.localPool.Add(tx)
}

// Helper functions

func weiToGwei(wei *uint256.Int) float64 {
	if wei == nil {
		return 0
	}
	gwei := new(uint256.Int).Div(wei, uint256.NewInt(1e9))
	return float64(gwei.Uint64())
}
