package disk

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Health holds SSD health information.
type Health struct {
	Device      string `json:"device"`
	Model       string `json:"model"`
	Protocol    string `json:"protocol"`
	SizeBytes   int64  `json:"size_bytes"`
	SizeHuman   string `json:"size_human"`
	WearLevel   string `json:"wear_level"`
	DataWritten string `json:"data_written"`
	SmartStatus string `json:"smart_status"`
}

// IOStats holds current I/O rate information.
type IOStats struct {
	ReadMBs   float64 `json:"read_mbs"`
	WriteMBs  float64 `json:"write_mbs"`
	ReadIOPS  float64 `json:"read_iops"`
	WriteIOPS float64 `json:"write_iops"`
}

// GetHealth returns disk health information for disk0.
func GetHealth() (*Health, error) {
	diskutilOut, err := exec.Command("diskutil", "info", "disk0").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run diskutil: %w", err)
	}

	h := parseDiskutilInfo(string(diskutilOut))

	// Try to get NVMe-specific data.
	nvmeOut, err := exec.Command("system_profiler", "SPNVMeDataType", "-json").Output()
	if err == nil {
		enrichWithNVMe(h, nvmeOut)
	}

	return h, nil
}

// GetIOStats returns current disk I/O rates by running iostat with two samples.
func GetIOStats() (*IOStats, error) {
	// Take 2 samples at 1-second interval; the second sample gives accurate rates.
	out, err := exec.Command("iostat", "-d", "-c", "2", "-w", "1").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run iostat: %w", err)
	}

	return parseIOStat(string(out))
}

func parseDiskutilInfo(output string) *Health {
	h := &Health{
		Device:      "disk0",
		SmartStatus: "unknown",
		WearLevel:   "unavailable",
		DataWritten: "unavailable",
	}

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "Device / Media Name":
			h.Model = val
		case "Protocol":
			h.Protocol = val
		case "Disk Size":
			h.SizeHuman = val
			h.SizeBytes = parseSizeBytes(val)
		case "SMART Status":
			h.SmartStatus = val
		}
	}

	return h
}

func parseSizeBytes(s string) int64 {
	// Format: "500.1 GB (500107862016 Bytes)" or similar.
	re := regexp.MustCompile(`\((\d+)\s+Bytes\)`)
	m := re.FindStringSubmatch(s)
	if len(m) > 1 {
		v, err := strconv.ParseInt(m[1], 10, 64)
		if err == nil {
			return v
		}
	}
	return 0
}

type nvmeProfiler struct {
	SPNVMeDataType []struct {
		Items []nvmeItem `json:"_items"`
	} `json:"SPNVMeDataType"`
}

type nvmeItem struct {
	DeviceName         string `json:"_name"`
	DeviceModel        string `json:"device_model"`
	WearLevelCount     string `json:"spnvme_wearleveling"`
	DataBytesWritten   string `json:"spnvme_byteswritten"`
	DataBytesRead      string `json:"spnvme_bytesread"`
	SmartHealthStatus  string `json:"spnvme_smart_status"`
}

func enrichWithNVMe(h *Health, data []byte) {
	var sp nvmeProfiler
	if err := json.Unmarshal(data, &sp); err != nil {
		return
	}

	for _, group := range sp.SPNVMeDataType {
		for _, item := range group.Items {
			// Use the first NVMe device found (usually the primary SSD).
			if item.DeviceModel != "" && h.Model == "" {
				h.Model = item.DeviceModel
			}
			if item.WearLevelCount != "" {
				h.WearLevel = item.WearLevelCount
			}
			if item.DataBytesWritten != "" {
				h.DataWritten = item.DataBytesWritten
			}
			if item.SmartHealthStatus != "" {
				h.SmartStatus = item.SmartHealthStatus
			}
			return // Use first device.
		}
	}
}

func parseIOStat(output string) (*IOStats, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// iostat outputs a header block and then data lines. With -c 2, we get
	// two data sections. We want the last data line (the second sample).
	// Lines look like:
	//              disk0
	//     KB/t  tps  MB/s
	//    xx.xx  xxx  x.xx
	//    xx.xx  xxx  x.xx

	var dataLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip header lines (contain non-numeric first field).
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		_, err := strconv.ParseFloat(fields[0], 64)
		if err == nil {
			dataLines = append(dataLines, line)
		}
	}

	if len(dataLines) < 2 {
		// If we have at least one data line, use it.
		if len(dataLines) == 1 {
			return parseIOStatLine(dataLines[0])
		}
		return nil, fmt.Errorf("insufficient iostat data")
	}

	// Use the last data line (second sample).
	return parseIOStatLine(dataLines[len(dataLines)-1])
}

func parseIOStatLine(line string) (*IOStats, error) {
	fields := strings.Fields(line)
	// Default iostat -d output: KB/t  tps  MB/s
	if len(fields) < 3 {
		return nil, fmt.Errorf("unexpected iostat format: %q", line)
	}

	tps, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tps: %w", err)
	}

	mbs, err := strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MB/s: %w", err)
	}

	// iostat -d without -I gives combined read+write.
	// We report them as combined since basic iostat doesn't distinguish.
	return &IOStats{
		ReadMBs:   mbs,
		WriteMBs:  0, // iostat -d gives combined, not separate.
		ReadIOPS:  tps,
		WriteIOPS: 0,
	}, nil
}
