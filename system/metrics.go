package system

import (
	"sync"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	gnet "github.com/shirou/gopsutil/net"
)

// NodeMetrics contains system resource usage information
type NodeMetrics struct {
	MemoryUsed  uint64  `json:"memory_used"`
	MemoryTotal uint64  `json:"memory_total"`
	SwapUsed    uint64  `json:"swap_used"`
	SwapTotal   uint64  `json:"swap_total"`

	DiskUsed      uint64  `json:"disk_used"`
	DiskTotal     uint64  `json:"disk_total"`
	DiskRead      uint64  `json:"disk_read"`
	DiskWrite     uint64  `json:"disk_write"`
	DiskReadRate  float64 `json:"disk_read_rate"`
	DiskWriteRate float64 `json:"disk_write_rate"`

	NetIn      uint64  `json:"net_in"`
	NetOut     uint64  `json:"net_out"`
	NetInRate  float64 `json:"net_in_rate"`
	NetOutRate float64 `json:"net_out_rate"`

	CPUUsed    float64 `json:"cpu_used"`
	CPUThreads int     `json:"cpu_threads"`
	CPUModel   string  `json:"cpu_model"`
}

type metricHistory struct {
	timestamps []time.Time
	diskReads  []uint64
	diskWrites []uint64
	netIns     []uint64
	netOuts    []uint64
	maxEntries int
	mu         sync.Mutex
}

const (
	// MB represents the number of bytes in a megabyte
	MB = 1024 * 1024
	
	// MinElapsedSeconds is the minimum elapsed time to avoid division by zero
	MinElapsedSeconds = 0.1
	
	// MinRateValue is the minimum rate value to ensure visibility of small changes
	MinRateValue = 0.01
	
	// DefaultHistorySize is the default number of history entries to maintain
	DefaultHistorySize = 5
)

var (
	history = &metricHistory{
		maxEntries: DefaultHistorySize,
	}
	isFirstRun = true
	initOnce   sync.Once
)

// add appends a new entry to the metric history
func (h *metricHistory) add(timestamp time.Time, diskRead, diskWrite, netIn, netOut uint64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.timestamps = append(h.timestamps, timestamp)
	h.diskReads = append(h.diskReads, diskRead)
	h.diskWrites = append(h.diskWrites, diskWrite)
	h.netIns = append(h.netIns, netIn)
	h.netOuts = append(h.netOuts, netOut)

	if len(h.timestamps) > h.maxEntries {
		h.timestamps = h.timestamps[1:]
		h.diskReads = h.diskReads[1:]
		h.diskWrites = h.diskWrites[1:]
		h.netIns = h.netIns[1:]
		h.netOuts = h.netOuts[1:]
	}
}

// calculateRates determines the rate of change for various metrics
func (h *metricHistory) calculateRates() (diskReadRate, diskWriteRate, netInRate, netOutRate float64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.timestamps) < 2 {
		return 0, 0, 0, 0
	}

	oldestIdx := 0
	newestIdx := len(h.timestamps) - 1

	elapsedSeconds := h.timestamps[newestIdx].Sub(h.timestamps[oldestIdx]).Seconds()
	if elapsedSeconds <= MinElapsedSeconds {
		elapsedSeconds = MinElapsedSeconds
	}

	diskReadDiff := float64(h.diskReads[newestIdx] - h.diskReads[oldestIdx])
	diskWriteDiff := float64(h.diskWrites[newestIdx] - h.diskWrites[oldestIdx])
	netInDiff := float64(h.netIns[newestIdx] - h.netIns[oldestIdx])
	netOutDiff := float64(h.netOuts[newestIdx] - h.netOuts[oldestIdx])

	// Handle counter resets
	if diskReadDiff < 0 {
		diskReadDiff = float64(h.diskReads[newestIdx])
	}
	if diskWriteDiff < 0 {
		diskWriteDiff = float64(h.diskWrites[newestIdx])
	}
	if netInDiff < 0 {
		netInDiff = float64(h.netIns[newestIdx])
	}
	if netOutDiff < 0 {
		netOutDiff = float64(h.netOuts[newestIdx])
	}

	// Calculate rates
	diskReadRate = diskReadDiff / elapsedSeconds
	diskWriteRate = diskWriteDiff / elapsedSeconds
	netInRate = netInDiff / elapsedSeconds
	netOutRate = netOutDiff / elapsedSeconds

	// Ensure small non-zero rates are visible
	applyMinimumRate(&diskReadRate, diskReadDiff)
	applyMinimumRate(&diskWriteRate, diskWriteDiff)
	applyMinimumRate(&netInRate, netInDiff)
	applyMinimumRate(&netOutRate, netOutDiff)

	return
}

// applyMinimumRate ensures small but non-zero rates are set to a minimum visible value
func applyMinimumRate(rate *float64, diff float64) {
	if diff > 0 && *rate < MinRateValue {
		*rate = MinRateValue
	}
}

// bytesToMB converts bytes to megabytes
func bytesToMB(bytes uint64) uint64 {
	return bytes / MB
}

// GetNodeMetrics collects and returns current system metrics
func GetNodeMetrics() (NodeMetrics, error) {
	// Get memory information
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return NodeMetrics{}, err
	}

	swapInfo, err := mem.SwapMemory()
	if err != nil {
		return NodeMetrics{}, err
	}

	// Get disk usage information
	diskInfo, err := disk.Usage("/")
	if err != nil {
		return NodeMetrics{}, err
	}

	// Get disk IO counters
	ioCounters, err := disk.IOCounters()
	if err != nil {
		return NodeMetrics{}, err
	}
	
	var diskReadAcc, diskWriteAcc uint64
	for _, io := range ioCounters {
		diskReadAcc += io.ReadBytes
		diskWriteAcc += io.WriteBytes
	}
	currentDiskReadMB := bytesToMB(diskReadAcc)
	currentDiskWriteMB := bytesToMB(diskWriteAcc)

	// Get CPU information
	cpuInfo, err := cpu.Info()
	if err != nil || len(cpuInfo) == 0 {
		return NodeMetrics{}, err
	}
	
	cpuUsage, err := cpu.Percent(0, false)
	if err != nil {
		return NodeMetrics{}, err
	}
	
	cpuThreads, err := cpu.Counts(true)
	if err != nil {
		return NodeMetrics{}, err
	}

	// Get network statistics
	netStats, err := gnet.IOCounters(true)
	if err != nil {
		return NodeMetrics{}, err
	}
	
	var netInAcc, netOutAcc uint64
	for _, ifc := range netStats {
		netInAcc += ifc.BytesRecv
		netOutAcc += ifc.BytesSent
	}
	currentNetInMB := bytesToMB(netInAcc)
	currentNetOutMB := bytesToMB(netOutAcc)

	// Update history and calculate rates
	now := time.Now()
	history.add(now, currentDiskReadMB, currentDiskWriteMB, currentNetInMB, currentNetOutMB)
	
	var diskReadRate, diskWriteRate, netInRate, netOutRate float64
	
	if isFirstRun {
		isFirstRun = false
	} else {
		diskReadRate, diskWriteRate, netInRate, netOutRate = history.calculateRates()
	}

	// Validate CPU usage
	cpuUsageValue := 0.0
	if len(cpuUsage) > 0 {
		cpuUsageValue = cpuUsage[0]
		if cpuUsageValue < 0 || cpuUsageValue > 100 {
			cpuUsageValue = 0.0
		}
	}

	// Construct and return metrics
	return NodeMetrics{
		MemoryUsed:    bytesToMB(memInfo.Used),
		MemoryTotal:   bytesToMB(memInfo.Total),
		SwapUsed:      bytesToMB(swapInfo.Used),
		SwapTotal:     bytesToMB(swapInfo.Total),

		DiskUsed:      bytesToMB(diskInfo.Used),
		DiskTotal:     bytesToMB(diskInfo.Total),
		DiskRead:      currentDiskReadMB,
		DiskWrite:     currentDiskWriteMB,
		DiskReadRate:  diskReadRate,
		DiskWriteRate: diskWriteRate,

		NetIn:         currentNetInMB,
		NetOut:        currentNetOutMB,
		NetInRate:     netInRate,
		NetOutRate:    netOutRate,

		CPUUsed:       cpuUsageValue,
		CPUThreads:    cpuThreads,
		CPUModel:      cpuInfo[0].ModelName,
	}, nil
}