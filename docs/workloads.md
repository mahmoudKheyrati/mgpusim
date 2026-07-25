# MGPUSim Workload Reference

A complete catalog of the 68 runnable workloads under `amd/samples/` — what each one
computes, which benchmark suite it comes from, its command-line parameters, its
GCN3/CDNA3 architecture support, whether it supports Unified Memory and true
multi-GPU data splitting, and (from the GPU-benchmarking literature) whether it is
compute-bound, memory-bound, or latency-bound. Sources are cited at the bottom.

All facts about flags/architecture/unified-memory/multi-GPU below come from reading
the actual Go source in this repo (`amd/samples/*/*.go` and `amd/benchmarks/**/*.go`),
not from documentation, since several benchmarks silently diverge from what their
`native/` leftover files would suggest.

---

## 1. How every sample is invoked

Every sample is a small `main.go` in `amd/samples/<name>/` that wires a benchmark
struct into the shared `amd/samples/runner` package. Run it with:

```bash
cd amd/samples/<name>
go run . [common runner flags] [sample-specific flags]
```

### 1.1 Common flags (apply to every sample, provided by `runner`)

| Flag | Default | Meaning |
|---|---|---|
| `-timing` | false | Run cycle-accurate timing simulation instead of fast functional emulation |
| `-arch` | `gcn3` | ISA of the kernel binary to load: `gcn3` or `cdna3` |
| `-gpu` | `r9nano` | Timing-mode hardware model to simulate: `r9nano` or `mi300x` (only matters with `-timing`) |
| `-gpus` | `1` | Comma-separated GPU IDs to use, e.g. `1,2,3,4` |
| `-unified-gpus` | "" | Same as `-gpus` but runs the set as one logical "unified" multi-GPU (cannot combine with `-gpus`) |
| `-use-unified-memory` | false | Use CPU/GPU unified (page-migratable) memory instead of explicit copies |
| `-parallel` | false | Run the simulation engine itself in parallel |
| `-verify` | false | Verify emulation output against a CPU reference |
| `-max-inst` | 0 | Stop after N instructions retire (0 = unlimited) |
| `-debug-isa` | false | Dump an ISA debug trace |
| `-magic-memory-copy` | false | Copy CPU→GPU memory directly, bypassing simulated DMA |
| `-report-all` | false | Write all metrics to a `.csv` |
| `-metric-file-name` | `metrics` | Base name of the metrics/output file |
| `-trace-mem` | false | Generate a memory access trace |
| `-report-inst-count` | false | Report instructions executed per CU |
| `-report-cache-latency` | false | Report average cache latency |
| `-report-cache-hit-rate` | false | Report per-cache hit rate |
| `-report-tlb-hit-rate` | false | Report per-TLB hit rate |
| `-report-rdma-transaction-count` | false | Report RDMA engine transaction counts (multi-GPU) |
| `-report-dram-transaction-count` | false | Report DRAM transaction counts |
| `-report-busy-time` | false | Report SIMD unit busy time |
| `-report-cpi-stack` | false | Report a CPI stack breakdown |
| `-trace-vis` | false | Emit a visualization trace (for the Akita trace viewer) |
| `-trace-vis-db` | `sqlite` | Backing store for the vis trace: `sqlite`, `mysql`, `csv` |
| `-trace-vis-db-file` | random | File name for the vis trace DB |
| `-trace-vis-start` / `-trace-vis-end` | -1 / -1 | Time window to collect the vis trace over |
| `-analyzer-name` / `-analyzer-period` | "" / 0 | Enable a periodic buffer-level analyzer |
| `-akitartm-port` / `-disable-rtm` | 0 / false | AkitaRTM live-monitoring portal port / disable it |

Only the **sample-specific** flags are listed per-workload below — don't re-derive
the ones above.

### 1.2 `-arch` vs `-gpu`: pair them correctly

`-arch` selects which HSACO ISA is loaded (GCN3 or CDNA3 — literally a different
compiled kernel binary) and `-gpu` selects which hardware timing model is built
(R9 Nano or MI300X). **The runner does not check that these agree** — nothing stops
you from running a CDNA3-only kernel against the `r9nano` timing model. In practice
you should always pair them:

- GCN3 kernels → `-arch gcn3 -gpu r9nano` (defaults, can be omitted)
- CDNA3-only kernels → `-arch cdna3 -gpu mi300x` (**required**, see §3 — most
  workloads in this repo are CDNA3-only)

### 1.3 GPU hardware models

| Model | Compute Units | Frequency | L2 cache | Simulated DRAM/GPU | Notes |
|---|---|---|---|---|---|
| `r9nano` (default) | 64 (16 shader arrays × 4 CU) | 1.0 GHz | 2 MB | 4 GB | Models the AMD R9 Nano (Fiji, GCN3) |
| `mi300x` | 304 (8 XCD × 19 SA × 2 CU) | 2.1 GHz | 4 MB | 4 GB (simplified; real MI300X has 192 GB HBM3, backed by a 256 MB Infinity Cache/MALL in the model) | Models an AMD Instinct MI300X (CDNA3) |

---

## 2. Quick-reference table

"Split?" = whether `-gpus 1,2,...` actually partitions the workload's data across
GPUs (true data-parallelism) vs. just accepting the flag without splitting
(effectively single-GPU) vs. hard single-GPU-only.

| Sample | Suite | Arch | Unified Mem | Multi-GPU | Compute/Memory profile |
|---|---|---|---|---|---|
| bitonicsort | AMD APP SDK | Both | ✅ | ✅ Split | Memory-bound |
| fastwalshtransform | AMD APP SDK | Both | ✅ | ⚠️ Redundant (no split) | Memory-bound |
| floydwarshall | AMD APP SDK | Both | ✅ | ⚠️ Not split | Compute-bound (at scale) |
| matrixmultiplication | AMD APP SDK | Both | ✅ | ✅ Split | Compute-bound |
| matrixtranspose | AMD APP SDK | Both | ✅ | ✅ Split | Memory-bound |
| nbody | AMD APP SDK | Both | ✅ | ⚠️ Not split | Compute-bound |
| simpleconvolution | AMD APP SDK | Both | ✅ | ✅ Split | Memory-bound (small mask) → Compute-bound (large mask) |
| vectoradd | AMD APP SDK | **GCN3 only** | ✅ | ✅ Split | Memory-bound |
| rodinia_backprop | Rodinia | **CDNA3 only** | ✅ | ❌ Single-GPU | Mixed, memory-leaning |
| rodinia_gaussian | Rodinia | **CDNA3 only** | ✅ | ❌ Single-GPU | Mixed / compute-leaning |
| rodinia_hotspot | Rodinia | **CDNA3 only** | ✅ | ❌ Single-GPU | Compute-bound |
| rodinia_hotspot3d | Rodinia | **CDNA3 only** | ✅ | ❌ Single-GPU | Compute-bound |
| rodinia_lavamd | Rodinia | **CDNA3 only** | ✅ | ❌ Single-GPU | Compute-bound |
| rodinia_lud | Rodinia | **CDNA3 only** | ✅ | ❌ Single-GPU | Compute-bound (mixed at small N) |
| rodinia_pathfinder | Rodinia | **CDNA3 only** | ✅ | ❌ Single-GPU | Memory-bound |
| rodinia_srad | Rodinia | **CDNA3 only** | ✅ | ❌ Single-GPU | Compute-bound |
| nw | Rodinia | Both | ✅ | ❌ Single-GPU (explicit panic) | Memory-bound |
| bfs | SHOC | Both | ✅ | ❌ Single-GPU (explicit panic) | Memory-bound |
| fft | SHOC | Both | ✅ | ⚠️ Not split | Compute-bound |
| spmv | SHOC | Both | ✅ | ⚠️ Not split | Memory-bound |
| stencil2d | SHOC | Both | ✅ | ⚠️ Not split | Memory-bound |
| parboil_cutcp | Parboil | **CDNA3 only** | ✅ | ❌ Single-GPU | Compute-bound |
| parboil_lbm | Parboil | **CDNA3 only** | ✅ | ❌ Single-GPU | Memory-bound |
| parboil_sgemm | Parboil | **CDNA3 only** | ✅ | ❌ Single-GPU | Compute-bound |
| parboil_stencil | Parboil | **CDNA3 only** | ✅ | ❌ Single-GPU | Memory-bound |
| atax | PolyBench | Both | ✅ | ⚠️ Not split | Memory-bound |
| bicg | PolyBench | Both | ✅ | ⚠️ Not split | Memory-bound |
| polybench_2dconv | PolyBench | **CDNA3 only** | ✅ | ❌ Single-GPU | Memory-bound |
| polybench_2mm | PolyBench | **CDNA3 only** | ✅ | ❌ Single-GPU | Compute-bound |
| polybench_3dconv | PolyBench | **CDNA3 only** | ✅ | ❌ Single-GPU | Memory-bound |
| polybench_3mm | PolyBench | **CDNA3 only** | ✅ | ❌ Single-GPU | Compute-bound |
| polybench_correlation | PolyBench | **CDNA3 only** | ✅ | ❌ Single-GPU | Mixed (memory-bound passes + compute-bound matmul) |
| polybench_fdtd2d | PolyBench | **CDNA3 only** | ✅ | ❌ Single-GPU | Memory-bound |
| polybench_gemm | PolyBench | **CDNA3 only** | ✅ | ❌ Single-GPU | Compute-bound |
| polybench_gramschmidt | PolyBench | **CDNA3 only** | ✅ | ❌ Single-GPU | Memory-bound |
| polybench_jacobi2d | PolyBench | **CDNA3 only** | ✅ | ❌ Single-GPU | Memory-bound |
| polybench_mvt | PolyBench | **CDNA3 only** | ✅ | ❌ Single-GPU | Memory-bound |
| polybench_syr2k | PolyBench | **CDNA3 only** | ✅ | ❌ Single-GPU | Compute-bound |
| npb_ep | NPB | **CDNA3 only** | ✅ | ❌ Single-GPU | Compute-bound |
| tango_binomial_options | Tango | **CDNA3 only** | ✅ | ❌ Single-GPU | Compute-bound |
| tango_blackscholes | Tango | **CDNA3 only** | ✅ | ❌ Single-GPU | Compute-bound |
| altis_cfd | Altis | **CDNA3 only** | ✅ | ❌ Single-GPU | Compute-bound (mixed) |
| altis_raytracing | Altis | **CDNA3 only** | ✅ | ❌ Single-GPU | Compute-bound |
| aes | HeteroMark | Both | ✅ | ✅ Split | Compute-bound |
| fir | HeteroMark | Both | ✅ | ✅ Split | Mixed (scales with taps) |
| kmeans | HeteroMark | Both | ✅ | ✅ Split | Memory-bound |
| pagerank | HeteroMark | Both | ✅ | ⚠️ Not split | Memory-bound |
| conv2d | DNN (layer) | Both | ❌ Not wired | ❌ Single-GPU (panics) | Compute-bound |
| im2col | DNN (layer) | Both | ❌ Not wired | ❌ Single-GPU (panics) | Memory-bound |
| relu | DNN (layer) | Both | ✅ | ✅ Split | Memory-bound |
| lenet | DNN (training) | Both* | ❌ Not supported (panics) | ✅ Data-parallel (mccl) | Compute-bound |
| minerva | DNN (training) | Both* | ❌ Not supported (panics) | ✅ Data-parallel (mccl) | Mixed, memory-leaning |
| vgg16 | DNN (training) | Both* | ❌ Not supported (panics) | ✅ Data-parallel (mccl) | Compute-bound |
| xor | DNN (training) | **GCN3 only*** | ❌ Not supported (panics) | ❌ Single-GPU (panics) | Overhead-bound (trivial size) |
| cache_latency | Microbench | **CDNA3 only** | ✅ | ❌ Single-GPU | Latency-bound |
| empty_kernel | Microbench | **CDNA3 only** | ✅ | ❌ Single-GPU | Overhead-bound |
| fp16_throughput | Microbench | **CDNA3 only** | ✅ | ❌ Single-GPU | Compute-bound |
| fp32_throughput | Microbench | **CDNA3 only** | ✅ | ❌ Single-GPU | Compute-bound |
| fp64_throughput | Microbench | **CDNA3 only** | ✅ | ❌ Single-GPU | Compute-bound |
| int32_throughput | Microbench | **CDNA3 only** | ✅ | ❌ Single-GPU | Compute-bound |
| memory_bandwidth | Microbench | **CDNA3 only** | ✅ | ❌ Single-GPU | Memory-bound |
| mixbench | Microbench | **CDNA3 only** | ✅ | ❌ Single-GPU | Configurable (memory→compute sweep) |
| shared_mem_bandwidth | Microbench | **CDNA3 only** | ✅ | ❌ Single-GPU | Memory-bound (LDS) |
| shared_mem_latency | Microbench | **CDNA3 only** | ✅ | ❌ Single-GPU | Latency-bound (LDS) |
| babelstream | Misc | **CDNA3 only** | ✅ | ❌ Single-GPU | Memory-bound |
| memcopy | Misc | N/A (no kernel) | ✅ | ❌ Single-GPU | Memory-bound |
| concurrentkernel | Misc (demo) | Both** | ⚠️ Not enabled | Single GPU, 2 concurrent kernels | Composite |
| concurrentworkload | Misc (demo) | Both** | ⚠️ Not enabled | ✅ 3 GPUs, 2 concurrent workloads | Composite |

\* `lenet`/`minerva`/`vgg16` sit on an architecture-capable engine, but their sample
`main.go` never wires `-arch` through, so they always run the GCN3 path today.
\** `xor`, `concurrentkernel`, `concurrentworkload` likewise never set `.Arch`, so
they run GCN3 regardless of the flag.

---

## 3. AMD APP SDK samples

### bitonicsort
- **Package**: `amd/benchmarks/amdappsdk/bitonicsort`
- **Computes**: Sorts an array of random uint32 values in place using a bitonic sorting network, one kernel launch per compare-exchange stage.
- **Flags**: `-length int` (default 1024) — array length to sort; `-order-asc bool` (default true) — ascending vs. descending.
- **Architecture**: Both (embeds `kernels.hsaco` + `kernels_gfx942.hsaco`, selected by `-arch`).
- **Unified memory**: Supported.
- **Multi-GPU**: Real data split — `exec()` divides the `Length/2` compare-exchange work-items across the GPU queues in `-gpus`.
- **Profile**: Memory-bound — a sort network does O(1) work (one compare-swap) per O(1) bytes moved per stage, and needs `log²(n)` full passes over the array; data movement dominates.
```bash
go run . -gpus 1,2 -timing -trace-vis -report-all -metric-file-name bitonicsort-test -length 65536 -order-asc=true
```

### fastwalshtransform
- **Package**: `amd/benchmarks/amdappsdk/fastwalshtransform`
- **Computes**: In-place Fast Walsh-Hadamard Transform of a float32 array, `log2(length)` kernel launches (one per butterfly stage).
- **Flags**: `-length int` (default 1024).
- **Architecture**: Both.
- **Unified memory**: Supported.
- **Multi-GPU**: Flag accepted, but each GPU redundantly recomputes the whole transform on a shared buffer — no actual split.
- **Profile**: Memory-bound — like FFT's butterfly network, one add/sub pair per element per stage; extremely low arithmetic intensity.
```bash
go run . -gpus 1 -timing -trace-vis -report-all -metric-file-name fastwalshtransform-test -length 65536
```

### floydwarshall
- **Package**: `amd/benchmarks/amdappsdk/floydwarshall`
- **Computes**: All-pairs shortest path on a random N×N graph via the classic O(N³) Floyd-Warshall triple loop, N kernel-launch iterations (one per intermediate vertex `k`).
- **Flags**: `-node int` (default 16) — graph size; `-iter int` (default 0, 0 = auto = N).
- **Architecture**: Both.
- **Unified memory**: Supported.
- **Multi-GPU**: Flag accepted but not used to split data.
- **Profile**: Compute-bound at scale — O(N³) `min+add` operations reuse the O(N²) distance matrix N times (high data reuse ratio), so arithmetic intensity grows with N even though each cell's per-visit work is tiny.
```bash
go run . -gpus 1 -timing -trace-vis -report-all -metric-file-name floydwarshall-test -node 512 -iter 0
```

### matrixmultiplication
- **Package**: `amd/benchmarks/amdappsdk/matrixmultiplication`
- **Computes**: Dense GEMM C = A×B (A is X×Y, B is Y×Z) using an LDS-blocked tiled kernel.
- **Flags**: `-x uint` (default 64), `-y uint` (default 64), `-z uint` (default 64) — matrix dimensions.
- **Architecture**: Both.
- **Unified memory**: Supported.
- **Multi-GPU**: Real data split — rows of C are partitioned across `len(gpus)` and each GPU gets its own kernel launch with a row offset.
- **Profile**: Compute-bound — classic dense GEMM, O(n³) FLOPs vs O(n²) data with LDS tiling to maximize reuse; the textbook compute-bound kernel.
```bash
go run . -gpus 1,2 -timing -trace-vis -report-all -metric-file-name mm-test -x 512 -y 512 -z 512
```

### matrixtranspose
- **Package**: `amd/benchmarks/amdappsdk/matrixtranspose`
- **Computes**: Transposes a square Width×Width matrix using LDS-tiled 16×16 blocks, 4 elements/thread.
- **Flags**: `-width int` (default 256).
- **Architecture**: Both.
- **Unified memory**: Supported.
- **Multi-GPU**: Real data split — column range partitioned per GPU via a `GroupXOffset`.
- **Profile**: Memory-bound — zero arithmetic, pure data movement (read once, write once per element); a bandwidth benchmark in disguise.
```bash
go run . -gpus 1,2 -timing -trace-vis -report-all -metric-file-name transpose-test -width 1024
```

### nbody
- **Package**: `amd/benchmarks/amdappsdk/nbody`
- **Computes**: Gravitational N-body simulation, `NumParticles` bodies, leapfrog position/velocity integration over `NumIterations` steps with a softening factor.
- **Flags**: `-particles int` (default 1024), `-iter int` (default 8).
- **Architecture**: Both (CDNA3 build uses global memory instead of LDS for `LocalPos`).
- **Unified memory**: Supported.
- **Multi-GPU**: Flag accepted, but no per-GPU particle partition exists in the code.
- **Profile**: Compute-bound — O(n²) pairwise force evaluations reuse each particle's position across n interactions; classic high-arithmetic-intensity kernel.
```bash
go run . -gpus 1 -timing -trace-vis -report-all -metric-file-name nbody-test -particles 4096 -iter 8
```

### simpleconvolution
- **Package**: `amd/benchmarks/amdappsdk/simpleconvolution`
- **Computes**: Applies a non-separable maskSize×maskSize convolution filter to a Width×Height image.
- **Flags**: `-width uint` (default 254), `-height uint` (default 254), `-mask-size uint` (default 3).
- **Architecture**: Both.
- **Unified memory**: Supported.
- **Multi-GPU**: Real data split — grid rows partitioned across GPUs with a `HiddenGlobalOffsetX`.
- **Profile**: Compute-to-memory ratio scales with `mask-size²` — memory-bound at the default 3×3 mask (9 MACs per pixel load), shifting toward compute-bound as the mask grows.
```bash
go run . -gpus 1,2 -timing -trace-vis -report-all -metric-file-name simpleconv-test -width 1024 -height 1024 -mask-size 7
```

### vectoradd
- **Package**: `amd/benchmarks/amdappsdk/vectoradd`
- **Computes**: Elementwise vector add C = B + C over Width×Height float32 elements.
- **Flags**: `-width uint` (default 1024), `-height uint` (default 1024).
- **Architecture**: **GCN3 only** — no `Arch` field/CDNA3 embed exists in this package; `-arch cdna3` has no effect here.
- **Unified memory**: Supported.
- **Multi-GPU**: Real data split (row-major partition, `Distribute()` used).
- **Profile**: Memory-bound — the textbook 1 FLOP / 2 loads + 1 store STREAM-style kernel; arithmetic intensity is at its theoretical floor.
```bash
go run . -gpus 1,2 -timing -trace-vis -report-all -metric-file-name vectoradd-test -width 4096 -height 4096
```

---

## 4. Rodinia samples

> **Important**: every Rodinia benchmark below except `nw` ships only a CDNA3
> (`gfx942`) kernel — the `native/*.hsaco` files sitting next to them are unused
> leftovers. They panic if you don't pass `-arch cdna3`.

### rodinia_backprop
- **Package**: `amd/benchmarks/rodinia/backprop`
- **Computes**: One forward+backward training epoch of a 2-layer NN (Input→Hidden→Output), 6 kernels (forward/backward passes + weight updates), sigmoid activation.
- **Flags**: `-input int` (default 64), `-hidden int` (default 32), `-output int` (default 4).
- **Architecture**: CDNA3 only. **Unified memory**: Supported. **Multi-GPU**: Single-GPU only.
- **Profile**: Mixed, memory-leaning — per the original Rodinia characterization, backprop has fairly low arithmetic intensity per weight touched.
```bash
go run . -arch cdna3 -gpu mi300x -timing -trace-vis -report-all -metric-file-name backprop-test -input 4096 -hidden 1024 -output 16
```

### rodinia_gaussian
- **Package**: `amd/benchmarks/rodinia/gaussian`
- **Computes**: Dense N×N Gaussian elimination (GPU forward elimination + CPU back-substitution).
- **Flags**: `-size int` (default 64).
- **Architecture**: CDNA3 only. **Unified memory**: Supported. **Multi-GPU**: Single-GPU only.
- **Profile**: Mixed / compute-leaning at larger N — O(n³) total elimination work over O(n²) data.
```bash
go run . -arch cdna3 -gpu mi300x -timing -trace-vis -report-all -metric-file-name gaussian-test -size 512
```

### rodinia_hotspot
- **Package**: `amd/benchmarks/rodinia/hotspot`
- **Computes**: Iterative 2D thermal stencil (temperature updated from 4 neighbors + power + resistances) over `NumIterations` steps.
- **Flags**: `-size int` (default 32), `-iterations int` (default 10).
- **Architecture**: CDNA3 only. **Unified memory**: Supported. **Multi-GPU**: Single-GPU only.
- **Profile**: Compute-bound — explicitly characterized as compute-intensive in the original Rodinia IISWC'09 paper (unlike most stencils, HotSpot's per-cell physics adds enough FLOPs to offset the low reuse).
```bash
go run . -arch cdna3 -gpu mi300x -timing -trace-vis -report-all -metric-file-name hotspot-test -size 512 -iterations 100
```

### rodinia_hotspot3d
- **Package**: `amd/benchmarks/rodinia/hotspot3d`
- **Computes**: 3D version of hotspot over an N×N×N grid, 6-neighbor stencil, ping-ponged over `NumIterations`.
- **Flags**: `-size int` (default 32), `-iterations int` (default 2), `-amb-temp float64` (default 80.0).
- **Architecture**: CDNA3 only. **Unified memory**: Supported. **Multi-GPU**: Single-GPU only.
- **Profile**: Compute-bound — same physics kernel as hotspot, extended to 3D.
```bash
go run . -arch cdna3 -gpu mi300x -timing -trace-vis -report-all -metric-file-name hotspot3d-test -size 64 -iterations 10 -amb-temp 80.0
```

### rodinia_lavamd
- **Package**: `amd/benchmarks/rodinia/lavamd`
- **Computes**: Short-range molecular dynamics (Lennard-Jones potential) with a cell-list of NumBoxes³ boxes; each box's particles interact with its own + up to 26 neighbor boxes.
- **Flags**: `-num-boxes int` (default 4), `-particles-per-box int` (default 100).
- **Architecture**: CDNA3 only. **Unified memory**: Supported. **Multi-GPU**: Single-GPU only.
- **Profile**: Compute-bound — LJ pairwise force evaluation is FLOP-heavy per particle pair with high data reuse; classic compute-bound MD kernel.
```bash
go run . -arch cdna3 -gpu mi300x -timing -trace-vis -report-all -metric-file-name lavamd-test -num-boxes 8 -particles-per-box 100
```

### rodinia_lud
- **Package**: `amd/benchmarks/rodinia/lud`
- **Computes**: Blocked LU factorization (no pivoting) of an N×N matrix (block size 16), 3 kernels per diagonal block (diagonal/perimeter/internal update).
- **Flags**: `-size int` (default 128, must be a multiple of 16).
- **Architecture**: CDNA3 only. **Unified memory**: Supported. **Multi-GPU**: Single-GPU only.
- **Profile**: Compute-bound at scale (mixed for small N) — dense blocked linear algebra, GEMM-like Schur-complement update dominates for larger N.
```bash
go run . -arch cdna3 -gpu mi300x -timing -trace-vis -report-all -metric-file-name lud-test -size 512
```

### rodinia_pathfinder
- **Package**: `amd/benchmarks/rodinia/pathfinder`
- **Computes**: Dynamic-programming min-cost path through a 2D weight grid, one row-kernel launch per row, double-buffered.
- **Flags**: `-rows int` (default 64), `-cols int` (default 128).
- **Architecture**: CDNA3 only. **Unified memory**: Supported. **Multi-GPU**: Single-GPU only.
- **Profile**: Memory-bound — per the Rodinia characterization, minimal compute per cell (a min + add), streaming access pattern.
```bash
go run . -arch cdna3 -gpu mi300x -timing -trace-vis -report-all -metric-file-name pathfinder-test -rows 1000 -cols 1000
```

### rodinia_srad
- **Package**: `amd/benchmarks/rodinia/srad`
- **Computes**: Speckle-Reducing Anisotropic Diffusion (Perona-Malik) image smoothing; `srad1` (gradients+diffusion coeff) then `srad2` (update), `NumIterations` times.
- **Flags**: `-size int` (default 32), `-iterations int` (default 10).
- **Architecture**: CDNA3 only. **Unified memory**: Supported. **Multi-GPU**: Single-GPU only.
- **Profile**: Compute-bound — explicitly characterized as compute-intensive in the original Rodinia paper.
```bash
go run . -arch cdna3 -gpu mi300x -timing -trace-vis -report-all -metric-file-name srad-test -size 512 -iterations 50
```

### nw (Needleman-Wunsch)
- **Package**: `amd/benchmarks/rodinia/nw`
- **Computes**: Global sequence alignment DP using BLOSUM62, filled diagonal-block-wave by diagonal-block-wave.
- **Flags**: `-length int` (default 64) — sequence length; the DP matrix is (length+1)².
- **Architecture**: **Both** (the one Rodinia benchmark that still ships GCN3 too).
- **Unified memory**: Supported.
- **Multi-GPU**: Explicitly panics ("nw does not support multi-GPU mode") if `len(gpus) > 1`.
- **Profile**: Memory-bound — per the Rodinia paper, low compute per DP cell (comparison + max of 3), streaming/diagonal access pattern.
```bash
go run . -timing -trace-vis -report-all -metric-file-name nw-test -length 2048
```

---

## 5. SHOC samples

### bfs
- **Package**: `amd/benchmarks/shoc/bfs`
- **Computes**: Level-synchronous BFS over a random or file-loaded directed graph, frontier expansion kernel per level up to `-depth`.
- **Flags**: `-load-graph string` (path to an edge-list file); `-node int` (default 64, generated-graph width); `-degree int` (default 3, generated-graph degree); `-depth int` (default 0 = unlimited). Note: `-node`/`-degree` cannot combine with `-load-graph`.
- **Architecture**: Both. **Unified memory**: Supported.
- **Multi-GPU**: Explicitly panics ("BFS does not support multi-GPU execution yet") if `len(gpus) > 1`.
- **Profile**: Memory-bound — irregular, data-dependent graph traversal with poor locality; canonical memory-bound graph workload.
```bash
go run . -timing -trace-vis -report-all -metric-file-name bfs-test -node 1000 -degree 4 -depth 0
```

### fft
- **Package**: `amd/benchmarks/shoc/fft`
- **Computes**: SHOC `fft1D_512` radix FFT over a batch of interleaved complex float pairs, `-passes` launches.
- **Flags**: `-MB int` (default 8, data size in MB); `-bytes int` (default 0, overrides `-MB` if >0); `-passes int` (default 1).
- **Architecture**: Both. **Unified memory**: Supported.
- **Multi-GPU**: Flag accepted, creates one queue per GPU, but only a single shared buffer is ever launched against — effectively single-GPU.
- **Profile**: Compute-bound — SHOC categorizes FFT as one of its more compute-intensive Level-1 kernels; the radix butterfly has moderate-to-high FLOP/byte.
```bash
go run . -timing -trace-vis -report-all -metric-file-name fft-test -MB 32 -passes 4
```

### spmv
- **Package**: `amd/benchmarks/shoc/spmv`
- **Computes**: CSR sparse matrix × dense vector (`spmv_csr_scalar_kernel`) on a synthetically-generated sparse Dim×Dim matrix.
- **Flags**: `-dim int` (default 128); `-sparsity float64` (default 0.01, non-zero ratio).
- **Architecture**: Both. **Unified memory**: Supported.
- **Multi-GPU**: Same "queue per GPU but single shared buffer" pattern as fft — effectively single-GPU.
- **Profile**: Memory-bound — classic sparse BLAS; irregular column-index indirection gives very low arithmetic intensity (1 FMA per nonzero, plus index loads).
```bash
go run . -timing -trace-vis -report-all -metric-file-name spmv-test -dim 4096 -sparsity 0.01
```

### stencil2d
- **Package**: `amd/benchmarks/shoc/stencil2d`
- **Computes**: SHOC 2D 9-point stencil averaging over a padded grid, ping-ponged for `-iter` steps.
- **Flags**: `-row int` (default 64), `-col int` (default 64), `-iter int` (default 1). (Sample internally adds a 2-cell halo to each dimension.)
- **Architecture**: Both. **Unified memory**: Supported.
- **Multi-GPU**: Same pattern as fft/spmv — effectively single-GPU.
- **Profile**: Memory-bound — SHOC uses stencil2d specifically as a bandwidth probe; 9 loads + a handful of FLOPs per output cell.
```bash
go run . -timing -trace-vis -report-all -metric-file-name stencil2d-test -row 512 -col 512 -iter 10
```

---

## 6. Parboil samples

> All 4 Parboil ports in this repo are **CDNA3-only** — no GCN3 kernel is embedded.

### parboil_cutcp
- **Package**: `amd/benchmarks/parboil/cutcp`
- **Computes**: Coulombic potential with cutoff — direct summation of point-charge interactions within a cutoff radius, evaluated at each 3D grid point.
- **Flags**: `-num-atoms int` (default 64), `-grid-side int` (default 8), `-grid-spacing float64` (default 0.5), `-cutoff float64` (default 12.0).
- **Profile**: Compute-bound — direct-summation cutoff electrostatics has a high FLOP count per atom-grid pair relative to the bytes touched.
```bash
go run . -arch cdna3 -gpu mi300x -timing -trace-vis -report-all -metric-file-name cutcp-test -num-atoms 4000 -grid-side 64 -grid-spacing 0.5 -cutoff 12.0
```

### parboil_lbm
- **Package**: `amd/benchmarks/parboil/lbm`
- **Computes**: D3Q19 Lattice-Boltzmann fluid simulation on an N×N×N grid, fused collide+stream kernel with bounce-back boundaries, `-timesteps` iterations.
- **Flags**: `-grid int` (default 16), `-timesteps int` (default 4), `-tau float64` (default 0.7).
- **Profile**: Memory-bound — explicitly documented as bandwidth-bound in the Parboil technical report (19 distribution functions read/written per cell per step, modest FLOPs).
```bash
go run . -arch cdna3 -gpu mi300x -timing -trace-vis -report-all -metric-file-name lbm-test -grid 64 -timesteps 20 -tau 0.7
```

### parboil_sgemm
- **Package**: `amd/benchmarks/parboil/sgemm`
- **Computes**: C = alpha·A·B + beta·C dense N×N matmul via a shared-memory-tiled kernel (16×16 tiles).
- **Flags**: `-size int` (default 128).
- **Profile**: Compute-bound — classic tiled dense GEMM (O(n³) FLOPs, O(n²) data with LDS reuse). Parboil's own report notes some bandwidth sensitivity at small problem sizes/tile counts, but at the sizes used for research (≥512) this is compute-bound.
```bash
go run . -arch cdna3 -gpu mi300x -timing -trace-vis -report-all -metric-file-name sgemm-test -size 512
```

### parboil_stencil
- **Package**: `amd/benchmarks/parboil/stencil`
- **Computes**: 7-point 3D Jacobi stencil (`out = c0·in + c1·Σ(6 face neighbors)`) over an N×N×N grid, `-timesteps` iterations.
- **Flags**: `-size int` (default 32), `-timesteps int` (default 4).
- **Profile**: Memory-bound — explicitly documented as bandwidth-limited in the Parboil report; low FLOPs (7 reads, 1 FMA-ish combine) per output cell.
```bash
go run . -arch cdna3 -gpu mi300x -timing -trace-vis -report-all -metric-file-name pb-stencil-test -size 64 -timesteps 20
```

---

## 7. PolyBench samples

> `atax`/`bicg` ship both architectures; every other PolyBench port here is
> **CDNA3-only**. None of the PolyBench samples split data across multiple GPUs
> (each either restricts to `b.gpus[0]` or creates unused per-GPU queues).

| Kernel | What it computes | Flags | Arch | Profile |
|---|---|---|---|---|
| atax | `y = Aᵀ(Ax)` | `-x` (4096), `-y` (4096) | Both | Memory-bound (BLAS2, O(n²) data & ops, no reuse) |
| bicg | `s=Aᵀr`, `q=Ap` | `-x` (4096), `-y` (4096) | Both | Memory-bound (BLAS2) |
| polybench_2dconv | Fixed 3×3 stencil conv | `-size` (64) | CDNA3 | Memory-bound (tiny fixed filter) |
| polybench_2mm | `D=αAB+βD; E=αCD+βE` | `-size` (128) | CDNA3 | Compute-bound (2 chained GEMMs) |
| polybench_3dconv | N×N×N volume conv, cubic filter | `-size` (32), `-filter-size` (3) | CDNA3 | Memory-bound (small filter) |
| polybench_3mm | 3 chained GEMMs | `-size` (64, applies to all dims) | CDNA3 | Compute-bound |
| polybench_correlation | Mean/stddev/normalize + N×N correlation matmul | `-size` (64) | CDNA3 | Mixed — memory-bound elementwise passes + compute-bound matmul tail |
| polybench_fdtd2d | 2D FDTD EM field update | `-size` (16), `-tmax` (10) | CDNA3 | Memory-bound (stencil-style field updates) |
| polybench_gemm | `C=αAB+βC` tiled GEMM | `-size` (512) | CDNA3 | Compute-bound |
| polybench_gramschmidt | QR via Gram-Schmidt | `-m` (32), `-n` (32) | CDNA3 | Memory-bound (dot-product/norm-heavy, BLAS1/2-like) |
| polybench_jacobi2d | 2D 5-point Jacobi stencil | `-size` (64), `-tsteps` (10) | CDNA3 | Memory-bound (canonical memory-bound stencil) |
| polybench_mvt | Matrix-vector product + transpose | `-size` (128) | CDNA3 | Memory-bound (BLAS2) |
| polybench_syr2k | Symmetric rank-2k update | `-size` (64), `-inner-size` (64) | CDNA3 | Compute-bound (BLAS3-like tiled kernel) |

Example commands (CDNA3 ones need `-arch cdna3 -gpu mi300x`):
```bash
# atax / bicg (dual-arch)
go run . -timing -trace-vis -report-all -metric-file-name atax-test -x 8192 -y 8192

# any CDNA3-only PolyBench kernel, e.g. gemm
go run . -arch cdna3 -gpu mi300x -timing -trace-vis -report-all -metric-file-name pb-gemm-test -size 1024
```

---

## 8. NPB, Tango, Altis

### npb_ep
- **Package**: `amd/benchmarks/npb/ep`
- **Computes**: NAS "Embarrassingly Parallel" kernel — per-thread LCG + Box-Muller Gaussian-pair generation, binned into 10 annuli.
- **Flags**: `-size int` (default 4096) — number of Gaussian pairs.
- **Architecture**: CDNA3 only. **Unified memory**: Supported. **Multi-GPU**: Single-GPU only.
- **Profile**: Compute-bound by design — EP was specifically built by NASA to measure peak achievable FLOPs with essentially no memory bottleneck and no inter-thread communication (confirmed in NPB literature).
```bash
go run . -arch cdna3 -gpu mi300x -timing -trace-vis -report-all -metric-file-name npb-ep-test -size 1000000
```

### tango_binomial_options
- **Package**: `amd/benchmarks/tango/binomialoptions`
- **Computes**: American put-option pricing via a Cox-Ross-Rubinstein binomial tree, one workgroup per option, LDS-resident backward induction.
- **Flags**: `-options int` (default 8), `-steps int` (default 64, tree depth).
- **Architecture**: CDNA3 only. **Unified memory**: Supported. **Multi-GPU**: Single-GPU only.
- **Profile**: Compute-bound — heavy sequential FLOPs per option with almost all state kept in LDS/registers; minimal global memory traffic.
```bash
go run . -arch cdna3 -gpu mi300x -timing -trace-vis -report-all -metric-file-name binomial-test -options 512 -steps 128
```

### tango_blackscholes
- **Package**: `amd/benchmarks/tango/blackscholes`
- **Computes**: European call/put pricing via the closed-form Black-Scholes formula with a polynomial cumulative-normal approximation.
- **Flags**: `-size int` (default 4096) — number of options.
- **Architecture**: CDNA3 only. **Unified memory**: Supported. **Multi-GPU**: Single-GPU only.
- **Profile**: Compute-bound — transcendental-heavy (exp/log/polynomial erf approximation) per option, tiny memory footprint.
```bash
go run . -arch cdna3 -gpu mi300x -timing -trace-vis -report-all -metric-file-name blackscholes-test -size 65536
```

### altis_cfd
- **Package**: `amd/benchmarks/altis/cfd`
- **Computes**: Rusanov (local Lax-Friedrichs) finite-volume flux computation for compressible Euler equations over a synthetic unstructured mesh (`-size` cells, 4 neighbors each). Note: only the dominant `compute_flux_kernel` runs, not a full multi-stage CFD solve loop.
- **Flags**: `-size int` (default 256) — mesh cell count.
- **Architecture**: CDNA3 only. **Unified memory**: Supported. **Multi-GPU**: Single-GPU only.
- **Profile**: Compute-bound (mixed) — per-cell flux math (5 conserved variables × Roe/Rusanov flux terms) gives moderate-to-high arithmetic intensity; Altis characterizes CFD Solver as one of its more compute-intensive macro-benchmarks.
```bash
go run . -arch cdna3 -gpu mi300x -timing -trace-vis -report-all -metric-file-name altis-cfd-test -size 8192
```

### altis_raytracing
- **Package**: `amd/benchmarks/altis/raytracing`
- **Computes**: One primary ray per pixel, ray-sphere intersection against `-spheres` spheres, Phong shading of the nearest hit.
- **Flags**: `-width int` (default 64), `-height int` (default 64), `-spheres int` (default 16).
- **Architecture**: CDNA3 only. **Unified memory**: Supported. **Multi-GPU**: Single-GPU only.
- **Profile**: Compute-bound — classic ray-tracing workload; cost scales with sphere count and is dominated by intersection-test arithmetic, not memory traffic.
```bash
go run . -arch cdna3 -gpu mi300x -timing -trace-vis -report-all -metric-file-name raytracing-test -width 512 -height 512 -spheres 64
```

---

## 9. HeteroMark samples

### aes
- **Package**: `amd/benchmarks/heteromark/aes`
- **Computes**: AES encryption of a random byte array using a precomputed key schedule/S-box on the GPU.
- **Flags**: `-length int` (default 65536) — bytes to encrypt.
- **Architecture**: Both. **Unified memory**: Supported. **Multi-GPU**: Real split via `driver.Distribute`.
- **Profile**: Compute-bound — SubBytes/ShiftRows/MixColumns/AddRoundKey per 10-14 rounds is heavy compute relative to the 16 bytes touched per block.
```bash
go run . -gpus 1,2 -timing -trace-vis -report-all -metric-file-name aes-test -length 1048576
```

### fir
- **Package**: `amd/benchmarks/heteromark/fir`
- **Computes**: FIR filter, `NumTapsParam`-tap convolution over `Length` input samples.
- **Flags**: `-length int` (default 4096), `-taps int` (default 16).
- **Architecture**: Both. **Unified memory**: Supported. **Multi-GPU**: Real split via `driver.Distribute`.
- **Profile**: Mixed, scales with tap count — with the default 16 taps arithmetic intensity is modest (16 MACs per sample load window); increasing `-taps` shifts it toward compute-bound.
```bash
go run . -gpus 1,2 -timing -trace-vis -report-all -metric-file-name fir-test -length 65536 -taps 64
```

### kmeans
- **Package**: `amd/benchmarks/heteromark/kmeans`
- **Computes**: K-means clustering of `-points` feature vectors into `-clusters` clusters, up to `-max-iter` Lloyd iterations.
- **Flags**: `-points int` (default 1024), `-clusters int` (default 5), `-features int` (default 32), `-max-iter int` (default 5).
- **Architecture**: Both. **Unified memory**: Supported. **Multi-GPU**: Real split via `driver.Distribute`.
- **Profile**: Memory-bound — each point-to-centroid distance pass re-reads all features with comparatively little compute per byte, and centroids are re-read every iteration.
```bash
go run . -gpus 1,2 -timing -trace-vis -report-all -metric-file-name kmeans-test -points 65536 -clusters 16 -features 34 -max-iter 10
```

### pagerank
- **Package**: `amd/benchmarks/heteromark/pagerank`
- **Computes**: PageRank on a randomly generated sparse (CSR) graph, `-iterations` power-iteration steps.
- **Flags**: `-node int` (default 16), `-sparsity float64` (default 0.001), `-iterations int` (default 16).
- **Architecture**: Both. **Unified memory**: Supported.
- **Multi-GPU**: `SelectGPU` stores the list but no `Distribute()` exists — effectively single-GPU.
- **Profile**: Memory-bound — sparse, irregular graph access with very low FLOPs per edge; canonical memory/latency-bound graph analytics kernel.
```bash
go run . -gpus 1 -timing -trace-vis -report-all -metric-file-name pagerank-test -node 10000 -sparsity 0.001 -iterations 20
```

---

## 10. DNN samples

### conv2d (layer microbenchmark)
- **Package**: `amd/benchmarks/dnn/layer_benchmarks/conv2d`
- **Computes**: A single Conv2D layer forward pass (optionally + backward) via the shared `gputensor.GPUOperator` tensor engine.
- **Flags**: `-N` (1), `-C` (1), `-H` (28), `-W` (28), `-output-channel` (3), `-kernel-height` (3), `-kernel-width` (3), `-pad-x`/`-pad-y` (0), `-stride-x`/`-stride-y` (1), `-enable-backward bool` (false).
- **Architecture**: Both (underlying `gputensor` engine embeds both, and this sample does wire `-arch` through).
- **Unified memory**: **Not functionally wired** — `SetUnifiedMemory()` sets a field nothing reads; the operator always uses `AllocateMemory`.
- **Multi-GPU**: Panics above 1 GPU ("Conv2D benchmark can only run on a single GPU for now").
- **Profile**: Compute-bound — implemented via im2col+GEMM, so overall cost is GEMM-dominated once the (memory-bound) im2col step completes.
```bash
go run . -gpus 1 -arch cdna3 -gpu mi300x -timing -trace-vis -report-all -metric-file-name conv2d-test -N 8 -C 64 -H 56 -W 56 -output-channel 128 -kernel-height 3 -kernel-width 3
```

### im2col (layer microbenchmark)
- **Package**: `amd/benchmarks/dnn/layer_benchmarks/im2col`
- **Computes**: Standalone im2col transform (unfolds convolution windows of an NCHW tensor into columns).
- **Flags**: `-N` (1), `-C` (1), `-H` (28), `-W` (28), `-kernel-height` (3), `-kernel-width` (3), `-pad-x`/`-pad-y` (0), `-stride-x`/`-stride-y` (1), `-dilate-x`/`-dilate-y` (1).
- **Architecture**: Both. **Unified memory**: Not functionally wired (same as conv2d). **Multi-GPU**: Panics above 1 GPU.
- **Profile**: Memory-bound — pure data-rearrangement/copy with duplicated reads (each input element is read once per overlapping window); no meaningful arithmetic, well documented in the CNN-acceleration literature as a bandwidth-heavy step.
```bash
go run . -arch cdna3 -gpu mi300x -timing -trace-vis -report-all -metric-file-name im2col-test -N 8 -C 64 -H 56 -W 56 -kernel-height 3 -kernel-width 3
```

### relu (layer microbenchmark)
- **Package**: `amd/benchmarks/dnn/layer_benchmarks/relu`
- **Computes**: Standalone elementwise ReLU (`max(0,x)`) over a `-length`-element array.
- **Flags**: `-length int` (default 4096).
- **Architecture**: Both. **Unified memory**: Supported. **Multi-GPU**: Real split (per-GPU offset).
- **Profile**: Memory-bound — one compare+select per element, 1 read + 1 write; activation functions are the classic memory-bound DNN layer.
```bash
go run . -gpus 1,2 -timing -trace-vis -report-all -metric-file-name relu-test -length 4194304
```

### lenet (training benchmark)
- **Package**: `amd/benchmarks/dnn/training_benchmarks/lenet`
- **Computes**: Trains a LeNet-style CNN (Conv→ReLU→AvgPool ×2 + 3 FC layers) on MNIST with Adam + softmax cross-entropy, via `gputraining.DataParallelismMultiGPUTrainer`.
- **Flags**: `-epoch int` (1), `-max-batch-per-epoch int` (2), `-batch-size int` (32), `-enable-testing bool` (false), `-enable-verification bool` (false).
- **Architecture**: Engine supports both, but this sample's `main.go` never sets `.Arch` — **always runs GCN3** regardless of `-arch`.
- **Unified memory**: **Not supported — panics** ("unified memory is not supported by dnn workloads").
- **Multi-GPU**: True data-parallel training — separate network replica per GPU, parameters synced via `mccl` ring-broadcast.
- **Profile**: Compute-bound — GEMM-backed conv/FC layers dominate cycles even at LeNet's small scale.
```bash
go run . -gpus 1,2 -timing -trace-vis -report-all -metric-file-name lenet-test -epoch 3 -batch-size 64 -max-batch-per-epoch 20
```

### minerva (training benchmark)
- **Package**: `amd/benchmarks/dnn/training_benchmarks/minerva`
- **Computes**: Trains a fully-connected MLP (784→256→100→100→10, ReLU) on MNIST, same trainer/optimizer setup as lenet.
- **Flags**: `-epoch int` (1), `-max-batch-per-epoch int` (2), `-batch-size int` (32), `-enable-testing bool` (false), `-enable-verification bool` (false).
- **Architecture**: Always GCN3 in practice (same reason as lenet). **Unified memory**: Not supported (panics). **Multi-GPU**: True data-parallel (mccl).
- **Profile**: Mixed, memory-leaning at small batch sizes — pure FC/GEMV-like layers have lower arithmetic intensity than convolutions; shifts toward compute-bound as `-batch-size` grows (GEMV → GEMM).
```bash
go run . -gpus 1,2 -timing -trace-vis -report-all -metric-file-name minerva-test -epoch 3 -batch-size 128 -max-batch-per-epoch 20
```

### vgg16 (training benchmark)
- **Package**: `amd/benchmarks/dnn/training_benchmarks/vgg16`
- **Computes**: Trains a VGG16-style deep CNN (13 conv layers + ReLU + 5 max-pool stages + 2 FC layers) on ImageNet-like data.
- **Flags**: `-epoch int` (1), `-max-batch-per-epoch int` (2), `-batch-size int` (8), `-enable-testing bool` (false), `-enable-verification bool` (false).
- **Architecture**: Always GCN3 in practice (same reason as lenet). **Unified memory**: Not supported (panics). **Multi-GPU**: True data-parallel (mccl).
- **Profile**: Compute-bound — deep, conv-heavy CNN training is the textbook compute-bound DNN workload.
```bash
go run . -gpus 1,2 -timing -trace-vis -report-all -metric-file-name vgg16-test -epoch 1 -batch-size 16 -max-batch-per-epoch 5
```

### xor (training benchmark)
- **Package**: `amd/benchmarks/dnn/training_benchmarks/xor`
- **Computes**: Trains a tiny 2→4→4→2 FC network to learn XOR; 50 epochs, batch size 4 — both hardcoded, not exposed as flags.
- **Flags**: none (only common runner flags).
- **Architecture**: GCN3 only (no `.Arch` ever set). **Unified memory**: Not supported (panics). **Multi-GPU**: Panics above 1 GPU.
- **Profile**: Overhead-bound — the problem is so tiny that kernel-launch/dispatch overhead dominates over both compute and memory traffic; useful as a smoke test, not a performance workload.
```bash
go run . -timing -trace-vis -report-all -metric-file-name xor-test
```

---

## 11. Microbenchmarks (`microbench/*`)

> All CDNA3-only, all explicitly single-GPU by design (they measure a single
> device's hardware characteristics, not scaling).

| Sample | Measures | Flags | Profile |
|---|---|---|---|
| cache_latency | Dependent pointer-chase load latency | `-array-bytes` (16384), `-cacheline-bytes` (64), `-measure-laps` (4), `-num-accesses` (0=auto), `-seed` (42) | **Latency-bound** — serialized dependent loads, no bandwidth exploited |
| empty_kernel | Kernel launch/dispatch overhead | `-num-blocks` (1), `-block-size` (256) | **Overhead-bound** — no compute or memory work at all |
| fp16_throughput | Packed half2 FMA throughput | `-fmas-per-thread` (64), `-num-blocks` (2), `-threads-per-block` (64) | **Compute-bound** by design (register-resident FMA chain) |
| fp32_throughput | FP32 FMA throughput | `-num-blocks` (4), `-fmas` (256), `-threads-per-block` (256) | **Compute-bound** by design |
| fp64_throughput | FP64 FMA throughput | `-num-blocks` (4), `-fmas-per-thread` (16), `-threads-per-block` (256) | **Compute-bound** by design |
| int32_throughput | INT32 multiply-add throughput | `-mads` (4096), `-blocks` (16), `-threads-per-block` (256) | **Compute-bound** by design |
| memory_bandwidth | Device-to-device copy bandwidth | `-size` (4096, float32 elements) | **Memory-bound** by design |
| mixbench | Configurable compute:memory ratio sweep | `-size` (4096), `-fmas` (16) | **Configurable** — 1 global load + `-fmas` FMAs + 1 store; sweeping `-fmas` moves it from memory-bound to compute-bound (this is the tool's whole purpose, cf. the original Mixbench roofline paper) |
| shared_mem_bandwidth | LDS (shared memory) bandwidth | `-num-blocks` (16), `-inner-iters` (8), `-block-size` (256), `-access-pattern` (no_conflict\|conflict\|both) | **Memory-bound** (on-chip LDS, not DRAM) |
| shared_mem_latency | LDS access latency | same flags as above | **Latency-bound** (on-chip LDS) |

```bash
# compute throughput example
go run . -arch cdna3 -gpu mi300x -timing -report-all -metric-file-name fp32-test -num-blocks 64 -fmas 1024 -threads-per-block 256
cd ../mixbench && go run . -arch cdna3 -gpu mi300x -timing -report-all -metric-file-name mixbench-sweep -size 1048576 -fmas 128
```

---

## 12. BabelStream & misc

### babelstream
- **Package**: `amd/benchmarks/babelstream`
- **Computes**: The classic STREAM kernels — `copy` (c=a), `scale` (b=s·c), `add` (c=a+b), `triad` (a=b+s·c) — run in sequence over length-N float32 arrays.
- **Flags**: `-size int` (default 4096), `-scalar float64` (default 2.0).
- **Architecture**: CDNA3 only. **Unified memory**: Supported. **Multi-GPU**: Single-GPU only.
- **Profile**: Memory-bound — this is *the* reference memory-bandwidth benchmark; it exists specifically to measure attainable HBM/DRAM bandwidth, with essentially zero arithmetic intensity.
```bash
go run . -arch cdna3 -gpu mi300x -timing -report-all -metric-file-name babelstream-test -size 16777216 -scalar 2.0
```

### memcopy
- **Package**: none — self-contained in `amd/samples/memcopy/main.go`, no kernel/HSACO at all.
- **Computes**: Pure H2D then D2H memory-copy round trip of a fixed 1 MiB (`1048576` byte) buffer; verifies round-tripped bytes match.
- **Flags**: none (buffer size is hardcoded, not exposed as a flag).
- **Architecture**: N/A (no ISA involved). **Unified memory**: Supported. **Multi-GPU**: Panics above 1 GPU.
- **Profile**: Memory-bound by definition — a raw PCIe/interconnect copy-bandwidth probe.
```bash
go run . -gpus 1 -timing -report-all -metric-file-name memcopy-test
```

### concurrentkernel (demo)
- **Package**: composes `heteromark/fir` (length 10240) + `amdappsdk/bitonicsort` (length 64), both hardcoded and both pinned to **GPU 1**.
- **Purpose**: Demonstrates two independent kernels running concurrently on the *same* device (scheduler/concurrency behavior, not a data-parallel scaling test).
- **Flags**: none. **Architecture**: Both engines support it, but never wired — runs GCN3. **Unified memory**: Not enabled in this demo.
- **Profile**: Composite of FIR (mixed) + bitonic sort (memory-bound); the point of the sample is concurrency, not a single roofline characterization.
```bash
go run . -timing -trace-vis -report-all -metric-file-name concurrentkernel-test
```

### concurrentworkload (demo)
- **Package**: composes `heteromark/fir` (length 10240, pinned to GPUs {1,2}) + `amdappsdk/bitonicsort` (length 64, pinned to GPU {3}).
- **Purpose**: Demonstrates two independent workloads running concurrently across *different* devices — FIR data-split across GPUs 1-2 while bitonic sort runs standalone on GPU 3, all at once. Needs at least 3 simulated GPUs.
- **Flags**: none.
- **Profile**: Composite (same components as concurrentkernel); the point is multi-device concurrency, not a single roofline number.
```bash
go run . -timing -trace-vis -report-all -metric-file-name concurrentworkload-test
```

---

## 13. Suggested starter commands by research goal

```bash
# Pure multi-GPU data-parallel scaling study (real split confirmed in code):
cd amd/samples/matrixmultiplication && go run . -gpus 1,2,3,4 -timing -trace-vis -report-all -metric-file-name mm-4gpu -x 1024 -y 1024 -z 1024

# Memory-bandwidth-bound stress test:
cd amd/samples/bitonicsort && go run . -gpus 1,2 -timing -trace-vis -report-all -metric-file-name bitonic-2gpu -length 1048576

# Compute-bound stress test:
cd amd/samples/aes && go run . -gpus 1,2 -timing -trace-vis -report-all -metric-file-name aes-2gpu -length 4194304

# Unified-memory vs. explicit-copy comparison (same workload, two runs):
cd amd/samples/vectoradd
go run . -gpus 1,2 -timing -report-all -metric-file-name vecadd-explicit -width 4096 -height 4096
go run . -gpus 1,2 -timing -use-unified-memory -report-all -metric-file-name vecadd-unified -width 4096 -height 4096

# Roofline sweep (single tool built for exactly this purpose):
cd amd/samples/mixbench
for f in 1 4 16 64 256 1024; do
  go run . -arch cdna3 -gpu mi300x -timing -report-all -metric-file-name mixbench-fmas$f -size 1048576 -fmas $f
done

# True data-parallel DNN training across GPUs (mccl collective, not a manual split):
cd amd/samples/lenet && go run . -gpus 1,2 -timing -trace-vis -report-all -metric-file-name lenet-2gpu -epoch 2 -batch-size 64 -max-batch-per-epoch 10
```

---

## Sources

Architecture/flag/unified-memory/multi-GPU facts: read directly from this repository's
Go source (`amd/samples/**/*.go`, `amd/benchmarks/**/*.go`,
`amd/samples/runner/{flag,runner}.go`, `amd/samples/runner/timingconfig/{r9nano,mi300x}/builder.go`).

Compute/memory-bound characterizations, cross-checked against:
- [Rodinia: A Benchmark Suite for Heterogeneous Computing](https://www.cs.virginia.edu/~skadron/Papers/rodinia_iiswc10.pdf) (Che et al., IISWC 2009/2010) — SRAD/HotSpot compute-intensive; NW/BFS/KMeans memory-bandwidth-limited.
- [Altis: Modernizing GPGPU Benchmarks](https://arxiv.org/pdf/1906.10347) (Hu et al.) — roofline/arithmetic-intensity methodology, CFD/Raytracing macro-benchmark characterization.
- [The Scalable HeterOgeneous Computing (SHOC) Benchmark Suite](https://www.researchgate.net/publication/220938804_The_Scalable_HeterOgeneous_Computing_SHOC_benchmark_suite) (Danalis et al.) — FFT/SpMV/Stencil2D/BFS Level-1 kernel design intent.
- [Parboil: A Revised Benchmark Suite](http://impact.crhc.illinois.edu/shared/report/impact-12-01.parboil.pdf) (UIUC IMPACT report) — LBM and Stencil documented as bandwidth-bound; CUTCP as contention/locality-bound compute kernel.
- [Hetero-Mark, A Benchmark Suite for CPU-GPU Collaborative Computing](https://www.researchgate.net/publication/309206460_Hetero-Mark_A_Benchmark_Suite_for_CPU-GPU_Collaborative_Computing) (Sun et al., IISWC 2016) — FIR/AES/KMeans/PageRank benchmark design.
- NAS Parallel Benchmarks EP kernel documentation — EP explicitly designed as a compute-only, near-zero-memory-traffic FLOPS probe.
- [BabelStream / GPU-STREAM](https://github.com/UoB-HPC/BabelStream) — canonical memory-bandwidth benchmark methodology (copy/scale/add/triad).
- CNN im2col+GEMM literature (im2col memory-overhead/duplication cost, GEMM-dominated conv layer compute cost) — general acceleration/compiler literature on lowering convolution to GEMM.
- PolyBench/GPU characterization literature — PolyBench kernels generally noted as having a higher memory-to-compute ratio than compute-oriented suites like Rodinia, consistent with the per-kernel BLAS1/2 (memory-bound) vs. BLAS3 (compute-bound) split used above.
