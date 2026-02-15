package power

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// MaxHistoryEntries is the maximum number of entries to keep in the history file.
	MaxHistoryEntries = 500

	// DefaultHistoryCount is the default number of entries to show.
	DefaultHistoryCount = 20

	historyFileName = "power-history.json"
)

// Snapshot holds a point-in-time power measurement.
type Snapshot struct {
	Timestamp    time.Time `json:"timestamp"`
	BatteryPct   int       `json:"battery_pct"`
	IsCharging   bool      `json:"is_charging"`
	CycleCount   int       `json:"cycle_count"`
	MaxCapacity  int       `json:"max_capacity_mah"`
	Temperature  float64   `json:"temperature_celsius"`
	ThermalLevel string    `json:"thermal_level"`
}

// TakeSnapshot captures the current power state as a Snapshot.
func TakeSnapshot() (*Snapshot, error) {
	status, err := GetStatus()
	if err != nil {
		return nil, fmt.Errorf("failed to get power status: %w", err)
	}

	thermal, err := GetThermal()
	if err != nil {
		return nil, fmt.Errorf("failed to get thermal info: %w", err)
	}

	return &Snapshot{
		Timestamp:    time.Now().UTC(),
		BatteryPct:   status.Percent,
		IsCharging:   status.IsCharging,
		CycleCount:   status.CycleCount,
		MaxCapacity:  status.MaxCapacity,
		Temperature:  status.Temperature,
		ThermalLevel: thermal.PressureLevel,
	}, nil
}

// historyPath returns the path to the power history file.
func historyPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".config", "macctl", historyFileName), nil
}

// LoadHistory reads all snapshots from the history file.
func LoadHistory() ([]Snapshot, error) {
	path, err := historyPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read history file: %w", err)
	}

	var snapshots []Snapshot
	if err := json.Unmarshal(data, &snapshots); err != nil {
		return nil, fmt.Errorf("failed to parse history file: %w", err)
	}

	return snapshots, nil
}

// SaveHistory writes snapshots to the history file, keeping at most MaxHistoryEntries.
func SaveHistory(snapshots []Snapshot) error {
	// Trim to max entries.
	if len(snapshots) > MaxHistoryEntries {
		snapshots = snapshots[len(snapshots)-MaxHistoryEntries:]
	}

	path, err := historyPath()
	if err != nil {
		return err
	}

	// Ensure directory exists.
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(snapshots, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write history file: %w", err)
	}

	return nil
}

// RecordSnapshot takes a snapshot and appends it to the history file.
func RecordSnapshot() (*Snapshot, error) {
	snap, err := TakeSnapshot()
	if err != nil {
		return nil, err
	}

	existing, err := LoadHistory()
	if err != nil {
		// If we can't load, start fresh.
		existing = nil
	}

	existing = append(existing, *snap)

	if err := SaveHistory(existing); err != nil {
		return nil, err
	}

	return snap, nil
}

// FilterHistory returns snapshots within the given duration from now.
func FilterHistory(snapshots []Snapshot, since time.Duration) []Snapshot {
	cutoff := time.Now().UTC().Add(-since)
	var filtered []Snapshot
	for _, s := range snapshots {
		if s.Timestamp.After(cutoff) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// ParseDuration parses a human-friendly duration string like "24h", "7d", "30m".
func ParseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration: %q", s)
	}

	unit := s[len(s)-1]
	numStr := s[:len(s)-1]

	var num int
	if _, err := fmt.Sscanf(numStr, "%d", &num); err != nil {
		return 0, fmt.Errorf("invalid duration: %q", s)
	}

	if num <= 0 {
		return 0, fmt.Errorf("duration must be positive: %q", s)
	}

	switch unit {
	case 'm':
		return time.Duration(num) * time.Minute, nil
	case 'h':
		return time.Duration(num) * time.Hour, nil
	case 'd':
		return time.Duration(num) * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported duration unit %q (use m, h, or d)", string(unit))
	}
}
