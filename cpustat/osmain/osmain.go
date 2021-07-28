package osmain

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"time"

	"inspect/cpustat/metrics"

	"inspect/cpustat/os/cpustat"
	"inspect/cpustat/os/memstat"
	"inspect/cpustat/os/misc"
	"inspect/cpustat/os/pidstat"
)

// Number of Pids(in future cgroups etc) to display for top-N metrics
const MaxEntries = 1

// DisplayWidgets represents various variables used for display
// Perhaps this belongs to main package

// Stats represents all statistics collected and printed by osmain
type Stats struct {
	CPUStat     *cpustat.CPUStat
	MemStat     *memstat.MemStat
	ProcessStat *pidstat.ProcessStat
	Problems    []string // various problems spotted
	OsSpecific  interface{}
}

func GetProcessStat(currentPID string) ProcessAndVMStat {
	processState := ProcessAndVMStat{}
	runtime.GOMAXPROCS(1)
	var stepSec int = 3
	m := metrics.NewMetricContext("system")
	step := time.Millisecond * time.Duration(stepSec) * 1000
	stats := Register(m, step)

	n := 1
	for n < 5 {
		time.Sleep(step)
		processState = *stats.GetProcessState(currentPID)
		// if processState.ProcessStat.Pid != "" && processState.ProcessStat.PName != "" {
		// 	//b, _ := json.Marshal(processState)
		// 	//fmt.Println("processStateChan---", string(b))
		// 	//break
		// }
		// be aggressive about reclaiming memory
		// tradeoff with CPU usage
		runtime.GC()
		debug.FreeOSMemory()
		n++
	}
	return processState
}

// Register starts metrics collection for all available metrics
func Register(m *metrics.MetricContext, step time.Duration) *Stats {
	stats := new(Stats)
	// Collect cpu/memory/disk/perpid metrics
	stats.CPUStat = cpustat.New(m, step)
	stats.MemStat = memstat.New(m, step)
	p := pidstat.NewProcessStat(m, step)
	// Filter processes which have < 1% of a CPU or < 1% memory
	p.SetPidFilter(pidstat.PidFilterFunc(func(p *pidstat.PerProcessStat) bool {
		memUsagePct := (p.MemUsage() / stats.MemStat.Total()) * 100.0
		if p.CPUUsage() > 0.01 || memUsagePct > 1 {
			return true
		}
		return true
	}))
	stats.ProcessStat = p
	// register os dependent metrics
	// these could be specific to the OS (say cgroups)
	// or stats which are implemented not on all supported
	// platforms yet
	stats.OsSpecific = registerOsSpecific(m, step, stats)
	return stats
}

type ProcessStat struct {
	Pid             string   `json:"pid"`
	PName           string   `json:"name"`
	CpuUsages       string   `json:"cpu_usage"`
	CpuUser         string   `json:"cpu_user"`
	CpuSystem       string   `json:"cpu_system"`
	MemoryUsage     string   `json:"memory_usage"`
	MemoryTotal     string   `json:"memory_total"`
	ProblemDetected []string `json:"issue_detected"`
}

type VMStat struct {
	CpuUsages        string `json:"cpu_usage"`
	CpuUser          string `json:"cpu_user"`
	CpuSystem        string `json:"cpu_system"`
	MemoryUsage      string `json:"memory_usage"`
	MemoryTotal      string `json:"memory_total"`
	DiskStorageUsged string `json:"disk_storage_usage"`
}

type ProcessAndVMStat struct {
	User        string      `json:"user"`
	ProcessStat ProcessStat `json:"process_stat"`
	VMStat      VMStat      `json:"vm_stat"`
	LastError   string      `json:"last_error"`
}

// Print inspects and prints various metrics collected started by Register
func (stats *Stats) GetProcessState(currentPID string) *ProcessAndVMStat {
	var processAndVmStat = new(ProcessAndVMStat)
	// deal with stats that are available on platforms
	memPctUsage := (stats.MemStat.Usage() / stats.MemStat.Total()) * 100
	cpuPctUsage := (stats.CPUStat.Usage() / stats.CPUStat.Total()) * 100
	cpuUserspacePctUsage := (stats.CPUStat.UserSpace() / stats.CPUStat.Total()) * 100
	cpuKernelPctUsage := (stats.CPUStat.Kernel() / stats.CPUStat.Total()) * 100
	// Top processes by usage
	//currentPID := strconv.Itoa(10907)
	//currentPID := strconv.Itoa(os.Getpid())

	procsByCPUUsage := stats.ProcessStat.ByCPUUsage(currentPID)
	processCPUUserUsage := stats.ProcessStat.ByCPUUserUsage(currentPID)
	processCPUKernalUsage := stats.ProcessStat.ByCPUKernalUsage(currentPID)

	procsByMemUsage := stats.ProcessStat.ByMemUsage(currentPID)
	procsByTotalMemory := pidstat.GetPidTotalMemory(currentPID)

	// summary
	summaryLine := fmt.Sprintf("cpu: %3.1f%%, cpuUserUsage : %3.1f%%, cpuKernelUsage: %3.1f%%, memUsage: %3.1f%%",
		cpuPctUsage, cpuUserspacePctUsage, cpuKernelPctUsage, memPctUsage)

	vmRamsize := fmt.Sprintf("%3.2fGB", stats.MemStat.Total()/1024/1024/1024)

	fmt.Println(vmRamsize, " - ", summaryLine, " --- ", procsByTotalMemory)

	vmCpuPctUsage := fmt.Sprintf("%3.1f%%", cpuPctUsage)
	vmCpuUserspacePctUsage := fmt.Sprintf("%3.1f%%", cpuUserspacePctUsage)
	vmCpuKernelPctUsage := fmt.Sprintf("%3.1f%%", cpuKernelPctUsage)
	vmMemPctUsage := fmt.Sprintf("%3.1f%%", memPctUsage)

	if cpuPctUsage > 80.0 {
		stats.Problems = append(stats.Problems, "CPU usage is > 80%")
	}
	if cpuKernelPctUsage > 30.0 {
		stats.Problems = append(stats.Problems, "CPU usage in kernel is > 30%")
	}
	if memPctUsage > 80.0 {
		stats.Problems = append(stats.Problems, "Memory usage > 80%")
	}

	// Processes by cpu usage
	n := MaxEntries
	if len(procsByCPUUsage) < MaxEntries {
		n = len(procsByCPUUsage)
	}
	var cpuUsagePct float64
	var pid, user, processName string
	for i := 0; i < n; i++ {
		cpuUsagePct = (procsByCPUUsage[i].CPUUsage() / stats.CPUStat.Total()) * 100
		pid = procsByCPUUsage[i].Pid()
		user = truncate(procsByCPUUsage[i].User(), 10)
		processName = truncate(procsByCPUUsage[i].Comm(), 10)
	}
	processCpuUsage := fmt.Sprintf("%3.1f%%", cpuUsagePct)

	var cpuUserPct float64
	pcpuUser := MaxEntries
	if len(processCPUUserUsage) < MaxEntries {
		pcpuUser = len(processCPUUserUsage)
	}
	for i := 0; i < pcpuUser; i++ {
		cpuUserPct = (procsByCPUUsage[i].CPUUserUsage() / stats.CPUStat.Total()) * 100
	}
	processCpuUserUsage := fmt.Sprintf("%3.1f%%", cpuUserPct)

	var cpuKernalPct float64
	pkernalUser := MaxEntries
	if len(processCPUKernalUsage) < MaxEntries {
		pkernalUser = len(processCPUKernalUsage)
	}
	for i := 0; i < pkernalUser; i++ {
		cpuKernalPct = (procsByCPUUsage[i].CPUKernalUsage() / stats.CPUStat.Total()) * 100
	}
	processKernalUsage := fmt.Sprintf("%3.1f%%", cpuKernalPct)

	n = MaxEntries
	if len(procsByMemUsage) < MaxEntries {
		n = len(procsByMemUsage)
	}
	var processMemory string
	for i := 0; i < n; i++ {
		processMemory = fmt.Sprintf("%v", misc.ByteSize(procsByMemUsage[i].MemUsage()))
	}

	//vmRamsize := fmt.Sprintf("%3.2fGB", stats.MemStat.Total()/1024/1024/1024)

	diskIO := OsSpecific(stats.OsSpecific)

	processStat := new(ProcessStat)
	vmStat := new(VMStat)
	processAndVmStat.User = user

	processStat.Pid = pid
	processStat.PName = processName
	processStat.ProblemDetected = stats.Problems

	processStat.MemoryUsage = processMemory
	processStat.MemoryTotal = fmt.Sprintf("%3.1fMB", procsByTotalMemory)
	processStat.CpuUsages = processCpuUsage
	processStat.CpuUser = processCpuUserUsage
	processStat.CpuSystem = processKernalUsage

	vmStat.CpuUsages = vmCpuPctUsage
	vmStat.CpuUser = vmCpuUserspacePctUsage
	vmStat.CpuSystem = vmCpuKernelPctUsage
	vmStat.MemoryUsage = vmMemPctUsage
	vmStat.MemoryTotal = vmRamsize
	vmStat.DiskStorageUsged = diskIO

	processAndVmStat.ProcessStat = *processStat
	processAndVmStat.VMStat = *vmStat
	return processAndVmStat
}

// few small helper functions
func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
