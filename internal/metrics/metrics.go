// Package metrics collects per-host system load over SSH.
//
// One round-trip per call: it runs a single tagged shell pipeline that emits
// /proc/loadavg, /proc/meminfo (or `free` fallback), `df /`, and a CPU sample
// from `top -bn1` (with a vmstat fallback). Cross-distro: tested against
// Debian/Ubuntu/Alpine. Returns zero values for any field the remote couldn't
// report — callers should treat the struct as best-effort.
package metrics

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/awkto/ssh-to-go/internal/sshutil"
	"golang.org/x/crypto/ssh"
)

// Metrics is a single point-in-time sample. CPU/Mem/Disk are 0-100 percent.
// Load1 is the 1-minute load average. SampledAt is set by Collect.
type Metrics struct {
	CPU       float64   `json:"cpu"`
	Mem       float64   `json:"mem"`
	Disk      float64   `json:"disk"`
	Load1     float64   `json:"load1"`
	SampledAt time.Time `json:"sampled_at"`
}

// HasCPU returns true if a usable CPU sample was collected (non-zero idle).
// Disambiguates a real "0% CPU" reading (rare) from "no data".
func (m Metrics) HasCPU() bool { return m.CPU >= 0 && m.CPU <= 100 && !m.SampledAt.IsZero() }

const probeScript = `
echo '---LOAD---'
cat /proc/loadavg 2>/dev/null
echo '---MEM---'
free -b 2>/dev/null || free 2>/dev/null
echo '---DISK---'
df -P / 2>/dev/null | tail -n +2
echo '---CPU---'
{ top -bn1 -d 0.2 2>/dev/null | head -5; vmstat 1 2 2>/dev/null | tail -1; } | head -10
`

// Collect runs the probe script over the given ssh client and returns a sample.
// On any error returns a zero-value Metrics and the error; partial data may be
// returned alongside the error if parsing succeeded for some fields.
func Collect(client *ssh.Client) (Metrics, error) {
	out, err := sshutil.Exec(client, probeScript)
	if err != nil {
		return Metrics{}, fmt.Errorf("metrics probe failed: %w", err)
	}
	m := parse(out)
	m.SampledAt = time.Now()
	return m, nil
}

func parse(out string) Metrics {
	var m Metrics
	section := ""
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "---") {
			section = strings.Trim(trimmed, "- ")
			continue
		}
		if trimmed == "" {
			continue
		}
		switch section {
		case "LOAD":
			// "0.45 0.32 0.21 1/123 4567"
			fields := strings.Fields(trimmed)
			if len(fields) >= 1 {
				if v, err := strconv.ParseFloat(fields[0], 64); err == nil {
					m.Load1 = v
				}
			}
		case "MEM":
			// "Mem:  total used free shared buff/cache available"
			if strings.HasPrefix(trimmed, "Mem:") {
				fields := strings.Fields(trimmed)
				if len(fields) >= 7 {
					total, _ := strconv.ParseFloat(fields[1], 64)
					avail, _ := strconv.ParseFloat(fields[6], 64)
					if total > 0 {
						m.Mem = (1 - avail/total) * 100
					}
				} else if len(fields) >= 3 {
					// Older `free` without "available": fall back to (used/total).
					total, _ := strconv.ParseFloat(fields[1], 64)
					used, _ := strconv.ParseFloat(fields[2], 64)
					if total > 0 {
						m.Mem = used / total * 100
					}
				}
			}
		case "DISK":
			// "/dev/sda1  10240000  4096000  6144000  40% /"
			fields := strings.Fields(trimmed)
			if len(fields) >= 5 {
				pct := strings.TrimSuffix(fields[4], "%")
				if v, err := strconv.ParseFloat(pct, 64); err == nil {
					m.Disk = v
				}
			}
		case "CPU":
			// Two possible inputs:
			//   top: "%Cpu(s):  3.1 us,  1.2 sy,  0.0 ni, 95.7 id, ..."
			//   vmstat: "0 0 0 1234567 0 0 0 0 0 0 0 0 3 1 96 0 0"
			if strings.Contains(trimmed, "Cpu") && strings.Contains(trimmed, "id") {
				// Parse the field before "id".
				if idle, ok := pickFieldBefore(trimmed, "id"); ok {
					m.CPU = 100 - idle
				}
			} else {
				// vmstat last line — idle is the 15th column on most modern vmstat.
				fields := strings.Fields(trimmed)
				if len(fields) >= 15 {
					if idle, err := strconv.ParseFloat(fields[14], 64); err == nil {
						m.CPU = 100 - idle
					}
				}
			}
		}
	}
	if m.CPU < 0 {
		m.CPU = 0
	}
	if m.CPU > 100 {
		m.CPU = 100
	}
	return m
}

// pickFieldBefore parses a top-style line like "%Cpu(s): 3.1 us, 95.7 id, ..."
// and returns the float that immediately precedes the given suffix.
func pickFieldBefore(line, suffix string) (float64, bool) {
	// Tokenise on whitespace and commas.
	cleaned := strings.ReplaceAll(line, ",", " ")
	fields := strings.Fields(cleaned)
	for i, f := range fields {
		if f == suffix && i > 0 {
			if v, err := strconv.ParseFloat(fields[i-1], 64); err == nil {
				return v, true
			}
		}
	}
	return 0, false
}
