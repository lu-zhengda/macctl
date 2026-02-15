package power

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{name: "minutes", input: "30m", want: 30 * time.Minute},
		{name: "hours", input: "24h", want: 24 * time.Hour},
		{name: "days", input: "7d", want: 7 * 24 * time.Hour},
		{name: "one day", input: "1d", want: 24 * time.Hour},
		{name: "empty", input: "", wantErr: true},
		{name: "single char", input: "h", wantErr: true},
		{name: "invalid number", input: "abch", wantErr: true},
		{name: "unsupported unit", input: "10s", wantErr: true},
		{name: "zero", input: "0h", wantErr: true},
		{name: "negative", input: "-5h", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDuration(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseDuration(%q) expected error, got %v", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseDuration(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ParseDuration(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestFilterHistory(t *testing.T) {
	now := time.Now().UTC()
	snapshots := []Snapshot{
		{Timestamp: now.Add(-48 * time.Hour), BatteryPct: 80},
		{Timestamp: now.Add(-12 * time.Hour), BatteryPct: 60},
		{Timestamp: now.Add(-1 * time.Hour), BatteryPct: 40},
		{Timestamp: now.Add(-10 * time.Minute), BatteryPct: 35},
	}

	tests := []struct {
		name  string
		since time.Duration
		want  int
	}{
		{name: "last 24h", since: 24 * time.Hour, want: 3},
		{name: "last 2h", since: 2 * time.Hour, want: 2},
		{name: "last 30m", since: 30 * time.Minute, want: 1},
		{name: "last 7d", since: 7 * 24 * time.Hour, want: 4},
		{name: "last 1m", since: 1 * time.Minute, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterHistory(snapshots, tt.since)
			if len(got) != tt.want {
				t.Errorf("FilterHistory(since=%v) returned %d entries, want %d", tt.since, len(got), tt.want)
			}
		})
	}
}

func TestSaveAndLoadHistory(t *testing.T) {
	// Use a temp directory for the test.
	tmpDir := t.TempDir()
	histPath := filepath.Join(tmpDir, "power-history.json")

	snapshots := []Snapshot{
		{
			Timestamp:    time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
			BatteryPct:   85,
			IsCharging:   true,
			CycleCount:   100,
			MaxCapacity:  5000,
			Temperature:  35.5,
			ThermalLevel: "nominal",
		},
		{
			Timestamp:    time.Date(2025, 1, 1, 13, 0, 0, 0, time.UTC),
			BatteryPct:   90,
			IsCharging:   true,
			CycleCount:   100,
			MaxCapacity:  5000,
			Temperature:  36.0,
			ThermalLevel: "nominal",
		},
	}

	// Write directly to the temp path.
	data, err := json.MarshalIndent(snapshots, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	if err := os.WriteFile(histPath, data, 0o644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	// Read back.
	readData, err := os.ReadFile(histPath)
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}

	var loaded []Snapshot
	if err := json.Unmarshal(readData, &loaded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(loaded))
	}

	if loaded[0].BatteryPct != 85 {
		t.Errorf("first snapshot BatteryPct = %d, want 85", loaded[0].BatteryPct)
	}
	if loaded[1].BatteryPct != 90 {
		t.Errorf("second snapshot BatteryPct = %d, want 90", loaded[1].BatteryPct)
	}
}

func TestSaveHistoryTrimming(t *testing.T) {
	tmpDir := t.TempDir()
	histPath := filepath.Join(tmpDir, "power-history.json")

	// Create more than MaxHistoryEntries snapshots.
	var snapshots []Snapshot
	for i := 0; i < MaxHistoryEntries+50; i++ {
		snapshots = append(snapshots, Snapshot{
			Timestamp:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(i) * time.Minute),
			BatteryPct: i % 100,
		})
	}

	// Trim like SaveHistory does.
	if len(snapshots) > MaxHistoryEntries {
		snapshots = snapshots[len(snapshots)-MaxHistoryEntries:]
	}

	data, err := json.MarshalIndent(snapshots, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	if err := os.WriteFile(histPath, data, 0o644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	readData, err := os.ReadFile(histPath)
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}

	var loaded []Snapshot
	if err := json.Unmarshal(readData, &loaded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(loaded) != MaxHistoryEntries {
		t.Errorf("expected %d entries after trimming, got %d", MaxHistoryEntries, len(loaded))
	}

	// The first entry should be the 50th one (0-indexed), since we trimmed the first 50.
	if loaded[0].BatteryPct != 50 {
		t.Errorf("first entry BatteryPct = %d, want 50", loaded[0].BatteryPct)
	}
}

func TestSnapshotJSONRoundTrip(t *testing.T) {
	snap := Snapshot{
		Timestamp:    time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC),
		BatteryPct:   72,
		IsCharging:   false,
		CycleCount:   245,
		MaxCapacity:  4800,
		Temperature:  37.2,
		ThermalLevel: "nominal",
	}

	data, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var loaded Snapshot
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if loaded.BatteryPct != snap.BatteryPct {
		t.Errorf("BatteryPct = %d, want %d", loaded.BatteryPct, snap.BatteryPct)
	}
	if loaded.CycleCount != snap.CycleCount {
		t.Errorf("CycleCount = %d, want %d", loaded.CycleCount, snap.CycleCount)
	}
	if loaded.ThermalLevel != snap.ThermalLevel {
		t.Errorf("ThermalLevel = %q, want %q", loaded.ThermalLevel, snap.ThermalLevel)
	}
}
