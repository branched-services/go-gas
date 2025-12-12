# Gas Estimator Service

High-performance, ultra low-latency Ethereum gas price estimator for Go.

`go-gas` is designed for high-frequency trading, MEV bots, and real-time applications where millisecond-latency matters. It uses a push-based architecture and highly optimized arithmetic (via `uint256`) to deliver gas estimates with minimal heap allocations.

[![Go Reference](https://pkg.go.dev/badge/github.com/branched-services/go-gas.svg)](https://pkg.go.dev/github.com/branched-services/go-gas)
[![Go Report Card](https://goreportcard.com/badge/github.com/branched-services/go-gas)](https://goreportcard.com/report/github.com/branched-services/go-gas)

## Features

- **Ultra Low Latency**: ~69µs calculation time per update.
- **Zero-Copy Math**: Built on `github.com/holiman/uint256` to avoid `math/big` GC pressure.
- **Push-Based**: Subscribes to WebSocket block headers and pending transactions.
- **Hybrid Strategy**: Combines historical block analysis (EIP-1559) with real-time mempool sampling.
- **Thread Safe**: Lock-free reads via atomic pointer swapping.
- **Flexible**: Run as a standalone gRPC/HTTP service or import as a Go library.

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

## Benchmarks

Performance is the primary goal of this library. We migrated from `math/big` to `uint256` to reduce garbage collection overhead on the hot path.

| Metric | Value | Notes |
| :--- | :--- | :--- |
| **Latency** | **~69 µs/op** | Core calculation loop |
| **Allocations** | **~1,033 allocs/op** | Reduced from >2,000 allocs/op |
| **Memory** | **~49 KB/op** | Mostly stack-allocated |

*Benchmarks run on Intel i9-9900K @ 3.60GHz*

To run benchmarks yourself:
```bash
go test -bench=. -benchmem ./pkg/estimator/...
```

## Installation

```bash
go get github.com/branched-services/go-gas
```

## Usage

### As a Library

Embed the estimator directly into your Go application for the lowest possible latency (no network hop).

```go
package main

import (
"context"
"fmt"
"log"
"time"

"github.com/branched-services/go-gas/pkg/estimator"
"github.com/branched-services/go-gas/pkg/eth"
)

func main() {
ctx := context.Background()

// 1. Initialize Ethereum Client
// Requires a node with WebSocket and Debug/TxPool API support
client, err := eth.NewClient(
"http://localhost:8545",
"ws://localhost:8546",
)
if err != nil {
log.Fatal(err)
}

// 2. Create the Provider (holds the current estimate)
provider := estimator.NewProvider()

// 3. Create and Configure Estimator
est := estimator.New(
client,   // BlockReader
client,   // TransactionReader
client,   // Subscriber
provider, // State container
estimator.WithHistorySize(20),
estimator.WithMempoolSamples(500),
estimator.WithRecalcInterval(100*time.Millisecond),
)

// 4. Start the Estimator in the background
go func() {
if err := est.Run(ctx); err != nil {
log.Printf("Estimator stopped: %v", err)
}
}()

// 5. Read Estimates (Thread-Safe, Non-Blocking)
ticker := time.NewTicker(1 * time.Second)
for range ticker.C {
estimate, err := provider.Current(ctx)
if err != nil {
log.Println("Estimator warming up...")
continue
}

fmt.Printf("BaseFee: %s Gwei\n", estimate.BaseFee.Dec())
fmt.Printf("Fast (90%%): %s Gwei (Priority: %s)\n",
estimate.Fast.MaxFeePerGas.Dec(),
estimate.Fast.MaxPriorityFeePerGas.Dec(),
)
}
}
```

### As a Standalone Service

You can also run `go-gas` as a standalone microservice that exposes estimates via gRPC or HTTP.

#### 1. Configuration
Configure via environment variables:

| Variable | Description | Default |
| :--- | :--- | :--- |
| `GAS_NODE_HTTP_URL` | Ethereum Node HTTP URL | `http://localhost:8545` |
| `GAS_NODE_WS_URL` | Ethereum Node WebSocket URL | `ws://localhost:8546` |
| `GAS_PORT` | Service Port | `8080` |
| `GAS_LOG_LEVEL` | Log Level (debug, info, warn, error) | `info` |

#### 2. Run with Docker

```bash
docker run -d \
  -e GAS_NODE_HTTP_URL=https://eth-mainnet.alchemyapi.io/v2/YOUR_KEY \
  -e GAS_NODE_WS_URL=wss://eth-mainnet.alchemyapi.io/v2/YOUR_KEY \
  -p 8080:8080 \
  ghcr.io/branched-services/go-gas:latest
```

#### 3. Run from Source

```bash
# Clone and build
git clone https://github.com/branched-services/go-gas.git
cd go-gas
go build -o gas-estimator cmd/estimator/main.go

# Run
export GAS_NODE_HTTP_URL=...
./gas-estimator
```

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to update tests and run benchmarks as appropriate.

## License

[MIT](https://choosealicense.com/licenses/mit/)
