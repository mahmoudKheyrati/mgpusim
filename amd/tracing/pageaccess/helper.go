package pageaccess

import (
	"github.com/sarchlab/akita/v4/mem/vm"
	"github.com/sarchlab/akita/v4/sim"
)

// TrackMemoryAccess is a helper function to track page accesses from memory operations.
// This can be called from various components (CUs, caches, etc.) to record page accesses.
//
// Parameters:
//   - virtualAddr: The virtual address being accessed
//   - physicalAddr: The physical address being accessed
//   - accessType: "read" or "write"
//   - gpuID: The GPU ID (e.g., "GPU1", "GPU2")
//   - cuID: The Compute Unit ID (e.g., "GPU1.ShaderArray0.CU0")
//   - pid: The process ID
//
// Example usage from a Compute Unit:
//
//	if pageaccess.GlobalPageAccessTracer != nil {
//	    pageaccess.TrackMemoryAccess(
//	        virtualAddr,
//	        physicalAddr,
//	        "read",
//	        "GPU1",
//	        cu.Name(),
//	        wave.PID(),
//	    )
//	}
func TrackMemoryAccess(
	virtualAddr uint64,
	physicalAddr uint64,
	accessType string,
	gpuID string,
	cuID string,
	pid vm.PID,
) {
	if GlobalPageAccessTracer != nil {
		GlobalPageAccessTracer.RecordPageAccess(
			virtualAddr,
			physicalAddr,
			accessType,
			gpuID,
			cuID,
			pid,
		)
	}
}

// TrackPageMigration is a helper function to manually track page migrations.
// This is typically called automatically by the PMC, but can be used for custom migration tracking.
func TrackPageMigration(
	pageAddr uint64,
	pageSize uint64,
	sourceGPU string,
	destGPU string,
	sourcePhysAddr uint64,
	destPhysAddr uint64,
	migrationDuration sim.VTimeInSec,
	dataTransferSize uint64,
) {
	if GlobalPageAccessTracer != nil {
		GlobalPageAccessTracer.RecordPageMigration(
			pageAddr,
			pageSize,
			sourceGPU,
			destGPU,
			sourcePhysAddr,
			destPhysAddr,
			migrationDuration,
			dataTransferSize,
		)
	}
}

// TrackPageReplication is a helper function to track page replications.
func TrackPageReplication(
	pageAddr uint64,
	pageSize uint64,
	sourceGPU string,
	destGPU string,
	replicaPhysAddr uint64,
) {
	if GlobalPageAccessTracer != nil {
		GlobalPageAccessTracer.RecordPageReplication(
			pageAddr,
			pageSize,
			sourceGPU,
			destGPU,
			replicaPhysAddr,
		)
	}
}
