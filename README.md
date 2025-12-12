# Gas Estimator Service

Ultra low-latency gas price estimation service for EIP-1559 chains.

## Features

- **Sub-millisecond reads**: Pre-computed estimates served via atomic load
- **Hybrid estimation**: Combines historical block data with mempool analysis
- **Real-time updates**: WebSocket subscription to new blocks
- **12-Factor compliant**: ENV config, stdout logs, graceful shutdown
- **Production ready**: Health probes, structured logging, clean shutdown

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Gas Estimator Service                     │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   ┌──────────┐     ┌───────────────┐     ┌──────────────┐   │
│   │  Node    │────▶│   Estimator   │────▶│   Provider   │   │
│   │ (WS+RPC) │     │ (Orchestrator)│     │ (atomic.Ptr) │   │
│   └──────────┘     └───────────────┘     └──────┬───────┘   │
│                           │                      │           │
│                    ┌──────┴──────┐               │           │
│                    │             │               ▼           │
│               ┌────┴────┐  ┌─────┴─────┐  ┌───────────┐     │
│               │ History │  │ Calculator│  │ API Server│     │
│               │ (Ring)  │  │ (Strategy)│  │  (HTTP)   │     │
│               └─────────┘  └───────────┘  └───────────┘     │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Quick Start

### Prerequisites

- Go 1.22+
- Access to an Ethereum node with:
  - HTTP RPC endpoint
  - WebSocket endpoint
  - `txpool_content` or `eth_pendingTransactions` support

### Run Locally

```bash
# Set required environment variables
export GAS_NODE_HTTP_URL=http://localhost:8545
export GAS_NODE_WS_URL=ws://localhost:8546

# Optional: customize settings
export GAS_GRPC_ADDR=:9090
export GAS_HTTP_ADDR=:8080
export GAS_LOG_LEVEL=debug
export GAS_LOG_FORMAT=text

# Run
go run ./cmd/estimator
```

### Docker

```bash
docker build -t gas-estimator .

docker run -d \
  -p 9090:9090 \
  -p 8080:8080 \
  -e GAS_NODE_HTTP_URL=http://host.docker.internal:8545 \
  -e GAS_NODE_WS_URL=ws://host.docker.internal:8546 \
  gas-estimator
```

## Configuration

All configuration is via environment variables (12-Factor: Config).

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `GAS_NODE_HTTP_URL` | Yes | - | Ethereum node HTTP RPC URL |
| `GAS_NODE_WS_URL` | Yes | - | Ethereum node WebSocket URL |
| `GAS_GRPC_ADDR` | No | `:9090` | API server listen address |
| `GAS_HTTP_ADDR` | No | `:8080` | Health server listen address |
| `GAS_HISTORY_BLOCKS` | No | `20` | Number of historical blocks to track |
| `GAS_MEMPOOL_SAMPLES` | No | `500` | Max pending transactions to sample |
| `GAS_RECALC_INTERVAL` | No | `200ms` | Estimate recalculation interval |
| `GAS_LOG_LEVEL` | No | `info` | Log level (debug, info, warn, error) |
| `GAS_LOG_FORMAT` | No | `json` | Log format (json, text) |

## API

### GET /v1/gas/estimate

Returns current gas price estimates.

**Response:**

```json
{
  "chain_id": 1,
  "block_number": 18500000,
  "timestamp": "2024-01-15T10:30:00.123456789Z",
  "base_fee": "20000000000",
  "estimates": {
    "urgent": {
      "max_priority_fee_per_gas": "2000000000",
      "max_fee_per_gas": "42000000000",
      "confidence": 0.99
    },
    "fast": {
      "max_priority_fee_per_gas": "1500000000",
      "max_fee_per_gas": "41500000000",
      "confidence": 0.90
    },
    "standard": {
      "max_priority_fee_per_gas": "1000000000",
      "max_fee_per_gas": "41000000000",
      "confidence": 0.50
    },
    "slow": {
      "max_priority_fee_per_gas": "500000000",
      "max_fee_per_gas": "40500000000",
      "confidence": 0.25
    }
  }
}
```

### GET /v1/gas/estimate/stream

Server-Sent Events stream of estimate updates.

```bash
curl -N http://localhost:9090/v1/gas/estimate/stream
```

### Health Endpoints

- `GET /healthz` - Liveness probe (always 200 if process is alive)
- `GET /readyz` - Readiness probe (200 when estimates are available)

## Design Decisions

### Why atomic.Pointer for estimates?

Reads vastly outnumber writes (~1000:1). `atomic.Pointer` provides:
- Zero-allocation reads
- No lock contention
- Single CPU instruction on read path

### Why RWMutex for history buffer?

Write frequency is ~1 per 12 seconds (new block). The simplicity of `RWMutex` outweighs any theoretical benefit of lock-free structures for this access pattern.

### Why hybrid estimation?

Pure historical data lags market conditions. Pure mempool data is noisy and gameable. The hybrid approach:
1. Uses mempool for leading signal (70% weight by default)
2. Anchors to historical reality (30% weight)
3. Applies exponential smoothing to reduce volatility

### Why 2x base fee buffer?

EIP-1559 base fee can increase up to 12.5% per block. A 2x buffer handles ~6 consecutive full blocks, covering most congestion scenarios without overpaying significantly.

## Project Structure

```
gas-estimator/
├── cmd/estimator/           # Application entry point
├── internal/
│   ├── config/              # ENV-based configuration
│   ├── eth/                 # Ethereum client & types
│   ├── estimator/           # Core estimation logic
│   │   ├── calculator.go    # Pure estimation functions
│   │   ├── estimator.go     # Orchestrator
│   │   ├── history.go       # Ring buffer for blocks
│   │   ├── provider.go      # Atomic estimate storage
│   │   ├── strategy.go      # Strategy interface
│   │   └── types.go         # Domain types
│   ├── api/grpc/            # HTTP/gRPC API server
│   ├── health/              # Health check endpoints
│   └── observability/       # Logging setup
├── Dockerfile
└── README.md
```

## Future Improvements

- [ ] Proper gRPC with protobuf (current impl is HTTP/JSON)
- [ ] Prometheus metrics export
- [ ] Multi-chain support
- [ ] Configurable estimation strategies
- [ ] Time-to-inclusion predictions
- [ ] Historical accuracy tracking

## License

MIT
