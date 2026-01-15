package system

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cashbacktv/backend/internal/domain"
)

var (
	// Cache for system info to reduce /proc file reads
	systemInfoCache struct {
		mu          sync.RWMutex
		data        *domain.SystemInfo
		lastUpdate  time.Time
		cacheExpiry time.Duration
	}
	
	// Static info that doesn't change (only read once)
	staticInfo struct {
		mu          sync.Once
		cpuCores    int
		cpuThreads  int
	}
)

func init() {
	// Initialize static info once
	staticInfo.mu.Do(func() {
		// Get physical cores and threads from /proc/cpuinfo
		cores, threads := getCPUInfo()
		staticInfo.cpuCores = cores
		staticInfo.cpuThreads = threads
	})
	
	// Set cache expiry to 5 seconds (balance between freshness and performance)
	systemInfoCache.cacheExpiry = 5 * time.Second
}

// getCPUInfo reads physical cores and logical threads from /proc/cpuinfo
func getCPUInfo() (cores int, threads int) {
	// Default fallback to runtime.NumCPU() (gives logical CPUs/threads)
	threads = runtime.NumCPU()
	cores = threads // Default to same if we can't determine

	// Try to read from /proc/cpuinfo
	cpuinfoPath := "/proc/cpuinfo"
	data, err := os.ReadFile(cpuinfoPath)
	if err != nil {
		// If /proc/cpuinfo doesn't exist (e.g., Windows/Mac in development), use runtime
		// Estimate cores based on common hyperthreading (2 threads per core)
		if threads >= 2 && threads%2 == 0 {
			cores = threads / 2
		}
		return cores, threads
	}

	// Parse /proc/cpuinfo - count unique (physical_id, core_id) combinations
	// Physical cores = unique (physical_id, core_id) pairs
	// Logical threads = number of processor entries
	coreMap := make(map[string]bool) // Key: "physical_id:core_id"
	cpuCount := 0

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	currentPhysicalID := ""
	currentCoreID := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		if line == "" {
			// Empty line indicates end of current processor entry
			// Count unique core if we have both IDs
			if currentPhysicalID != "" && currentCoreID != "" {
				coreKey := fmt.Sprintf("%s:%s", currentPhysicalID, currentCoreID)
				coreMap[coreKey] = true
			}
			if currentPhysicalID != "" || currentCoreID != "" {
				cpuCount++ // Count this logical CPU
			}
			currentPhysicalID = ""
			currentCoreID = ""
			continue
		}

		// Parse processor line (logical CPU number) - just count them
		if strings.HasPrefix(line, "processor") {
			// Don't count here, count on empty line or end of file
			continue
		}

		// Parse physical id (socket)
		if strings.HasPrefix(line, "physical id") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				currentPhysicalID = strings.TrimSpace(parts[1])
			}
		}

		// Parse core id (core within socket)
		if strings.HasPrefix(line, "core id") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				currentCoreID = strings.TrimSpace(parts[1])
			}
		}
	}

	// Handle last entry if file doesn't end with empty line
	if currentPhysicalID != "" && currentCoreID != "" {
		coreKey := fmt.Sprintf("%s:%s", currentPhysicalID, currentCoreID)
		coreMap[coreKey] = true
	}
	if currentPhysicalID != "" || currentCoreID != "" {
		cpuCount++
	}

	// Set results based on what we found
	if len(coreMap) > 0 {
		// Successfully parsed: physical cores from unique combinations
		cores = len(coreMap)
		if cpuCount > 0 {
			threads = cpuCount
		} else {
			// Fallback: count processor lines if cpuCount is 0
			threads = runtime.NumCPU()
		}
	} else if cpuCount > 0 {
		// We counted processors but couldn't determine physical cores
		threads = cpuCount
		// Estimate: assume hyperthreading (2 threads per core)
		if threads >= 2 && threads%2 == 0 {
			cores = threads / 2
		} else {
			cores = threads // Fallback: assume no HT
		}
	} else {
		// Couldn't parse anything, use runtime estimation
		threads = runtime.NumCPU()
		if threads >= 2 && threads%2 == 0 {
			cores = threads / 2
		} else {
			cores = threads
		}
	}

	return cores, threads
}

// GetSystemInfo retrieves current system information with caching
func GetSystemInfo() (*domain.SystemInfo, error) {
	systemInfoCache.mu.RLock()
	
	// Return cached data if still valid
	if systemInfoCache.data != nil && time.Since(systemInfoCache.lastUpdate) < systemInfoCache.cacheExpiry {
		cached := *systemInfoCache.data // Copy to avoid race conditions
		systemInfoCache.mu.RUnlock()
		return &cached, nil
	}
	systemInfoCache.mu.RUnlock()

	// Cache expired or doesn't exist, update it
	systemInfoCache.mu.Lock()
	defer systemInfoCache.mu.Unlock()

	// Double-check after acquiring write lock (another goroutine might have updated it)
	if systemInfoCache.data != nil && time.Since(systemInfoCache.lastUpdate) < systemInfoCache.cacheExpiry {
		cached := *systemInfoCache.data
		return &cached, nil
	}

	info := &domain.SystemInfo{}

	// Get CPU information (static, doesn't change)
	info.CPUCores = staticInfo.cpuCores
	info.CPUThreads = staticInfo.cpuThreads

	// Get CPU usage from /proc/stat (lightweight, cached internally)
	cpuUsage, err := getCPUUsage()
	if err == nil {
		info.CPUUsage = cpuUsage
	}

	// Get memory information from /proc/meminfo
	memInfo, err := getMemoryInfo()
	if err == nil {
		info.MemoryTotal = memInfo.Total
		info.MemoryUsed = memInfo.Used
		info.MemoryAvailable = memInfo.Available
		info.MemoryPercent = memInfo.Percent
	}

	// Get load average from /proc/loadavg
	loadAvg, err := getLoadAverage()
	if err == nil {
		info.LoadAverage1 = loadAvg[0]
		info.LoadAverage5 = loadAvg[1]
		info.LoadAverage15 = loadAvg[2]
	}

	// Get uptime from /proc/uptime
	uptime, err := getUptime()
	if err == nil {
		info.Uptime = uptime
	}

	// Get GPU information
	gpus, err := getGPUInfo()
	if err == nil {
		info.GPUs = gpus
	}

	// Update cache
	systemInfoCache.data = info
	systemInfoCache.lastUpdate = time.Now()

	// Return a copy to avoid race conditions
	result := *info
	return &result, nil
}

// CPU usage tracking with mutex for thread safety
var (
	cpuStatsMu   sync.Mutex
	lastCPUStats *cpuStats
	lastCPUTime  time.Time
)

type cpuStats struct {
	user    uint64
	nice    uint64
	system  uint64
	idle    uint64
	iowait  uint64
	irq     uint64
	softirq uint64
	steal   uint64
	guest   uint64
}

func getCPUUsage() (float64, error) {
	cpuStatsMu.Lock()
	defer cpuStatsMu.Unlock()

	statPath := "/proc/stat"
	data, err := os.ReadFile(statPath)
	if err != nil {
		return 0, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	if !scanner.Scan() {
		return 0, fmt.Errorf("could not read CPU line")
	}

	line := scanner.Text()
	if !strings.HasPrefix(line, "cpu ") {
		return 0, fmt.Errorf("invalid CPU line")
	}

	fields := strings.Fields(line[4:]) // Skip "cpu "
	if len(fields) < 8 {
		return 0, fmt.Errorf("insufficient CPU fields")
	}

	stats := &cpuStats{}
	stats.user, _ = strconv.ParseUint(fields[0], 10, 64)
	stats.nice, _ = strconv.ParseUint(fields[1], 10, 64)
	stats.system, _ = strconv.ParseUint(fields[2], 10, 64)
	stats.idle, _ = strconv.ParseUint(fields[3], 10, 64)
	stats.iowait, _ = strconv.ParseUint(fields[4], 10, 64)
	stats.irq, _ = strconv.ParseUint(fields[5], 10, 64)
	stats.softirq, _ = strconv.ParseUint(fields[6], 10, 64)
	stats.steal, _ = strconv.ParseUint(fields[7], 10, 64)
	if len(fields) > 8 {
		stats.guest, _ = strconv.ParseUint(fields[8], 10, 64)
	}

	now := time.Now()

	if lastCPUStats == nil {
		lastCPUStats = stats
		lastCPUTime = now
		return 0, nil // First call, return 0
	}

	// Calculate CPU usage percentage
	totalTime := (stats.user + stats.nice + stats.system + stats.idle + stats.iowait + stats.irq + stats.softirq + stats.steal) -
		(lastCPUStats.user + lastCPUStats.nice + lastCPUStats.system + lastCPUStats.idle + lastCPUStats.iowait + lastCPUStats.irq + lastCPUStats.softirq + lastCPUStats.steal)

	idleTime := stats.idle - lastCPUStats.idle
	usedTime := totalTime - idleTime

	elapsed := now.Sub(lastCPUTime).Seconds()
	if elapsed == 0 || totalTime == 0 {
		// Update stats but return previous value or 0
		lastCPUStats = stats
		lastCPUTime = now
		return 0, nil
	}

	// CPU usage percentage
	cpuUsage := (float64(usedTime) / float64(totalTime)) * 100.0

	// Update last stats
	lastCPUStats = stats
	lastCPUTime = now

	return cpuUsage, nil
}

type memoryInfo struct {
	Total     int64
	Used      int64
	Available int64
	Percent   float64
}

func getMemoryInfo() (*memoryInfo, error) {
	memInfoPath := "/proc/meminfo"
	data, err := os.ReadFile(memInfoPath)
	if err != nil {
		return nil, err
	}

	info := &memoryInfo{}
	
	// Optimize: Only scan for the lines we need (MemTotal, MemAvailable, MemFree)
	// This is faster than scanning all lines
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		
		// Early exit if we found all needed values
		if info.Total > 0 && info.Available > 0 {
			break
		}
		
		// Only process lines we care about
		if !strings.HasPrefix(line, "MemTotal:") && 
		   !strings.HasPrefix(line, "MemAvailable:") && 
		   !strings.HasPrefix(line, "MemFree:") {
			continue
		}
		
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		key := strings.TrimSuffix(fields[0], ":")
		value, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			continue
		}

		// Values are in KB, convert to bytes
		valueBytes := value * 1024

		switch key {
		case "MemTotal":
			info.Total = valueBytes
		case "MemAvailable":
			info.Available = valueBytes
		case "MemFree":
			// Use MemAvailable if not set
			if info.Available == 0 {
				info.Available = valueBytes
			}
		}
	}

	if info.Total > 0 {
		info.Used = info.Total - info.Available
		info.Percent = (float64(info.Used) / float64(info.Total)) * 100.0
	}

	return info, nil
}

func getLoadAverage() ([]float64, error) {
	loadAvgPath := "/proc/loadavg"
	data, err := os.ReadFile(loadAvgPath)
	if err != nil {
		return nil, err
	}

	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return nil, fmt.Errorf("insufficient load average fields")
	}

	load1, _ := strconv.ParseFloat(fields[0], 64)
	load5, _ := strconv.ParseFloat(fields[1], 64)
	load15, _ := strconv.ParseFloat(fields[2], 64)

	return []float64{load1, load5, load15}, nil
}

func getUptime() (int64, error) {
	uptimePath := "/proc/uptime"
	data, err := os.ReadFile(uptimePath)
	if err != nil {
		return 0, err
	}

	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0, fmt.Errorf("insufficient uptime fields")
	}

	uptime, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, err
	}

	return int64(uptime), nil
}

// getGPUInfo retrieves GPU information using nvidia-smi
func getGPUInfo() ([]domain.GPUInfo, error) {
	// Query NVIDIA GPU status
	// format: index, name, utilization.gpu [%], memory.used [MiB], memory.total [MiB], temperature.gpu [C]
	cmd := exec.Command("nvidia-smi", "--query-gpu=index,name,utilization.gpu,memory.used,memory.total,temperature.gpu", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	gpus := make([]domain.GPUInfo, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Split(line, ",")
		if len(fields) < 6 {
			continue
		}

		gpu := domain.GPUInfo{
			ID:   strings.TrimSpace(fields[0]),
			Name: strings.TrimSpace(fields[1]),
		}

		// Parse utilization
		if val, err := strconv.ParseFloat(strings.TrimSpace(fields[2]), 64); err == nil {
			gpu.Utilization = val
		}

		// Parse memory used (MiB to bytes)
		if val, err := strconv.ParseInt(strings.TrimSpace(fields[3]), 10, 64); err == nil {
			gpu.MemoryUsed = val * 1024 * 1024
		}

		// Parse memory total (MiB to bytes)
		if val, err := strconv.ParseInt(strings.TrimSpace(fields[4]), 10, 64); err == nil {
			gpu.MemoryTotal = val * 1024 * 1024
		}

		// Parse temperature
		if val, err := strconv.Atoi(strings.TrimSpace(fields[5])); err == nil {
			gpu.Temperature = val
		}

		gpus = append(gpus, gpu)
	}

	return gpus, nil
}
