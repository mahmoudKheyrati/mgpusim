package runner

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/sarchlab/akita/v4/sim"
	"github.com/sarchlab/mgpusim/v4/amd/driver/internal"
)

// memAllocTracer tracks memory allocation over time for plotting
type memAllocTracer struct {
	sync.Mutex
	sim.TimeTeller

	memAllocator   internal.MemoryAllocator
	devices        []*internal.Device
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
	memAllocator internal.MemoryAllocator,
	devices []*internal.Device,
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

	return &memAllocTracer{
		TimeTeller:     timeTeller,
		memAllocator:   memAllocator,
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

		deviceName := fmt.Sprintf("Device%d", device.ID)
		if device.Type == internal.DeviceTypeCPU {
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
