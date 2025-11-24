#!/usr/bin/env python3
"""
Plot memory utilization metrics from MGPUSim SQLite database.

This script plots both OLD and NEW memory metrics:
- OLD metrics (dramTracer): Basic DRAM stats like sizes, transaction counts, latencies
- NEW metrics (memUtilTracer): Advanced metrics like bandwidth, outstanding requests

Usage:
    python plot_memory_utilization.py <database_file.sqlite3>

Examples:
    # Generate database with all metrics
    ./benchmark -timing -report-all

    # Or specifically enable memory utilization metrics
    ./benchmark -timing -report-mem-util

    # Plot the results
    python plot_memory_utilization.py akita_sim_12345.sqlite3

The script generates up to 7 plots:
1. DRAM memory sizes (read/write bytes)
2. DRAM transaction counts
3. DRAM latencies
4. DRAM bandwidth (calculated from sizes)
5. Memory bandwidth (from memUtilTracer - NEW)
6. Outstanding requests (average/max - NEW)
7. Request counts (from memUtilTracer - NEW)
"""

import sqlite3
import sys
import matplotlib.pyplot as plt
import pandas as pd
import numpy as np
from pathlib import Path


def load_metrics(db_path):
    """Load metrics from the SQLite database."""
    if not Path(db_path).exists():
        raise FileNotFoundError(f"Database file not found: {db_path}")

    conn = sqlite3.connect(db_path)

    # Read the mgpusim_metrics table
    query = "SELECT Location, What, Value, Unit FROM mgpusim_metrics"
    df = pd.read_sql_query(query, conn)
    conn.close()

    return df


def plot_dram_memory_usage(df):
    """Plot DRAM memory read/write sizes."""
    # Filter DRAM-related metrics
    dram_df = df[df['Location'].str.contains('DRAM', case=False, na=False)]

    # Get read and write sizes
    read_size_df = dram_df[dram_df['What'] == 'read_size']
    write_size_df = dram_df[dram_df['What'] == 'write_size']

    if read_size_df.empty and write_size_df.empty:
        print("No DRAM memory size data found in database.")
        return None

    fig, ax = plt.subplots(figsize=(12, 6))

    locations = sorted(set(read_size_df['Location'].tolist() + write_size_df['Location'].tolist()))
    x = np.arange(len(locations))
    width = 0.35

    read_values = []
    write_values = []

    for loc in locations:
        read_val = read_size_df[read_size_df['Location'] == loc]['Value'].values
        write_val = write_size_df[write_size_df['Location'] == loc]['Value'].values

        read_values.append(read_val[0] if len(read_val) > 0 else 0)
        write_values.append(write_val[0] if len(write_val) > 0 else 0)

    # Convert bytes to MB for better readability
    read_values_mb = [v / (1024 * 1024) for v in read_values]
    write_values_mb = [v / (1024 * 1024) for v in write_values]

    ax.bar(x - width/2, read_values_mb, width, label='Read', color='skyblue')
    ax.bar(x + width/2, write_values_mb, width, label='Write', color='lightcoral')

    ax.set_xlabel('DRAM Component')
    ax.set_ylabel('Data Size (MB)')
    ax.set_title('DRAM Memory Read/Write Sizes')
    ax.set_xticks(x)
    ax.set_xticklabels(locations, rotation=45, ha='right')
    ax.legend()
    ax.grid(axis='y', alpha=0.3)

    plt.tight_layout()
    return fig


def plot_dram_transaction_counts(df):
    """Plot DRAM transaction counts."""
    dram_df = df[df['Location'].str.contains('DRAM', case=False, na=False)]

    read_trans_df = dram_df[dram_df['What'] == 'read_trans_count']
    write_trans_df = dram_df[dram_df['What'] == 'write_trans_count']

    if read_trans_df.empty and write_trans_df.empty:
        print("No DRAM transaction count data found in database.")
        return None

    fig, ax = plt.subplots(figsize=(12, 6))

    locations = sorted(set(read_trans_df['Location'].tolist() + write_trans_df['Location'].tolist()))
    x = np.arange(len(locations))
    width = 0.35

    read_counts = []
    write_counts = []

    for loc in locations:
        read_val = read_trans_df[read_trans_df['Location'] == loc]['Value'].values
        write_val = write_trans_df[write_trans_df['Location'] == loc]['Value'].values

        read_counts.append(read_val[0] if len(read_val) > 0 else 0)
        write_counts.append(write_val[0] if len(write_val) > 0 else 0)

    ax.bar(x - width/2, read_counts, width, label='Read Transactions', color='steelblue')
    ax.bar(x + width/2, write_counts, width, label='Write Transactions', color='indianred')

    ax.set_xlabel('DRAM Component')
    ax.set_ylabel('Transaction Count')
    ax.set_title('DRAM Read/Write Transaction Counts')
    ax.set_xticks(x)
    ax.set_xticklabels(locations, rotation=45, ha='right')
    ax.legend()
    ax.grid(axis='y', alpha=0.3)

    plt.tight_layout()
    return fig


def plot_dram_latencies(df):
    """Plot DRAM average latencies."""
    dram_df = df[df['Location'].str.contains('DRAM', case=False, na=False)]

    read_lat_df = dram_df[dram_df['What'] == 'read_avg_latency']
    write_lat_df = dram_df[dram_df['What'] == 'write_avg_latency']

    if read_lat_df.empty and write_lat_df.empty:
        print("No DRAM latency data found in database.")
        return None

    fig, ax = plt.subplots(figsize=(12, 6))

    locations = sorted(set(read_lat_df['Location'].tolist() + write_lat_df['Location'].tolist()))
    x = np.arange(len(locations))
    width = 0.35

    read_lats = []
    write_lats = []

    for loc in locations:
        read_val = read_lat_df[read_lat_df['Location'] == loc]['Value'].values
        write_val = write_lat_df[write_lat_df['Location'] == loc]['Value'].values

        read_lats.append(read_val[0] * 1e9 if len(read_val) > 0 else 0)  # Convert to nanoseconds
        write_lats.append(write_val[0] * 1e9 if len(write_val) > 0 else 0)

    ax.bar(x - width/2, read_lats, width, label='Read Latency', color='lightblue')
    ax.bar(x + width/2, write_lats, width, label='Write Latency', color='salmon')

    ax.set_xlabel('DRAM Component')
    ax.set_ylabel('Average Latency (ns)')
    ax.set_title('DRAM Average Read/Write Latencies')
    ax.set_xticks(x)
    ax.set_xticklabels(locations, rotation=45, ha='right')
    ax.legend()
    ax.grid(axis='y', alpha=0.3)

    plt.tight_layout()
    return fig


def plot_memory_bandwidth_utilization(df):
    """Plot memory bandwidth utilization (combined read + write)."""
    dram_df = df[df['Location'].str.contains('DRAM', case=False, na=False)]

    read_size_df = dram_df[dram_df['What'] == 'read_size']
    write_size_df = dram_df[dram_df['What'] == 'write_size']

    if read_size_df.empty and write_size_df.empty:
        print("No DRAM size data found in database.")
        return None

    # Get kernel time to calculate bandwidth
    kernel_time_df = df[df['What'] == 'kernel_time']
    if kernel_time_df.empty:
        print("No kernel time data found. Cannot calculate bandwidth.")
        return None

    kernel_time = kernel_time_df.iloc[0]['Value']  # Get total kernel execution time

    fig, ax = plt.subplots(figsize=(12, 6))

    locations = sorted(set(read_size_df['Location'].tolist() + write_size_df['Location'].tolist()))

    bandwidths = []
    total_data = []

    for loc in locations:
        read_val = read_size_df[read_size_df['Location'] == loc]['Value'].values
        write_val = write_size_df[write_size_df['Location'] == loc]['Value'].values

        read_bytes = read_val[0] if len(read_val) > 0 else 0
        write_bytes = write_val[0] if len(write_val) > 0 else 0
        total_bytes = read_bytes + write_bytes

        # Calculate bandwidth in GB/s
        bandwidth_gbps = (total_bytes / kernel_time) / (1024**3)
        bandwidths.append(bandwidth_gbps)
        total_data.append(total_bytes / (1024**2))  # MB

    x = np.arange(len(locations))

    # Create bar plot for bandwidth
    colors = plt.cm.viridis(np.linspace(0.3, 0.9, len(locations)))
    bars = ax.bar(x, bandwidths, color=colors)

    ax.set_xlabel('DRAM Component')
    ax.set_ylabel('Bandwidth (GB/s)')
    ax.set_title(f'Memory Bandwidth Utilization\n(Total Kernel Time: {kernel_time:.4f}s)')
    ax.set_xticks(x)
    ax.set_xticklabels(locations, rotation=45, ha='right')
    ax.grid(axis='y', alpha=0.3)

    # Add value labels on bars
    for i, (bar, bw, data) in enumerate(zip(bars, bandwidths, total_data)):
        height = bar.get_height()
        ax.text(bar.get_x() + bar.get_width()/2., height,
                f'{bw:.2f} GB/s\n({data:.1f} MB)',
                ha='center', va='bottom', fontsize=8)

    plt.tight_layout()
    return fig


def plot_bandwidth_metrics(df):
    """Plot bandwidth metrics from memUtilTracer (NEW metrics)."""
    # Filter for bandwidth metrics
    bw_df = df[df['What'].isin(['read_bandwidth', 'write_bandwidth', 'total_bandwidth'])]

    if bw_df.empty:
        print("No bandwidth metrics found. (Run with --report-mem-util)")
        return None

    # Get unique locations
    locations = sorted(bw_df['Location'].unique())

    fig, ax = plt.subplots(figsize=(14, 6))

    x = np.arange(len(locations))
    width = 0.25

    read_bw = []
    write_bw = []
    total_bw = []

    for loc in locations:
        loc_data = bw_df[bw_df['Location'] == loc]

        read_val = loc_data[loc_data['What'] == 'read_bandwidth']['Value'].values
        write_val = loc_data[loc_data['What'] == 'write_bandwidth']['Value'].values
        total_val = loc_data[loc_data['What'] == 'total_bandwidth']['Value'].values

        # Convert to GB/s
        read_bw.append((read_val[0] / (1024**3)) if len(read_val) > 0 else 0)
        write_bw.append((write_val[0] / (1024**3)) if len(write_val) > 0 else 0)
        total_bw.append((total_val[0] / (1024**3)) if len(total_val) > 0 else 0)

    ax.bar(x - width, read_bw, width, label='Read BW', color='skyblue')
    ax.bar(x, write_bw, width, label='Write BW', color='lightcoral')
    ax.bar(x + width, total_bw, width, label='Total BW', color='mediumpurple')

    ax.set_xlabel('Memory Component')
    ax.set_ylabel('Bandwidth (GB/s)')
    ax.set_title('Memory Bandwidth (from memUtilTracer)')
    ax.set_xticks(x)
    ax.set_xticklabels(locations, rotation=45, ha='right')
    ax.legend()
    ax.grid(axis='y', alpha=0.3)

    plt.tight_layout()
    return fig


def plot_outstanding_requests(df):
    """Plot outstanding request metrics."""
    # Filter for outstanding request metrics
    out_df = df[df['What'].isin(['avg_outstanding_requests', 'max_outstanding_requests'])]

    if out_df.empty:
        print("No outstanding request metrics found.")
        return None

    locations = sorted(out_df['Location'].unique())

    fig, ax = plt.subplots(figsize=(12, 6))

    x = np.arange(len(locations))
    width = 0.35

    avg_out = []
    max_out = []

    for loc in locations:
        loc_data = out_df[out_df['Location'] == loc]

        avg_val = loc_data[loc_data['What'] == 'avg_outstanding_requests']['Value'].values
        max_val = loc_data[loc_data['What'] == 'max_outstanding_requests']['Value'].values

        avg_out.append(avg_val[0] if len(avg_val) > 0 else 0)
        max_out.append(max_val[0] if len(max_val) > 0 else 0)

    ax.bar(x - width/2, avg_out, width, label='Average', color='steelblue')
    ax.bar(x + width/2, max_out, width, label='Maximum', color='indianred')

    ax.set_xlabel('Memory Component')
    ax.set_ylabel('Outstanding Requests')
    ax.set_title('Outstanding Memory Requests')
    ax.set_xticks(x)
    ax.set_xticklabels(locations, rotation=45, ha='right')
    ax.legend()
    ax.grid(axis='y', alpha=0.3)

    plt.tight_layout()
    return fig


def plot_request_counts(df):
    """Plot memory request counts from memUtilTracer."""
    # Filter for request count metrics
    req_df = df[df['What'].isin(['read_request_count', 'write_request_count'])]

    if req_df.empty:
        print("No request count metrics found.")
        return None

    locations = sorted(req_df['Location'].unique())

    fig, ax = plt.subplots(figsize=(12, 6))

    x = np.arange(len(locations))
    width = 0.35

    read_reqs = []
    write_reqs = []

    for loc in locations:
        loc_data = req_df[req_df['Location'] == loc]

        read_val = loc_data[loc_data['What'] == 'read_request_count']['Value'].values
        write_val = loc_data[loc_data['What'] == 'write_request_count']['Value'].values

        read_reqs.append(read_val[0] if len(read_val) > 0 else 0)
        write_reqs.append(write_val[0] if len(write_val) > 0 else 0)

    ax.bar(x - width/2, read_reqs, width, label='Read Requests', color='steelblue')
    ax.bar(x + width/2, write_reqs, width, label='Write Requests', color='indianred')

    ax.set_xlabel('Memory Component')
    ax.set_ylabel('Request Count')
    ax.set_title('Memory Request Counts (from memUtilTracer)')
    ax.set_xticks(x)
    ax.set_xticklabels(locations, rotation=45, ha='right')
    ax.legend()
    ax.grid(axis='y', alpha=0.3)

    plt.tight_layout()
    return fig


def main():
    if len(sys.argv) < 2:
        print("Usage: python plot_memory_utilization.py <database_file.sqlite3>")
        print("\nTo generate a database file, run a simulation with flags:")
        print("  ./benchmark -timing -report-all")
        print("\nThis will create a file like: akita_sim_<id>.sqlite3")
        sys.exit(1)

    db_path = sys.argv[1]

    try:
        # Load metrics from database
        print(f"Loading metrics from {db_path}...")
        df = load_metrics(db_path)

        print(f"Total metrics loaded: {len(df)}")
        print(f"\nAvailable metric types:")
        for metric_type in sorted(df['What'].unique()):
            count = len(df[df['What'] == metric_type])
            print(f"  - {metric_type}: {count} entries")

        # Generate plots
        print("\nGenerating plots...")

        figures = []

        # OLD metrics (from dramTracer)
        print("\n--- Old DRAM Metrics ---")

        # Plot 1: Memory sizes
        fig1 = plot_dram_memory_usage(df)
        if fig1:
            figures.append(('dram_memory_sizes', fig1))

        # Plot 2: Transaction counts
        fig2 = plot_dram_transaction_counts(df)
        if fig2:
            figures.append(('dram_transaction_counts', fig2))

        # Plot 3: Latencies
        fig3 = plot_dram_latencies(df)
        if fig3:
            figures.append(('dram_latencies', fig3))

        # Plot 4: Bandwidth utilization (calculated)
        fig4 = plot_memory_bandwidth_utilization(df)
        if fig4:
            figures.append(('dram_bandwidth_calculated', fig4))

        # NEW metrics (from memUtilTracer)
        print("\n--- New Memory Utilization Metrics ---")

        # Plot 5: Bandwidth metrics (direct from tracer)
        fig5 = plot_bandwidth_metrics(df)
        if fig5:
            figures.append(('bandwidth_metrics', fig5))

        # Plot 6: Outstanding requests
        fig6 = plot_outstanding_requests(df)
        if fig6:
            figures.append(('outstanding_requests', fig6))

        # Plot 7: Request counts (from memUtilTracer)
        fig7 = plot_request_counts(df)
        if fig7:
            figures.append(('request_counts', fig7))

        if not figures:
            print("\nNo memory-related metrics found in database.")
            print("Make sure the simulation was run with -report-all or -report-mem-util flag.")
            return

        # Save figures
        base_name = Path(db_path).stem
        for name, fig in figures:
            output_file = f"{base_name}_{name}.png"
            fig.savefig(output_file, dpi=150, bbox_inches='tight')
            print(f"Saved: {output_file}")

        # Show all plots
        print("\nDisplaying plots...")
        plt.show()

    except FileNotFoundError as e:
        print(f"Error: {e}")
        sys.exit(1)
    except Exception as e:
        print(f"Error: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == "__main__":
    main()
