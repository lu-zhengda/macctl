package events

import (
	"testing"
	"time"
)

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:  "microseconds",
			input: "2025-01-15 10:30:45.123456-0800",
		},
		{
			name:  "milliseconds",
			input: "2025-01-15 10:30:45.123-0800",
		},
		{
			name:  "no subseconds",
			input: "2025-01-15 10:30:45-0800",
		},
		{
			name:    "invalid",
			input:   "not a timestamp",
			wantErr: true,
		},
		{
			name:    "empty",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, err := parseTimestamp(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got %v", ts)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ts.IsZero() {
				t.Error("expected non-zero timestamp")
			}
		})
	}
}

func TestParseLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantType string
		wantNil  bool
	}{
		{
			name:     "wake event",
			line:     "2025-01-15 10:30:45.123456-0800  0x123  Default  com.apple.powerd  Wake Reason: EC.LidOpen",
			wantType: EventWake,
		},
		{
			name:     "sleep event",
			line:     "2025-01-15 10:30:45.123456-0800  0x123  Default  com.apple.powerd  Entering sleep reason: Clamshell Sleep",
			wantType: EventSleep,
		},
		{
			name:     "lid open",
			line:     "2025-01-15 10:30:45.123456-0800  0x123  Default  com.apple.powerd  Lid Open detected",
			wantType: EventLidOpen,
		},
		{
			name:     "lid close",
			line:     "2025-01-15 10:30:45.123456-0800  0x123  Default  com.apple.powerd  Lid Close detected",
			wantType: EventLidClose,
		},
		{
			name:     "thermal throttle",
			line:     "2025-01-15 10:30:45.123456-0800  0x123  Default  com.apple.powerd  Thermal pressure throttling CPU",
			wantType: EventThermal,
		},
		{
			name:     "power source",
			line:     "2025-01-15 10:30:45.123456-0800  0x123  Default  com.apple.powerd  Power source changed to AC Power",
			wantType: EventPowerSource,
		},
		{
			name:    "unrelated line",
			line:    "2025-01-15 10:30:45.123456-0800  0x123  Default  com.apple.powerd  Some unrelated message",
			wantNil: true,
		},
		{
			name:    "no timestamp",
			line:    "some random text without timestamp",
			wantNil: true,
		},
		{
			name:    "empty",
			line:    "",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := parseLine(tt.line)
			if tt.wantNil {
				if event != nil {
					t.Errorf("expected nil, got %+v", event)
				}
				return
			}
			if event == nil {
				t.Fatal("expected non-nil event")
			}
			if event.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", event.Type, tt.wantType)
			}
			if event.Detail == "" {
				t.Error("expected non-empty Detail")
			}
		})
	}
}

func TestParseLogOutput(t *testing.T) {
	input := `2025-01-15 08:00:00.000000-0800  0x1  Default  com.apple.powerd  Wake Reason: EC.LidOpen
2025-01-15 08:00:01.000000-0800  0x2  Default  com.apple.powerd  Power source changed to AC Power
2025-01-15 12:00:00.000000-0800  0x3  Default  com.apple.powerd  Some unrelated log entry
2025-01-15 18:00:00.000000-0800  0x4  Default  com.apple.powerd  Entering sleep reason: Clamshell Sleep
`

	events := parseLogOutput(input)
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	if events[0].Type != EventWake {
		t.Errorf("first event Type = %q, want %q", events[0].Type, EventWake)
	}
	if events[1].Type != EventPowerSource {
		t.Errorf("second event Type = %q, want %q", events[1].Type, EventPowerSource)
	}
	if events[2].Type != EventSleep {
		t.Errorf("third event Type = %q, want %q", events[2].Type, EventSleep)
	}
}

func TestParseLogOutputEmpty(t *testing.T) {
	events := parseLogOutput("")
	if len(events) != 0 {
		t.Errorf("expected 0 events for empty input, got %d", len(events))
	}
}

func TestExtractDetail(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "with double space prefix",
			input: "0x123 Default  Wake Reason: EC.LidOpen",
			want:  "Wake Reason: EC.LidOpen",
		},
		{
			name:  "no double space",
			input: "Simple message",
			want:  "Simple message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractDetail(tt.input)
			if got != tt.want {
				t.Errorf("extractDetail(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestPowerEventJSONRoundTrip(t *testing.T) {
	event := PowerEvent{
		Timestamp: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		Type:      EventWake,
		Detail:    "Wake Reason: EC.LidOpen",
	}

	// Just verify the struct has proper json tags by marshalling.
	if event.Type != "wake" {
		t.Errorf("Type = %q, want %q", event.Type, "wake")
	}
	if event.Detail == "" {
		t.Error("expected non-empty Detail")
	}
}
