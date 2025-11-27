package runner

import (
	"sync"

	"github.com/sarchlab/akita/v4/datarecording"
	"github.com/sarchlab/akita/v4/sim"
)

// GlobalPageAccessTracer is a global instance that can be accessed by other components
var GlobalPageAccessTracer *PageAccessTracer

const (
	pageAccessTableName        = "page_accesses"
	pageSharingTableName       = "page_sharing_summary"
	pageMigrationTableName     = "page_migrations"
	pageReplicationTableName   = "page_replications"
)

// PageAccessRecord represents a single page access event
type PageAccessRecord struct {
	Time         sim.VTimeInSec
	PageAddr     uint64
	AccessType   string // "read", "write"
	GPUID        string
	CUID         string
	PID          sim.PID
	VirtualAddr  uint64
	PhysicalAddr uint64
}

// PageSharingSummary represents sharing statistics for a page
type PageSharingSummary struct {
	PageAddr      uint64
	AccessCount   int
	GPUList       string // Comma-separated list of GPU IDs
	NumGPUs       int
	NumReads      int
	NumWrites     int
	FirstAccess   sim.VTimeInSec
	LastAccess    sim.VTimeInSec
}

// PageMigrationRecord represents a page migration event
type PageMigrationRecord struct {
	Time              sim.VTimeInSec
	PageAddr          uint64
	PageSize          uint64
	SourceGPU         string
	DestGPU           string
	SourcePhysAddr    uint64
	DestPhysAddr      uint64
	MigrationDuration sim.VTimeInSec
	DataTransferSize  uint64
}

// PageReplicationRecord represents a page replication event
type PageReplicationRecord struct {
	Time           sim.VTimeInSec
	PageAddr       uint64
	PageSize       uint64
	SourceGPU      string
	DestGPU        string
	ReplicaPhysAddr uint64
}

// PageAccessTracer tracks page accesses, migrations, and sharing patterns
type PageAccessTracer struct {
	sync.Mutex

	dataRecorder datarecording.DataRecorder
	timeTeller   sim.TimeTeller
	log2PageSize uint64

	// Page access tracking
	pageAccessEnabled bool

	// Page sharing summary
	pageSharingMap map[uint64]*pageSharingInfo

	// Migration tracking
	migrationEnabled bool
	migrationCount   int

	// Replication tracking
	replicationEnabled bool
	replicationCount   int
}

// Internal structure to track page sharing
type pageSharingInfo struct {
	pageAddr     uint64
	gpuSet       map[string]bool
	accessCount  int
	readCount    int
	writeCount   int
	firstAccess  sim.VTimeInSec
	lastAccess   sim.VTimeInSec
}

// NewPageAccessTracer creates a new page access tracer
func NewPageAccessTracer(
	dataRecorder datarecording.DataRecorder,
	timeTeller sim.TimeTeller,
	log2PageSize uint64,
) *PageAccessTracer {
	tracer := &PageAccessTracer{
		dataRecorder:       dataRecorder,
		timeTeller:         timeTeller,
		log2PageSize:       log2PageSize,
		pageAccessEnabled:  true,
		pageSharingMap:     make(map[uint64]*pageSharingInfo),
		migrationEnabled:   true,
		replicationEnabled: true,
	}

	// Create database tables
	tracer.createTables()

	return tracer
}

// createTables creates the database tables for storing page access data
func (t *PageAccessTracer) createTables() {
	t.dataRecorder.CreateTable(pageAccessTableName, PageAccessRecord{})
	t.dataRecorder.CreateTable(pageSharingTableName, PageSharingSummary{})
	t.dataRecorder.CreateTable(pageMigrationTableName, PageMigrationRecord{})
	t.dataRecorder.CreateTable(pageReplicationTableName, PageReplicationRecord{})
}

// GetPageAddress extracts the page address from a physical or virtual address
func (t *PageAccessTracer) GetPageAddress(addr uint64) uint64 {
	pageSize := uint64(1) << t.log2PageSize
	return (addr / pageSize) * pageSize
}

// RecordPageAccess records a page access event
func (t *PageAccessTracer) RecordPageAccess(
	virtualAddr uint64,
	physicalAddr uint64,
	accessType string,
	gpuID string,
	cuID string,
	pid sim.PID,
) {
	if !t.pageAccessEnabled {
		return
	}

	t.Lock()
	defer t.Unlock()

	pageAddr := t.GetPageAddress(physicalAddr)
	currentTime := t.timeTeller.CurrentTime()

	// Record individual access
	record := PageAccessRecord{
		Time:         currentTime,
		PageAddr:     pageAddr,
		AccessType:   accessType,
		GPUID:        gpuID,
		CUID:         cuID,
		PID:          pid,
		VirtualAddr:  virtualAddr,
		PhysicalAddr: physicalAddr,
	}
	t.dataRecorder.InsertData(pageAccessTableName, record)

	// Update sharing summary
	t.updatePageSharing(pageAddr, gpuID, accessType, currentTime)
}

// updatePageSharing updates the page sharing information
func (t *PageAccessTracer) updatePageSharing(
	pageAddr uint64,
	gpuID string,
	accessType string,
	currentTime sim.VTimeInSec,
) {
	info, exists := t.pageSharingMap[pageAddr]
	if !exists {
		info = &pageSharingInfo{
			pageAddr:    pageAddr,
			gpuSet:      make(map[string]bool),
			firstAccess: currentTime,
		}
		t.pageSharingMap[pageAddr] = info
	}

	info.gpuSet[gpuID] = true
	info.accessCount++
	info.lastAccess = currentTime

	if accessType == "read" {
		info.readCount++
	} else if accessType == "write" {
		info.writeCount++
	}
}

// RecordPageMigration records a page migration event
func (t *PageAccessTracer) RecordPageMigration(
	pageAddr uint64,
	pageSize uint64,
	sourceGPU string,
	destGPU string,
	sourcePhysAddr uint64,
	destPhysAddr uint64,
	migrationDuration sim.VTimeInSec,
	dataTransferSize uint64,
) {
	if !t.migrationEnabled {
		return
	}

	t.Lock()
	defer t.Unlock()

	currentTime := t.timeTeller.CurrentTime()

	record := PageMigrationRecord{
		Time:              currentTime,
		PageAddr:          pageAddr,
		PageSize:          pageSize,
		SourceGPU:         sourceGPU,
		DestGPU:           destGPU,
		SourcePhysAddr:    sourcePhysAddr,
		DestPhysAddr:      destPhysAddr,
		MigrationDuration: migrationDuration,
		DataTransferSize:  dataTransferSize,
	}

	t.dataRecorder.InsertData(pageMigrationTableName, record)
	t.migrationCount++
}

// RecordPageReplication records a page replication event
func (t *PageAccessTracer) RecordPageReplication(
	pageAddr uint64,
	pageSize uint64,
	sourceGPU string,
	destGPU string,
	replicaPhysAddr uint64,
) {
	if !t.replicationEnabled {
		return
	}

	t.Lock()
	defer t.Unlock()

	currentTime := t.timeTeller.CurrentTime()

	record := PageReplicationRecord{
		Time:            currentTime,
		PageAddr:        pageAddr,
		PageSize:        pageSize,
		SourceGPU:       sourceGPU,
		DestGPU:         destGPU,
		ReplicaPhysAddr: replicaPhysAddr,
	}

	t.dataRecorder.InsertData(pageReplicationTableName, record)
	t.replicationCount++
}

// Finalize writes the page sharing summary to the database
func (t *PageAccessTracer) Finalize() {
	t.Lock()
	defer t.Unlock()

	for _, info := range t.pageSharingMap {
		gpuList := ""
		numGPUs := len(info.gpuSet)
		count := 0
		for gpuID := range info.gpuSet {
			if count > 0 {
				gpuList += ","
			}
			gpuList += gpuID
			count++
		}

		summary := PageSharingSummary{
			PageAddr:    info.pageAddr,
			AccessCount: info.accessCount,
			GPUList:     gpuList,
			NumGPUs:     numGPUs,
			NumReads:    info.readCount,
			NumWrites:   info.writeCount,
			FirstAccess: info.firstAccess,
			LastAccess:  info.lastAccess,
		}

		t.dataRecorder.InsertData(pageSharingTableName, summary)
	}
}

// GetMigrationCount returns the total number of page migrations
func (t *PageAccessTracer) GetMigrationCount() int {
	t.Lock()
	defer t.Unlock()
	return t.migrationCount
}

// GetReplicationCount returns the total number of page replications
func (t *PageAccessTracer) GetReplicationCount() int {
	t.Lock()
	defer t.Unlock()
	return t.replicationCount
}

// GetTotalPagesAccessed returns the total number of unique pages accessed
func (t *PageAccessTracer) GetTotalPagesAccessed() int {
	t.Lock()
	defer t.Unlock()
	return len(t.pageSharingMap)
}

// SetPageAccessEnabled enables or disables page access tracking
func (t *PageAccessTracer) SetPageAccessEnabled(enabled bool) {
	t.Lock()
	defer t.Unlock()
	t.pageAccessEnabled = enabled
}

// SetMigrationEnabled enables or disables migration tracking
func (t *PageAccessTracer) SetMigrationEnabled(enabled bool) {
	t.Lock()
	defer t.Unlock()
	t.migrationEnabled = enabled
}

// SetReplicationEnabled enables or disables replication tracking
func (t *PageAccessTracer) SetReplicationEnabled(enabled bool) {
	t.Lock()
	defer t.Unlock()
	t.replicationEnabled = enabled
}
