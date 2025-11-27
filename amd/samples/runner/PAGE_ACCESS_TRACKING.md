# GPU Page Access Tracking in MGPUSim

This document describes the GPU page access tracking system in MGPUSim, which enables tracking of:
- Individual page accesses by each GPU and CU
- Page sharing patterns across multiple GPUs
- Page migration events between GPUs
- Page replication events

## Overview

The page access tracking system provides detailed insights into memory access patterns in multi-GPU simulations, similar to the data shown in academic papers analyzing GPU page sharing and migration behavior.

## Architecture

### Components

1. **PageAccessTracer** (`pageaccesstracer.go`): Core tracking component that:
   - Records individual page access events with timestamps
   - Maintains page sharing statistics across GPUs
   - Tracks page migrations between GPUs
   - Stores all data in database tables

2. **Database Tables**:
   - `page_accesses`: Individual page access events
   - `page_sharing_summary`: Aggregated sharing statistics per page
   - `page_migrations`: Page migration events
   - `page_replications`: Page replication events
   - `mgpusim_metrics`: Summary metrics

3. **PMC Integration**: Automatic tracking of page migrations through callbacks in the Page Migration Controller

## Usage

### Enabling Page Access Tracking

Use the `-report-page-access` flag when running simulations:

```bash
./benchmark -report-page-access -timing -report-all
```

Or enable it programmatically in your benchmark code:

```go
runner := runner.Runner{}
runner.ReportPageAccess = true
```

### Database Schema

#### page_accesses Table

Records every page access event:

| Column | Type | Description |
|--------|------|-------------|
| Time | float64 | Simulation time of access |
| PageAddr | uint64 | Physical page address (page-aligned) |
| AccessType | string | "read" or "write" |
| GPUID | string | GPU identifier (e.g., "GPU1.PMC") |
| CUID | string | Compute Unit identifier |
| PID | int | Process ID |
| VirtualAddr | uint64 | Original virtual address |
| PhysicalAddr | uint64 | Original physical address |

#### page_sharing_summary Table

Aggregated sharing statistics per page:

| Column | Type | Description |
|--------|------|-------------|
| PageAddr | uint64 | Physical page address |
| AccessCount | int | Total number of accesses |
| GPUList | string | Comma-separated list of GPU IDs |
| NumGPUs | int | Number of unique GPUs accessing this page |
| NumReads | int | Total read accesses |
| NumWrites | int | Total write accesses |
| FirstAccess | float64 | Timestamp of first access |
| LastAccess | float64 | Timestamp of last access |

#### page_migrations Table

Page migration events between GPUs:

| Column | Type | Description |
|--------|------|-------------|
| Time | float64 | Migration completion time |
| PageAddr | uint64 | Page address being migrated |
| PageSize | uint64 | Size of page in bytes |
| SourceGPU | string | Source GPU identifier |
| DestGPU | string | Destination GPU identifier |
| SourcePhysAddr | uint64 | Source physical address |
| DestPhysAddr | uint64 | Destination physical address |
| MigrationDuration | float64 | Time taken for migration |
| DataTransferSize | uint64 | Bytes transferred |

#### page_replications Table

Page replication events:

| Column | Type | Description |
|--------|------|-------------|
| Time | float64 | Replication time |
| PageAddr | uint64 | Page address being replicated |
| PageSize | uint64 | Size of page in bytes |
| SourceGPU | string | Source GPU identifier |
| DestGPU | string | Destination GPU identifier |
| ReplicaPhysAddr | uint64 | Physical address of replica |

## Analyzing Results

### Using SQLite

The metrics are stored in SQLite database (typically `metrics.db`). You can query them using SQL:

```sql
-- Get pages shared by multiple GPUs
SELECT PageAddr, NumGPUs, AccessCount, GPUList
FROM page_sharing_summary
WHERE NumGPUs > 1
ORDER BY AccessCount DESC;

-- Get migration statistics
SELECT SourceGPU, DestGPU, COUNT(*) as MigrationCount,
       AVG(MigrationDuration) as AvgDuration
FROM page_migrations
GROUP BY SourceGPU, DestGPU;

-- Get page access timeline for a specific page
SELECT Time, GPUID, AccessType
FROM page_accesses
WHERE PageAddr = 0x...
ORDER BY Time;

-- Calculate sharing distribution (like Figure 4)
SELECT NumGPUs, COUNT(*) as PageCount
FROM page_sharing_summary
GROUP BY NumGPUs
ORDER BY NumGPUs;
```

### Using Python/Pandas

```python
import sqlite3
import pandas as pd
import matplotlib.pyplot as plt

# Connect to database
conn = sqlite3.connect('metrics.db')

# Load page sharing summary
df_sharing = pd.read_sql_query(
    "SELECT * FROM page_sharing_summary",
    conn
)

# Calculate sharing distribution
sharing_dist = df_sharing.groupby('NumGPUs').size()

# Plot distribution (similar to the provided figure)
fig, ax = plt.subplots()
sharing_dist.plot(kind='bar', ax=ax)
ax.set_xlabel('Number of GPUs Sharing Page')
ax.set_ylabel('Number of Pages')
ax.set_title('Distribution of Page Sharing')
plt.show()

# Load migration data
df_migrations = pd.read_sql_query(
    "SELECT * FROM page_migrations",
    conn
)

print(f"Total migrations: {len(df_migrations)}")
print(f"Average migration duration: {df_migrations['MigrationDuration'].mean()}")
```

## Advanced Usage

### Custom Page Access Tracking

To track page accesses from custom components:

```go
import "github.com/sarchlab/mgpusim/v4/amd/samples/runner"

// In your component code
if runner.GlobalPageAccessTracer != nil {
    runner.TrackMemoryAccess(
        virtualAddr,
        physicalAddr,
        "read",  // or "write"
        "GPU1",
        "GPU1.ShaderArray0.CU0",
        processID,
    )
}
```

### Custom Migration Tracking

If implementing custom migration logic:

```go
import "github.com/sarchlab/mgpusim/v4/amd/samples/runner"

// Track a custom migration
if runner.GlobalPageAccessTracer != nil {
    runner.TrackPageMigration(
        pageAddr,
        pageSize,
        "GPU1.PMC",
        "GPU2.PMC",
        sourcePhysAddr,
        destPhysAddr,
        migrationDuration,
        dataTransferSize,
    )
}
```

### Custom Replication Tracking

For page replication scenarios:

```go
import "github.com/sarchlab/mgpusim/v4/amd/samples/runner"

if runner.GlobalPageAccessTracer != nil {
    runner.TrackPageReplication(
        pageAddr,
        pageSize,
        "GPU1.PMC",
        "GPU2.PMC",
        replicaPhysAddr,
    )
}
```

## Performance Considerations

Page access tracking adds overhead to the simulation:

1. **Database writes**: Each access is recorded to the database
2. **Memory overhead**: In-memory tracking of page sharing statistics
3. **Processing overhead**: Page address calculation and data structure updates

For large simulations, consider:
- Using `-report-page-access` only when needed
- Sampling page accesses instead of recording all
- Using faster database backends (e.g., in-memory SQLite)

## Implementation Details

### Page Address Calculation

Pages are identified by their aligned physical address:
```
pageSize = 1 << log2PageSize  // Typically 4KB (log2PageSize = 12)
pageAddr = (physicalAddr / pageSize) * pageSize
```

### Migration Detection

Page migrations are automatically detected through the Page Migration Controller (PMC). When a page migration completes, the PMC callback triggers the recording of migration data.

### Sharing Analysis

The system tracks which GPUs access each page by maintaining a set of GPU IDs per page. At the end of the simulation, it generates the sharing summary with:
- Number of unique GPUs that accessed the page
- Total access count
- Read/write breakdown
- First and last access timestamps

## Troubleshooting

### No data in database tables

- Ensure `-report-page-access` or `-report-all` flag is set
- Check that page accesses are occurring in your workload
- Verify the database file is being created (check with `ls -la *.db`)

### Missing migration data

- Page migrations only occur with specific memory management policies
- Ensure your simulation uses timing mode (`-timing` flag)
- Check that PMC components are created in your GPU configuration

### Performance issues

- Consider sampling: modify `PageAccessTracer` to record only every Nth access
- Use smaller workloads for initial testing
- Disable other tracing flags if not needed

## Example Output

Sample metrics from `mgpusim_metrics` table:

```
Location: PageAccessTracer
What: total_pages_accessed
Value: 12584
Unit: count

Location: PageAccessTracer
What: total_page_migrations
Value: 342
Unit: count

Location: PageAccessTracer
What: total_page_replications
Value: 0
Unit: count
```

## References

This implementation supports the analysis of GPU page sharing patterns similar to those shown in:
- Papers analyzing multi-GPU memory management
- Studies on page migration overhead
- Research on NUMA-aware GPU scheduling

## Future Enhancements

Possible improvements:
1. Page access heatmaps over time
2. Migration path visualization
3. Automatic detection of thrashing
4. Page access pattern classification
5. Integration with visualization tools
