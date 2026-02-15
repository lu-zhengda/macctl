package disk

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseDiskutilInfo(t *testing.T) {
	input := `   Device Identifier:         disk0
   Device Node:               /dev/disk0
   Whole:                     Yes
   Part of Whole:             disk0
   Device / Media Name:       APPLE SSD AP0512Q
   Volume Name:               Not applicable
   Mounted:                   Not applicable
   File System:               None
   Content (IOContent):       GUID_partition_scheme
   OS Can Be Installed:       No
   Media Type:                Generic
   Protocol:                  Apple Fabric
   SMART Status:              Verified
   Disk Size:                 500.1 GB (500107862016 Bytes)
`

	h := parseDiskutilInfo(input)

	if h.Model != "APPLE SSD AP0512Q" {
		t.Errorf("Model = %q, want %q", h.Model, "APPLE SSD AP0512Q")
	}
	if h.Protocol != "Apple Fabric" {
		t.Errorf("Protocol = %q, want %q", h.Protocol, "Apple Fabric")
	}
	if h.SmartStatus != "Verified" {
		t.Errorf("SmartStatus = %q, want %q", h.SmartStatus, "Verified")
	}
	if h.SizeBytes != 500107862016 {
		t.Errorf("SizeBytes = %d, want %d", h.SizeBytes, 500107862016)
	}
	if h.Device != "disk0" {
		t.Errorf("Device = %q, want %q", h.Device, "disk0")
	}
}

func TestParseDiskutilInfoMinimal(t *testing.T) {
	input := `   Device Identifier:         disk0
   Protocol:                  NVMe
   SMART Status:              Not Supported
`

	h := parseDiskutilInfo(input)

	if h.Protocol != "NVMe" {
		t.Errorf("Protocol = %q, want %q", h.Protocol, "NVMe")
	}
	if h.SmartStatus != "Not Supported" {
		t.Errorf("SmartStatus = %q, want %q", h.SmartStatus, "Not Supported")
	}
	if h.Model != "" {
		t.Errorf("Model = %q, want empty", h.Model)
	}
}

func TestParseSizeBytes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int64
	}{
		{
			name:  "standard format",
			input: "500.1 GB (500107862016 Bytes)",
			want:  500107862016,
		},
		{
			name:  "terabyte",
			input: "1.0 TB (1000204886016 Bytes)",
			want:  1000204886016,
		},
		{
			name:  "no bytes",
			input: "500.1 GB",
			want:  0,
		},
		{
			name:  "empty",
			input: "",
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSizeBytes(tt.input)
			if got != tt.want {
				t.Errorf("parseSizeBytes(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestEnrichWithNVMe(t *testing.T) {
	h := &Health{
		Device:      "disk0",
		SmartStatus: "unknown",
		WearLevel:   "unavailable",
		DataWritten: "unavailable",
	}

	nvmeJSON := []byte(`{
		"SPNVMeDataType": [
			{
				"_items": [
					{
						"_name": "APPLE SSD AP0512Q",
						"device_model": "APPLE SSD AP0512Q",
						"spnvme_wearleveling": "1%",
						"spnvme_byteswritten": "15.23 TB",
						"spnvme_smart_status": "Verified"
					}
				]
			}
		]
	}`)

	enrichWithNVMe(h, nvmeJSON)

	if h.WearLevel != "1%" {
		t.Errorf("WearLevel = %q, want %q", h.WearLevel, "1%")
	}
	if h.DataWritten != "15.23 TB" {
		t.Errorf("DataWritten = %q, want %q", h.DataWritten, "15.23 TB")
	}
	if h.SmartStatus != "Verified" {
		t.Errorf("SmartStatus = %q, want %q", h.SmartStatus, "Verified")
	}
}

func TestEnrichWithNVMeInvalidJSON(t *testing.T) {
	h := &Health{
		Device:      "disk0",
		SmartStatus: "original",
		WearLevel:   "unavailable",
	}

	enrichWithNVMe(h, []byte("not json"))

	// Should not modify anything on invalid JSON.
	if h.SmartStatus != "original" {
		t.Errorf("SmartStatus should be unchanged, got %q", h.SmartStatus)
	}
}

func TestParseIOStat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantMBs float64
		wantTPS float64
		wantErr bool
	}{
		{
			name: "two samples",
			input: `              disk0
    KB/t  tps  MB/s
   24.00   10  0.23
   16.00   25  1.50
`,
			wantMBs: 1.50,
			wantTPS: 25,
		},
		{
			name: "single sample",
			input: `              disk0
    KB/t  tps  MB/s
   24.00   10  0.23
`,
			wantMBs: 0.23,
			wantTPS: 10,
		},
		{
			name:    "empty output",
			input:   "",
			wantErr: true,
		},
		{
			name: "headers only",
			input: `              disk0
    KB/t  tps  MB/s
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseIOStat(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ReadMBs != tt.wantMBs {
				t.Errorf("ReadMBs = %f, want %f", got.ReadMBs, tt.wantMBs)
			}
			if got.ReadIOPS != tt.wantTPS {
				t.Errorf("ReadIOPS = %f, want %f", got.ReadIOPS, tt.wantTPS)
			}
		})
	}
}

func TestParseIOStatLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantErr bool
	}{
		{name: "valid", line: "16.00 25 1.50"},
		{name: "too few fields", line: "16.00", wantErr: true},
		{name: "invalid tps", line: "16.00 abc 1.50", wantErr: true},
		{name: "invalid mbs", line: "16.00 25 abc", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseIOStatLine(tt.line)
			if tt.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestDiskHealthJSONRoundTrip(t *testing.T) {
	h := Health{
		Device:      "disk0",
		Model:       "APPLE SSD AP0512Q",
		Protocol:    "Apple Fabric",
		SizeBytes:   500107862016,
		SizeHuman:   "500.1 GB",
		WearLevel:   "1%",
		DataWritten: "15.23 TB",
		SmartStatus: "Verified",
	}

	data, err := json.Marshal(h)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var loaded Health
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if loaded.Model != h.Model {
		t.Errorf("Model = %q, want %q", loaded.Model, h.Model)
	}
	if loaded.SizeBytes != h.SizeBytes {
		t.Errorf("SizeBytes = %d, want %d", loaded.SizeBytes, h.SizeBytes)
	}
}

func TestIOStatsJSONRoundTrip(t *testing.T) {
	stats := IOStats{
		ReadMBs:   1.5,
		WriteMBs:  0.8,
		ReadIOPS:  250,
		WriteIOPS: 120,
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var loaded IOStats
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if loaded.ReadMBs != stats.ReadMBs {
		t.Errorf("ReadMBs = %f, want %f", loaded.ReadMBs, stats.ReadMBs)
	}
}

func TestDiskHistorySaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	histPath := filepath.Join(tmpDir, "disk-history.json")

	snapshots := []HealthSnapshot{
		{
			Timestamp:   time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
			Model:       "APPLE SSD AP0512Q",
			SmartStatus: "Verified",
			WearLevel:   "1%",
			DataWritten: "10 TB",
			SizeBytes:   500107862016,
		},
		{
			Timestamp:   time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC),
			Model:       "APPLE SSD AP0512Q",
			SmartStatus: "Verified",
			WearLevel:   "2%",
			DataWritten: "12 TB",
			SizeBytes:   500107862016,
		},
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

	var loaded []HealthSnapshot
	if err := json.Unmarshal(readData, &loaded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(loaded))
	}

	if loaded[0].WearLevel != "1%" {
		t.Errorf("first WearLevel = %q, want %q", loaded[0].WearLevel, "1%")
	}
	if loaded[1].WearLevel != "2%" {
		t.Errorf("second WearLevel = %q, want %q", loaded[1].WearLevel, "2%")
	}
}

func TestDiskFilterHistory(t *testing.T) {
	now := time.Now().UTC()
	snapshots := []HealthSnapshot{
		{Timestamp: now.Add(-48 * time.Hour), Model: "old"},
		{Timestamp: now.Add(-12 * time.Hour), Model: "recent"},
		{Timestamp: now.Add(-1 * time.Hour), Model: "new"},
	}

	filtered := FilterHistory(snapshots, 24*time.Hour)
	if len(filtered) != 2 {
		t.Errorf("expected 2 entries within 24h, got %d", len(filtered))
	}

	filtered = FilterHistory(snapshots, 2*time.Hour)
	if len(filtered) != 1 {
		t.Errorf("expected 1 entry within 2h, got %d", len(filtered))
	}
}
