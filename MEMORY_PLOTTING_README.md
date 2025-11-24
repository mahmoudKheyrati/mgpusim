# Memory Utilization Plotting for MGPUSim

This guide explains how to visualize memory utilization metrics from MGPUSim simulations.

## Overview

The `plot_memory_utilization.py` script reads metrics from MGPUSim's SQLite database and generates visualizations of:
- DRAM read/write sizes
- DRAM transaction counts
- DRAM average latencies
- Memory bandwidth utilization

## Quick Start

### 1. Install Dependencies

```bash
pip install -r requirements.txt
```

Or manually install:
```bash
pip install matplotlib pandas numpy
```

### 2. Generate a Database

Run any MGPUSim benchmark with the `-timing` and `-report-all` flags:

```bash
cd amd/samples/matrixmultiplication
go build
./matrixmultiplication -timing -report-all
```

This will create a database file named `akita_sim_<random_id>.sqlite3` in the current directory.

### 3. Plot the Data

```bash
python plot_memory_utilization.py akita_sim_*.sqlite3
```

## Output

The script generates four types of plots:

1. **Memory Sizes** - Total bytes read/written per DRAM component
2. **Transaction Counts** - Number of read/write transactions per DRAM
3. **Latencies** - Average read/write latency per DRAM component
4. **Bandwidth** - Effective memory bandwidth (GB/s) per DRAM

Each plot is saved as a PNG file and displayed interactively.

## Database Schema

The MGPUSim metrics database contains a table called `mgpusim_metrics` with the following structure:

```
Location (TEXT)  - Component name (e.g., "DRAM[0]", "CU[1]")
What (TEXT)      - Metric type (e.g., "read_size", "write_trans_count")
Value (REAL)     - Metric value
Unit (TEXT)      - Unit of measurement (e.g., "bytes", "second")
```

### DRAM-Related Metrics

- `read_trans_count` - Total read transactions (unit: count)
- `write_trans_count` - Total write transactions (unit: count)
- `read_avg_latency` - Average read latency (unit: second)
- `write_avg_latency` - Average write latency (unit: second)
- `read_size` - Total bytes read (unit: bytes)
- `write_size` - Total bytes written (unit: bytes)

## Important Note: Aggregated vs Time-Series Data

**The current MGPUSim database stores aggregated metrics, not time-series data.**

The metrics in the `mgpusim_metrics` table represent cumulative values collected over the entire simulation. For example:
- `read_size` is the **total** bytes read during the entire kernel execution
- `read_trans_count` is the **total** number of transactions

### If You Need Time-Series Data

To get time-stamped event data over time, you need to use the visualization tracer:

1. **Enable visualization tracing:**
   ```bash
   ./benchmark -timing -trace-vis -trace-vis-db sqlite
   ```

2. **The visualization tracer creates a separate database** with event-level data including timestamps. This database will contain task start/end times for individual memory transactions.

3. **To plot time-series memory utilization**, you would need to:
   - Query the visualization trace database for memory transaction events
   - Extract timestamps and memory access sizes
   - Aggregate data into time windows
   - Plot memory usage over time

If you need time-series plotting functionality, the script would need to be extended to read from the visualization trace database (which has a different schema managed by the Akita framework).

## Example Usage

```bash
# Generate database from a benchmark
cd amd/samples/fir
go build
./fir -timing -report-all -length=65536

# Plot the results
cd ../../..
python plot_memory_utilization.py amd/samples/fir/akita_sim_*.sqlite3

# The script will output:
# - akita_sim_<id>_memory_sizes.png
# - akita_sim_<id>_transaction_counts.png
# - akita_sim_<id>_latencies.png
# - akita_sim_<id>_bandwidth.png
```

## Customization

You can modify the script to:
- Add additional metrics (cache hits, TLB stats, etc.)
- Change plot styles and colors
- Filter specific DRAM components
- Export data to CSV for further analysis

## Available Flags for Simulation

When running simulations, use these flags to control what metrics are collected:

- `-timing` - Enable timing simulation (required for metrics)
- `-report-all` - Report all available metrics to the database
- `-report-inst-count` - Report instruction counts
- `-report-cache-latency` - Report cache latency metrics
- `-report-cache-hit-rate` - Report cache hit rates
- `-report-tlb-hit-rate` - Report TLB hit rates
- `-report-rdma-transaction-count` - Report RDMA transaction counts
- `-report-dram-transaction-count` - Report DRAM transaction counts

Use `-report-all` to capture all metrics including memory utilization data.
