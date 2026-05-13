package system

import (
	"os"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

type Stats struct {
	Hostname    string  `json:"hostname"`
	OS          string  `json:"os"`
	Platform    string  `json:"platform"`
	Uptime      uint64  `json:"uptime"`
	CPUPercent  float64 `json:"cpu_percent"`
	CPUCores    int     `json:"cpu_cores"`
	LoadAvg1    float64 `json:"load1"`
	LoadAvg5    float64 `json:"load5"`
	LoadAvg15   float64 `json:"load15"`
	MemUsed     uint64  `json:"mem_used"`
	MemTotal    uint64  `json:"mem_total"`
	MemPercent  float64 `json:"mem_percent"`
	DiskUsed    uint64  `json:"disk_used"`
	DiskTotal   uint64  `json:"disk_total"`
	DiskPercent float64 `json:"disk_percent"`
	NetUp       uint64  `json:"net_up"`
	NetDown     uint64  `json:"net_down"`
	GoVersion   string  `json:"go_version"`
}

func Collect() Stats {
	s := Stats{GoVersion: runtime.Version(), CPUCores: runtime.NumCPU()}
	if hn, err := os.Hostname(); err == nil {
		s.Hostname = hn
	}
	if info, err := host.Info(); err == nil {
		s.OS = info.OS
		s.Platform = info.Platform + " " + info.PlatformVersion
		s.Uptime = info.Uptime
	}
	if pcts, err := cpu.Percent(150*time.Millisecond, false); err == nil && len(pcts) > 0 {
		s.CPUPercent = pcts[0]
	}
	if l, err := load.Avg(); err == nil {
		s.LoadAvg1, s.LoadAvg5, s.LoadAvg15 = l.Load1, l.Load5, l.Load15
	}
	if m, err := mem.VirtualMemory(); err == nil {
		s.MemUsed = m.Used
		s.MemTotal = m.Total
		s.MemPercent = m.UsedPercent
	}
	if d, err := disk.Usage("/"); err == nil {
		s.DiskUsed = d.Used
		s.DiskTotal = d.Total
		s.DiskPercent = d.UsedPercent
	}
	if io, err := net.IOCounters(false); err == nil && len(io) > 0 {
		s.NetUp = io[0].BytesSent
		s.NetDown = io[0].BytesRecv
	}
	return s
}
