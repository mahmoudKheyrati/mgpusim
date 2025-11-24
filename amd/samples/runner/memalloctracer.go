package runner

import (
	"fmt"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/sarchlab/akita/v4/sim"
)

// deviceMemoryState defines the interface for querying device memory state
type deviceMemoryState interface {
	GetStorageSize() uint64
	GetFreeMemorySize() uint64
}

// deviceInfo holds information about a device
type deviceInfo struct {
	ID       int
	Type     int
	MemState deviceMemoryState
}

// memAllocTracer tracks memory allocation over time for plotting
type memAllocTracer struct {
	sync.Mutex
	sim.TimeTeller

	devices        []*deviceInfo
	samplingPeriod sim.VTimeInSec
	realTimePeriod time.Duration
	lastSampleTime sim.VTimeInSec

	// Time-series data
	samples []memAllocSample

	// Output file
	outputFile *os.File

	// Background sampling control
	stopChan chan bool
	doneChan chan bool
	running  bool
}

// memAllocSample represents a snapshot of memory allocation at a point in time
type memAllocSample struct {
	time           sim.VTimeInSec
	deviceID       int
	allocatedBytes uint64
	totalBytes     uint64
}

func newMemAllocTracer(
	timeTeller sim.TimeTeller,
	memAllocator interface{},
	devicesRaw interface{},
	samplingPeriod float64,
	outputFileName string,
) *memAllocTracer {
	file, err := os.Create(outputFileName)
	if err != nil {
		panic(fmt.Sprintf("Failed to create memory allocation trace file: %v", err))
	}

	// Write CSV header
	_, err = file.WriteString("Time(s),DeviceID,DeviceName,AllocatedBytes,TotalBytes,AllocatedMB,TotalMB,UtilizationPercent\n")
	if err != nil {
		panic(fmt.Sprintf("Failed to write CSV header: %v", err))
	}

	// Convert devices to our local type using reflection/interface{}
	devices := convertDevices(devicesRaw)

	return &memAllocTracer{
		TimeTeller:     timeTeller,
		devices:        devices,
		samplingPeriod: sim.VTimeInSec(samplingPeriod),
		realTimePeriod: time.Duration(float64(time.Millisecond) * 100), // Sample every 100ms real time
		lastSampleTime: 0,
		samples:        make([]memAllocSample, 0),
		outputFile:     file,
		stopChan:       make(chan bool),
		doneChan:       make(chan bool),
		running:        false,
	}
}

// convertDevices converts the internal Device slice to our local deviceInfo slice
// using reflection to access exported fields without importing the internal package
func convertDevices(devicesRaw interface{}) []*deviceInfo {
	// Use reflection to access the slice
	rv := reflect.ValueOf(devicesRaw)

	// Check if it's a slice
	if rv.Kind() != reflect.Slice {
		fmt.Printf("Warning: devicesRaw is not a slice, got %v\n", rv.Kind())
		return nil
	}

	devices := make([]*deviceInfo, rv.Len())

	for i := 0; i < rv.Len(); i++ {
		elem := rv.Index(i)

		// Handle pointer to struct
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}

		if elem.Kind() != reflect.Struct {
			fmt.Printf("Warning: device element %d is not a struct, got %v\n", i, elem.Kind())
			continue
		}

		// Extract exported fields by name
		// The internal.Device struct has: ID int, Type int, MemState DeviceMemoryState
		idField := elem.FieldByName("ID")
		typeField := elem.FieldByName("Type")
		memStateField := elem.FieldByName("MemState")

		if !idField.IsValid() || !typeField.IsValid() || !memStateField.IsValid() {
			fmt.Printf("Warning: device element %d missing required fields\n", i)
			continue
		}

		// Extract values
		deviceID := int(idField.Int())
		deviceType := int(typeField.Int())

		// MemState should implement our deviceMemoryState interface
		memState, ok := memStateField.Interface().(deviceMemoryState)
		if !ok {
			fmt.Printf("Warning: device %d MemState does not implement deviceMemoryState interface\n", i)
			continue
		}

		devices[i] = &deviceInfo{
			ID:       deviceID,
			Type:     deviceType,
			MemState: memState,
		}
	}

	return devices
}

// Sample takes a snapshot of current memory allocation
func (t *memAllocTracer) Sample() {
	t.Lock()
	defer t.Unlock()

	currentTime := sim.VTimeInSec(t.TimeTeller.CurrentTime())

	// Skip if not enough time has passed
	if currentTime-t.lastSampleTime < t.samplingPeriod {
		return
	}

	t.lastSampleTime = currentTime

	// Sample each device
	for _, device := range t.devices {
		totalBytes := device.MemState.GetStorageSize()
		freeBytes := device.MemState.GetFreeMemorySize()
		allocatedBytes := totalBytes - freeBytes

		sample := memAllocSample{
			time:           currentTime,
			deviceID:       device.ID,
			allocatedBytes: allocatedBytes,
			totalBytes:     totalBytes,
		}

		t.samples = append(t.samples, sample)

		// Write to file immediately for real-time monitoring
		allocatedMB := float64(allocatedBytes) / (1024 * 1024)
		totalMB := float64(totalBytes) / (1024 * 1024)
		utilization := 0.0
		if totalBytes > 0 {
			utilization = float64(allocatedBytes) * 100.0 / float64(totalBytes)
		}

		// DeviceTypeCPU is 1 in internal package
		deviceName := fmt.Sprintf("Device%d", device.ID)
		if device.Type == 1 {
			deviceName = "CPU"
		} else {
			deviceName = fmt.Sprintf("GPU%d", device.ID)
		}

		line := fmt.Sprintf("%.9f,%d,%s,%d,%d,%.2f,%.2f,%.2f\n",
			currentTime, device.ID, deviceName,
			allocatedBytes, totalBytes,
			allocatedMB, totalMB, utilization)

		_, err := t.outputFile.WriteString(line)
		if err != nil {
			fmt.Printf("Warning: Failed to write memory allocation sample: %v\n", err)
		}
	}
}

// Start begins background sampling
func (t *memAllocTracer) Start() {
	t.Lock()
	if t.running {
		t.Unlock()
		return
	}
	t.running = true
	t.Unlock()

	go t.backgroundSampler()
}

// Stop stops background sampling
func (t *memAllocTracer) Stop() {
	t.Lock()
	if !t.running {
		t.Unlock()
		return
	}
	t.Unlock()

	t.stopChan <- true
	<-t.doneChan
}

// backgroundSampler runs in a goroutine and samples periodically
func (t *memAllocTracer) backgroundSampler() {
	ticker := time.NewTicker(t.realTimePeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.Sample()
		case <-t.stopChan:
			// Final sample
			t.Sample()
			t.Lock()
			t.running = false
			t.Unlock()
			t.doneChan <- true
			return
		}
	}
}

// Close closes the output file
func (t *memAllocTracer) Close() {
	t.Stop()

	t.Lock()
	defer t.Unlock()

	if t.outputFile != nil {
		t.outputFile.Close()
	}
}

// GetSamples returns all collected samples
func (t *memAllocTracer) GetSamples() []memAllocSample {
	t.Lock()
	defer t.Unlock()

	return t.samples
}

// GetSampleCount returns the number of samples collected
func (t *memAllocTracer) GetSampleCount() int {
	t.Lock()
	defer t.Unlock()

	return len(t.samples)
}
