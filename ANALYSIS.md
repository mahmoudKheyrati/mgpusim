# Performance Analysis

This document describes the GPU trace analysis notebook (`mgpusim_analysis.ipynb`) that provides comprehensive performance insights into MGPUSim simulation traces stored in ClickHouse.

## How to Run the Notebook

### Prerequisites

1. **ClickHouse** running at `localhost:8123` with the trace data loaded (see main README for simulation instructions).
2. **Python 3.8+** with Jupyter:

```bash
pip install jupyter clickhouse-connect pandas numpy matplotlib seaborn plotly networkx
jupyter notebook mgpusim_analysis.ipynb
```

3. Edit **Section 0** connection config cell to match your ClickHouse credentials:

```python
CH_HOST     = "localhost"
CH_PORT     = 8123
CH_USER     = "mahmoud"
CH_PASSWORD = "yourpassword"
CH_DATABASE = "akitasimd7a9s1gjk23i97fmegs0"
TIME_BUCKETS = 100   # resolution of time-series charts
```

4. Run all cells: **Kernel → Restart & Run All**

All charts are saved as PNG (static) or HTML (interactive) files in the working directory.

---

## What Each Section Reveals

### Section 0 — Setup & Connection
Establishes the ClickHouse connection and defines helper functions for:
- **`parse_location(loc)`** — parses `GPU[N].SA[M].CU[K].VALU` strings into structured fields
- **`base_id(id_str)`** — strips `@location` and `_req_out` suffixes to get root request IDs
- **Global metadata** — simulation time range, total events, bucket width

### Section 1 — Workload Overview (Charts 1–4)

| Chart | What it shows |
|-------|---------------|
| **Chart 1** — Timeline heatmap | Activity density per GPU per time bucket; reveals synchronization barriers (vertical white stripes) and idle GPUs (dark rows) |
| **Chart 2** — Kind distribution | Ratio of `inst` vs `req_in`/`req_out` events; high `req_*` share = memory-bound workload |
| **Chart 3** — What distribution | Instruction and request type breakdown; VALU-heavy = compute-bound, VMem-heavy = memory-bound |
| **Chart 4** — Time coverage | Per-GPU active span and busy-time utilization %; gaps between GPU spans reveal load imbalance |

### Section 2 — GPU-level Performance (Charts 5–8)

| Chart | What it shows |
|-------|---------------|
| **Chart 5** — Instruction throughput | Instructions per nanosecond per GPU; highlights underperforming GPUs |
| **Chart 6** — Memory request volume | Read vs Write traffic per GPU; write-heavy workloads stress write-back caches |
| **Chart 7** — Load balance | Coefficient of Variation (CV) for key metrics; CV > 20% demands attention |
| **Chart 8** — Active time fraction | Fraction of total simulation time each GPU is active; idle GPUs = dispatch or synchronization problems |

#### How to Interpret Load Balance (Chart 7)

The **Coefficient of Variation (CV = std/mean × 100%)** quantifies imbalance:

- **CV < 5%** (green): Excellent — all GPUs are doing similar amounts of work
- **CV 5–20%** (orange): Acceptable for complex workloads with irregular data access
- **CV > 20%** (red): Problematic — the slowest GPU serializes the entire system (Amdahl's Law)

Root causes of high CV:
- Irregular data partitioning (e.g., graph workloads with skewed adjacency)
- Work-group dispatch not accounting for GPU compute capacity
- Inter-GPU data dependencies creating stalls on some GPUs

### Section 3 — Memory Hierarchy Analysis (Charts 9–12)

| Chart | What it shows |
|-------|---------------|
| **Chart 9** — Latency distribution | Box-plot per component level: L1VROB → L2Cache → DRAM; reveals cache miss pressure |
| **Chart 10** — L1 pressure heatmap | Requests per SA per time bucket; hot rows = cache pressure, vertical gaps = barriers |
| **Chart 11** — Flow Sankey (interactive) | Request volume flowing through CU → L1 → L2 → DRAM hierarchy |
| **Chart 12** — R/W ratio heatmap | Read vs Write balance per GPU × SA; write-dominant SAs risk saturating write buffers |

#### How to Interpret Memory Latency (Chart 9)

Expected latency ranges for the R9 Nano configuration (1 GHz):

| Level | Typical latency | Indicates if higher |
|-------|----------------|-------------------|
| L1V cache hit | 1–5 ns | — |
| L1V cache miss → L2 | 50–150 ns | High L1 miss rate |
| L2 cache hit | 100–200 ns | — |
| L2 miss → DRAM | 400–1000 ns | High L2 miss rate |
| Cross-GPU (RDMA) | 500–2000 ns | High inter-GPU traffic |

A **wide IQR** (long box) on the latency chart indicates bimodal behavior — some requests hit the cache (fast path) and others miss (slow path). If the median is close to DRAM latency, the workload has poor cache reuse.

### Section 4 — Compute Unit Deep Dive (Charts 13–16)

| Chart | What it shows |
|-------|---------------|
| **Chart 13** — CU utilization heatmap | Instruction count per SA × CU grid per GPU; spots hot/cold CUs |
| **Chart 14** — Instruction type mix | VALU/VMem/Scalar/Branch/LDS share per CU; reveals compute vs memory character |
| **Chart 15** — Pipeline stall proxy | Memory-request time / instruction time ratio; > 40% = memory-bound CU |
| **Chart 16** — Instruction exec time | Latency distribution per instruction type; wide VMem distribution = cache miss variation |

### Section 5 — Request Chain Analysis (Charts 17–19)

| Chart | What it shows |
|-------|---------------|
| **Chart 17** — Chain depth histogram | Distribution of children per parent request; log-scale reveals rare deep chains |
| **Chart 18** — Critical path waterfall | Top-10 longest latency chains shown hop-by-hop; identifies which hierarchy level dominates |
| **Chart 19** — Fan-out distribution | How many sub-requests each parent spawns; high fan-out = memory request splitting |

### Section 6 — Inter-GPU Communication (Charts 20–21)

| Chart | What it shows |
|-------|---------------|
| **Chart 20** — RDMA traffic | Event count and time per GPU at RDMA components; identifies communication hubs |
| **Chart 21** — Compute/comm overlap | Time-series of compute vs RDMA events per GPU; non-overlapping regions = serialized comm |

### Section 7 — Performance Summary Dashboard (Charts 22–23)

| Chart | What it shows |
|-------|---------------|
| **Chart 22** — Dashboard | 6 key metrics: total instructions, memory requests, avg latency, bandwidth, imbalance CV, sim time |
| **Chart 23** — Radar chart | Per-GPU normalized comparison across 5 axes: throughput, bandwidth, latency, utilization, comms |

---

## GPU Architecture Concepts Referenced

### Memory-Bound vs Compute-Bound
A workload is **compute-bound** if the bottleneck is arithmetic throughput (VALU pipelines). It is **memory-bound** if CUs spend more time waiting for data than executing instructions. The stall-rate chart (15) and instruction mix (14) together diagnose this.

Reference: *Volkov, V. (2010). Better Performance at Lower Occupancy. GPU Technology Conference.*

### Cache Thrashing
When multiple wavefronts access working sets larger than the L1V cache (32 KB per CU in GCN3), they continuously evict each other's cache lines. This appears as: high L1VROB request rate, high L1→L2 traffic in the Sankey, and bimodal latency distribution in chart 9.

Reference: *Rogers, T. G., et al. (2012). Cache-Conscious Wavefront Scheduling. MICRO-45.*

### Load Imbalance
In multi-GPU systems, the slowest GPU determines system throughput. Load imbalance arises from: irregular data structures (graphs, sparse matrices), non-uniform work-group sizes, or dynamic kernel behavior. CV > 20% typically requires data repartitioning or dynamic load redistribution.

Reference: *Sun, Y., et al. (2019). MGPUSim: Enabling Multi-GPU Performance Modeling and Optimization. ISCA-46.*

### GCN3 Compute Unit Architecture
Each CU has: 4 SIMD units (16 lanes each = 64 work-items/wavefront), 1 scalar unit, 1 branch unit, 1 vector memory unit, 1 LDS unit, and a 40-wavefront pool. Peak throughput = 4 SIMDs × 64 ops/cycle × 1 GHz = 256 GFLOPS per CU.

Reference: *AMD GCN3 Architecture Whitepaper (2015). AMD Inc.*

### Request Chaining and Critical Paths
A memory request from a CU travels: CU → L1VROB → L1VCache → L2Cache → DRAM. Each hop adds latency. The critical path is the longest non-overlapping chain. Reducing cache misses shortens this path. Section 5 directly identifies the worst-case chains.

Reference: *Bakhoda, A., et al. (2009). Analyzing CUDA Workloads Using a Detailed GPU Simulator. ISPASS.*

---

## Citation

If you use this notebook in research, please cite the MGPUSim paper:

```bibtex
@inproceedings{sun19mgpusim,
    author    = {Sun, Yifan and Baruah, Trinayan and Mojumder, Saiful A. and
                 Dong, Shi and Gong, Xiang and Treadway, Shane and Bao, Yuhui and
                 Hance, Spencer and McCardwell, Carter and Zhao, Vincent and
                 Barclay, Harrison and Ziabari, Amir Kavyan and Chen, Zhongliang and
                 Ubal, Rafael and Abellán, José L. and Kim, John and
                 Joshi, Ajay and Kaeli, David},
    title     = {MGPUSim: Enabling Multi-GPU Performance Modeling and Optimization},
    booktitle = {Proceedings of the 46th International Symposium on Computer Architecture},
    series    = {ISCA '19},
    year      = {2019},
    pages     = {197--209},
    doi       = {10.1145/3307650.3322230}
}
```
