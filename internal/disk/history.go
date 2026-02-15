package disk

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lu-zhengda/macctl/internal/power"
)

const (
	// MaxHistoryEntries is the maximum number of disk history entries to keep.
	MaxHistoryEntries = 500

	// DefaultHistoryCount is the default number of entries to show.
	DefaultHistoryCount = 20

	historyFileName = "disk-history.json"
)

// HealthSnapshot holds a point-in-time disk health measurement.
type HealthSnapshot struct {
	Timestamp   time.Time `json:"timestamp"`
	Model       string    `json:"model"`
	SmartStatus string    `json:"smart_status"`
	WearLevel   string    `json:"wear_level"`
	DataWritten string    `json:"data_written"`
	SizeBytes   int64     `json:"size_bytes"`
}

// historyPath returns the path to the disk history file.
func historyPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".config", "macctl", historyFileName), nil
}

// LoadHistory reads all disk health snapshots from the history file.
func LoadHistory() ([]HealthSnapshot, error) {
	path, err := historyPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read disk history file: %w", err)
	}

	var snapshots []HealthSnapshot
	if err := json.Unmarshal(data, &snapshots); err != nil {
		return nil, fmt.Errorf("failed to parse disk history file: %w", err)
	}

	return snapshots, nil
}

// SaveHistory writes disk health snapshots to the history file.
func SaveHistory(snapshots []HealthSnapshot) error {
	if len(snapshots) > MaxHistoryEntries {
		snapshots = snapshots[len(snapshots)-MaxHistoryEntries:]
	}

	path, err := historyPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(snapshots, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal disk history: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write disk history file: %w", err)
	}

	return nil
}

// RecordSnapshot takes a disk health snapshot and appends it to history.
func RecordSnapshot() (*HealthSnapshot, error) {
	health, err := GetHealth()
	if err != nil {
		return nil, err
	}

	snap := &HealthSnapshot{
		Timestamp:   time.Now().UTC(),
		Model:       health.Model,
		SmartStatus: health.SmartStatus,
		WearLevel:   health.WearLevel,
		DataWritten: health.DataWritten,
		SizeBytes:   health.SizeBytes,
	}

	existing, err := LoadHistory()
	if err != nil {
		existing = nil
	}

	existing = append(existing, *snap)

	if err := SaveHistory(existing); err != nil {
		return nil, err
	}

	return snap, nil
}

// FilterHistory returns snapshots within the given duration from now.
// Reuses power.ParseDuration for duration string parsing.
func FilterHistory(snapshots []HealthSnapshot, since time.Duration) []HealthSnapshot {
	cutoff := time.Now().UTC().Add(-since)
	var filtered []HealthSnapshot
	for _, s := range snapshots {
		if s.Timestamp.After(cutoff) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// ParseDuration delegates to power.ParseDuration for consistent duration parsing.
var ParseDuration = power.ParseDuration
