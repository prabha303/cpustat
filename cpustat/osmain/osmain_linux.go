//
// +build linux

package osmain

import (
	"fmt"
	"log"
	"strings"
	"time"

	"inspect/cpustat/metrics"
	"inspect/cpustat/os/diskstat"
)

type linuxStats struct {
	//osind       *Stats
	dstat *diskstat.DiskStat
	// fsstat      *fsstat.FSStat
	// ifstat      *interfacestat.InterfaceStat
	// netstat     *netstat.NetStat
	// cgMem       *memstat.CgroupStat
	// cgCPU       *cpustat.CgroupStat
	// loadstat    *loadstat.LoadStat
	// uptimestat  *uptimestat.UptimeStat
	// entropystat *entropystat.EntropyStat
}

// RegisterOsSpecific registers OS dependent statistics
func registerOsSpecific(m *metrics.MetricContext, step time.Duration,
	osind *Stats) *linuxStats {
	s := new(linuxStats)
	s.dstat = diskstat.New(m, step)
	return s
}

// PrintOsSpecific prints OS dependent statistics
func OsSpecific(v interface{}) string {
	stats, ok := v.(*linuxStats)
	if !ok {
		log.Fatalf("Type assertion failed on printOsSpecific")
	}
	// disk stats
	diskIOByUsage := stats.dstat.ByUsage()
	var diskio []string
	// TODO(syamp): remove magic number
	for i := 0; i < 5; i++ {
		diskName := "-"
		diskIO := 0.0
		if len(diskIOByUsage) > i {
			d := diskIOByUsage[i]
			diskName = d.Name
			diskIO = d.Usage()
		}
		if diskName != "-" {
			diskio = append(diskio, fmt.Sprintf("%6s:%5s,", diskName, fmt.Sprintf("%3.1f%%", diskIO)))
		}
	}
	return strings.ReplaceAll(fmt.Sprintf("%v", diskio), " ", "")
}
