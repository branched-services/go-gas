# Benchmark History

Tracking performance improvements over time.

## Run Log

| Date       | Version / Change                                                                     | Metric                 | Value            | Change vs Baseline |
| :--------- | :----------------------------------------------------------------------------------- | :--------------------- | :--------------- | :----------------- |
| 2025-12-12 | **Baseline** (Initial)                                                               | `LocalTxPool_Add`      | 69.48 ns/op      | -                  |
|            |                                                                                      | `LocalTxPool_Snapshot` | 53,846 ns/op     | -                  |
|            |                                                                                      | `Strategy_Calculate`   | 71,172 ns/op     | -                  |
| 2025-12-12 | **Optimization 1**<br>• `goccy/go-json`<br>• `slices.SortFunc`<br>• Pre-calc History | `LocalTxPool_Add`      | **65.50 ns/op**  | **-5.7%**          |
|            |                                                                                      | `LocalTxPool_Snapshot` | **49,295 ns/op** | **-8.5%**          |
|            |                                                                                      | `Strategy_Calculate`   | **61,300 ns/op** | **-13.9%**         |

## Detailed Analysis

### Optimization 1 (Current)

**Changes Implemented:**
1.  **Sorting Optimization**: Replaced reflection-based `sort.Slice` with generic `slices.SortFunc` in the hot calculation path. This accounts for the majority of the ~14% speedup in `Strategy_Calculate`.
2.  **JSON Parsing**: Switched to `github.com/goccy/go-json` for faster RPC payload handling (not captured in micro-benchmarks but improves end-to-end latency).
3.  **Pre-calculated History**: Storing `*BlockData` instead of raw `*eth.Block` in the history ring buffer reduces overhead during the calculation phase.

**Impact:**
- Core calculation latency dropped from **~71µs** to **~61µs**.
- Snapshotting the pool is faster, likely due to reduced GC pressure or better cache locality from other changes.
