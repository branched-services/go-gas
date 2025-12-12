# Library Usage Example

This example demonstrates how to embed the `go-gas` library directly into your Go application. This is useful for building trading bots, backend services, or any application that needs low-latency gas estimates without running a separate microservice.

## Prerequisites

- Go 1.22 or higher
- An Ethereum Node URL (HTTP and WebSocket)
  - You can use services like Infura, Alchemy, or a local node (e.g., Geth, Reth).

## Running the Example

1.  **Set Environment Variables**

    You need to provide the connection details for your Ethereum node.

    ```bash
    export GAS_NODE_HTTP_URL="https://mainnet.infura.io/v3/YOUR_API_KEY"
    export GAS_NODE_WS_URL="wss://mainnet.infura.io/ws/v3/YOUR_API_KEY"
    ```

    *Note: Replace `YOUR_API_KEY` with your actual provider key.*

2.  **Run the Code**

    Execute the `main.go` file:

    ```bash
    go run main.go
    ```

## What to Expect

The program will:
1.  Connect to the Ethereum node.
2.  Start the `Estimator` in the background.
3.  Wait for the first block and mempool data to be processed.
4.  Print gas estimates to the console every 2 seconds.

**Sample Output:**

```text
Waiting for first estimate...

--- Gas Estimate (Block 19283746) ---
Base Fee: 15000000000 wei
Urgent (99%): 2000000000 wei (Total: 32000000000)
Fast   (90%): 1500000000 wei
Std    (50%): 1000000000 wei
Slow   (25%): 500000000 wei
```

## Key Concepts Demonstrated

- **Dependency Injection**: How to initialize `eth.Client`, `eth.WSSubscriber`, and `estimator.Provider`.
- **Configuration**: Using `estimator.Option` functions to tune performance (e.g., `WithHistorySize`, `WithRecalcInterval`).
- **Lifecycle Management**: Using `context.Context` for graceful shutdown.
- **Thread-Safe Access**: Reading estimates from `provider.Current()` safely from a separate goroutine.
