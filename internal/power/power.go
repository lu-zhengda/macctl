package power

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Status holds battery status information.
type Status struct {
	Percent           int     `json:"percent"`
	IsCharging        bool    `json:"is_charging"`
	ExternalConnected bool    `json:"external_connected"`
	TimeRemaining     string  `json:"time_remaining"`
	CycleCount        int     `json:"cycle_count"`
	Temperature       float64 `json:"temperature_celsius"`
	CurrentCapacity   int     `json:"current_capacity_mah"`
	MaxCapacity       int     `json:"max_capacity_mah"`
}

// Health holds battery health information.
type Health struct {
	DesignCapacity int     `json:"design_capacity_mah"`
	MaxCapacity    int     `json:"max_capacity_mah"`
	HealthPercent  float64 `json:"health_percent"`
	CycleCount     int     `json:"cycle_count"`
	Condition      string  `json:"condition"`
}

// ThermalInfo holds thermal status information.
type ThermalInfo struct {
	PressureLevel string `json:"pressure_level"`
	CPUTemp       string `json:"cpu_temp"`
}

// Assertion holds a power assertion entry.
type Assertion struct {
	PID    int    `json:"pid"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

// EnergyHog holds a top energy-consuming process.
type EnergyHog struct {
	PID     int     `json:"pid"`
	Command string  `json:"command"`
	CPU     float64 `json:"cpu_percent"`
}

// GetStatus returns current battery status.
func GetStatus() (*Status, error) {
	out, err := exec.Command("ioreg", "-r", "-c", "AppleSmartBattery", "-w", "0").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to read battery info: %w", err)
	}

	s := &Status{}
	raw := string(out)

	// CurrentCapacity and MaxCapacity from ioreg are percentages (0-100).
	// Use AppleRawCurrentCapacity and NominalChargeCapacity for mAh.
	s.Percent = extractInt(raw, `"CurrentCapacity"\s*=\s*(\d+)`)
	s.CurrentCapacity = extractInt(raw, `"AppleRawCurrentCapacity"\s*=\s*(\d+)`)
	s.MaxCapacity = extractInt(raw, `"NominalChargeCapacity"\s*=\s*(\d+)`)
	s.CycleCount = extractInt(raw, `"CycleCount"\s*=\s*(\d+)`)
	s.IsCharging = extractBool(raw, `"IsCharging"\s*=\s*(Yes|No)`)
	s.ExternalConnected = extractBool(raw, `"ExternalConnected"\s*=\s*(Yes|No)`)

	tempRaw := extractInt(raw, `"Temperature"\s*=\s*(\d+)`)
	if tempRaw > 0 {
		s.Temperature = float64(tempRaw) / 100.0
	}

	// Get time remaining from pmset.
	pmOut, err := exec.Command("pmset", "-g", "batt").Output()
	if err == nil {
		s.TimeRemaining = extractTimeRemaining(string(pmOut))
	}

	return s, nil
}

// GetHealth returns battery health information.
func GetHealth() (*Health, error) {
	out, err := exec.Command("ioreg", "-r", "-c", "AppleSmartBattery", "-w", "0").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to read battery health: %w", err)
	}

	raw := string(out)
	h := &Health{}

	h.DesignCapacity = extractInt(raw, `"DesignCapacity"\s*=\s*(\d+)`)
	h.MaxCapacity = extractInt(raw, `"NominalChargeCapacity"\s*=\s*(\d+)`)
	h.CycleCount = extractInt(raw, `"CycleCount"\s*=\s*(\d+)`)

	if h.DesignCapacity > 0 {
		h.HealthPercent = float64(h.MaxCapacity) / float64(h.DesignCapacity) * 100.0
	}

	h.Condition = "Normal"
	if h.HealthPercent < 80 {
		h.Condition = "Service Recommended"
	} else if h.HealthPercent < 50 {
		h.Condition = "Replace Soon"
	}

	return h, nil
}

// GetThermal returns thermal status information.
func GetThermal() (*ThermalInfo, error) {
	info := &ThermalInfo{
		PressureLevel: "nominal",
		CPUTemp:       "unavailable",
	}

	// Try to read thermal pressure from pmset.
	out, err := exec.Command("pmset", "-g", "thermlog").Output()
	if err == nil {
		raw := string(out)
		if strings.Contains(raw, "CPU_Speed_Limit") {
			re := regexp.MustCompile(`CPU_Speed_Limit\s*=\s*(\d+)`)
			if m := re.FindStringSubmatch(raw); len(m) > 1 {
				limit, _ := strconv.Atoi(m[1])
				switch {
				case limit >= 100:
					info.PressureLevel = "nominal"
				case limit >= 80:
					info.PressureLevel = "fair"
				case limit >= 50:
					info.PressureLevel = "serious"
				default:
					info.PressureLevel = "critical"
				}
			}
		}
	}

	// Try to read CPU temperature from powermetrics (may require sudo).
	tempOut, err := exec.Command("ioreg", "-r", "-c", "AppleSmartBattery", "-w", "0").Output()
	if err == nil {
		temp := extractInt(string(tempOut), `"Temperature"\s*=\s*(\d+)`)
		if temp > 0 {
			info.CPUTemp = fmt.Sprintf("%.1f C (battery sensor)", float64(temp)/100.0)
		}
	}

	return info, nil
}

// GetAssertions returns active power assertions.
func GetAssertions() ([]Assertion, error) {
	out, err := exec.Command("pmset", "-g", "assertions").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to read power assertions: %w", err)
	}

	return parseAssertions(string(out)), nil
}

// GetEnergyHogs returns top energy-consuming processes.
func GetEnergyHogs(n int) ([]EnergyHog, error) {
	out, err := exec.Command("ps", "-eo", "pid,pcpu,comm", "-r").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get energy hogs: %w", err)
	}

	return parseEnergyHogs(string(out), n), nil
}

func extractInt(s, pattern string) int {
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return 0
	}
	v, _ := strconv.Atoi(m[1])
	return v
}

func extractBool(s, pattern string) bool {
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return false
	}
	return m[1] == "Yes"
}

func extractTimeRemaining(s string) string {
	if strings.Contains(s, "charged") {
		return "fully charged"
	}
	if strings.Contains(s, "finishing charge") {
		return "finishing charge"
	}
	if strings.Contains(s, "(no estimate)") {
		return "calculating"
	}
	re := regexp.MustCompile(`(\d+:\d+)\s+remaining`)
	m := re.FindStringSubmatch(s)
	if len(m) > 1 {
		return m[1]
	}
	return "unknown"
}

func parseAssertions(output string) []Assertion {
	var assertions []Assertion

	// Match lines like:
	//   pid 123(processname): [0x0000123400000042] 00:00:00 AssertionType named: "Reason"
	re := regexp.MustCompile(`pid\s+(\d+)\(([^)]+)\):\s+\[0x[0-9a-f]+\]\s+[\d:]+\s+(\S+)\s+named:\s+"([^"]*)"`)
	for _, m := range re.FindAllStringSubmatch(output, -1) {
		pid, _ := strconv.Atoi(m[1])
		assertions = append(assertions, Assertion{
			PID:    pid,
			Name:   m[2],
			Type:   m[3],
			Reason: m[4],
		})
	}

	return assertions
}

func parseEnergyHogs(output string, n int) []EnergyHog {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		return nil
	}

	var hogs []EnergyHog
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		cpu, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			continue
		}
		command := strings.Join(fields[2:], " ")
		// Extract just the binary name.
		if idx := strings.LastIndex(command, "/"); idx >= 0 {
			command = command[idx+1:]
		}
		hogs = append(hogs, EnergyHog{
			PID:     pid,
			Command: command,
			CPU:     cpu,
		})
		if len(hogs) >= n {
			break
		}
	}

	return hogs
}
