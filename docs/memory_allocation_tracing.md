# Memory Allocation Time-Series Tracing

This feature allows you to track memory allocation over time during simulation and visualize it with Python.

## Overview

MGPUSim now supports tracking how much memory is allocated on each device (CPU and GPUs) at different points in time during simulation. This helps analyze:

- Memory usage patterns over time
- Peak memory allocation
- Memory utilization percentage
- Memory allocation trends per device

## Usage

### Step 1: Run Simulation with Memory Allocation Tracing

Enable memory allocation tracing when running your benchmark:

```bash
# Basic usage
./your_benchmark --timing --trace-mem-alloc

# Custom sampling period (default: 0.001 seconds simulation time)
./your_benchmark --timing --trace-mem-alloc --trace-mem-alloc-period 0.01

# Custom output file name (default: memory_allocation.csv)
./your_benchmark --timing --trace-mem-alloc --trace-mem-alloc-file my_trace.csv
```

### Step 2: Visualize with Python

After the simulation completes, use the provided Python script to plot the results:

```bash
# Plot from default file
python amd/samples/runner/plot_memory_allocation.py

# Plot from custom file
python amd/samples/runner/plot_memory_allocation.py my_trace.csv

# Save plot to file
python amd/samples/runner/plot_memory_allocation.py --output memory_plot.png

# Save without displaying interactively
python amd/samples/runner/plot_memory_allocation.py --output memory_plot.png --no-show
```

### Python Requirements

The plotting script requires:
- Python 3.x
- pandas
- matplotlib
- numpy

Install dependencies:
```bash
pip install pandas matplotlib numpy
```

## Output Format

The tracer outputs a CSV file with the following columns:

| Column | Description |
|--------|-------------|
| Time(s) | Simulation time in seconds |
| DeviceID | Device ID number |
| DeviceName | Device name (CPU, GPU1, GPU2, etc.) |
| AllocatedBytes | Currently allocated memory in bytes |
| TotalBytes | Total available memory in bytes |
| AllocatedMB | Currently allocated memory in MB |
| TotalMB | Total available memory in MB |
| UtilizationPercent | Memory utilization percentage (0-100%) |

Example:
```csv
Time(s),DeviceID,DeviceName,AllocatedBytes,TotalBytes,AllocatedMB,TotalMB,UtilizationPercent
0.001000000,0,CPU,0,4294967296,0.00,4096.00,0.00
0.001000000,1,GPU1,524288000,4294967296,500.00,4096.00,12.21
0.002000000,1,GPU1,1048576000,4294967296,1000.00,4096.00,24.41
```

## Plots Generated

The Python script generates two plots:

1. **Memory Allocation Over Time**:
   - Shows allocated memory (solid lines) vs. total capacity (dashed lines)
   - Displays in MB or GB depending on memory size
   - One line per device

2. **Memory Utilization Percentage Over Time**:
   - Shows utilization percentage (0-100%)
   - One line per device

## Summary Statistics

The script also prints summary statistics for each device:
- Total Memory
- Peak Allocation
- Average Allocation
- Peak Utilization
- Average Utilization

Example output:
```
=== Memory Allocation Summary ===

GPU1:
  Total Memory: 4096.00 MB (4.00 GB)
  Peak Allocation: 3200.00 MB (3.12 GB)
  Average Allocation: 2400.00 MB (2.34 GB)
  Peak Utilization: 78.13%
  Average Utilization: 58.59%
```

## Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--trace-mem-alloc` | false | Enable memory allocation tracing |
| `--trace-mem-alloc-period` | 0.001 | Sampling period in simulation seconds |
| `--trace-mem-alloc-file` | memory_allocation.csv | Output CSV file name |
| `--report-all` | false | Enables all reporting including memory allocation tracing |

## How It Works

1. **Background Sampling**: A goroutine samples memory allocation state every 100ms of real time
2. **Device Query**: For each device, queries the memory allocator to get:
   - Total storage size
   - Free memory size
   - Allocated = Total - Free
3. **CSV Output**: Writes samples to CSV file in real-time
4. **Plotting**: Python script reads CSV and creates visualizations

## Notes

- Sampling period is in **simulation time**, not real time
- Real-time sampling interval is fixed at 100ms for performance
- Works with both default and buddy memory allocators
- Tracks all devices: CPU and all GPUs
- CSV file is written incrementally during simulation
- No significant performance overhead

## Example Workflow

```bash
# 1. Run benchmark with tracing
cd amd/samples/matrixmultiplication
go build
./matrixmultiplication --timing --trace-mem-alloc

# 2. Plot results
python ../../runner/plot_memory_allocation.py memory_allocation.csv --output mm_memory.png

# 3. View the plot (mm_memory.png) or the CSV data
```

## Integration with Existing Metrics

Memory allocation tracing complements the existing `--report-mem-util` flag:

- `--report-mem-util`: Reports memory **bandwidth** and **traffic** (requests/sec, bytes/sec)
- `--trace-mem-alloc`: Reports memory **capacity** allocation over time (how much is allocated)

Both can be used together:
```bash
./benchmark --timing --report-all --trace-mem-alloc
```

This will generate:
- `metrics.csv` - Contains bandwidth, latency, and other metrics
- `memory_allocation.csv` - Contains time-series allocation data
