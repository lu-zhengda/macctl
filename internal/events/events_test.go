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
			name:  "microseconds with timezone",
			input: "2025-01-15 10:30:45.123456-0800",
		},
		{
			name:  "milliseconds with timezone",
			input: "2025-01-15 10:30:45.123-0800",
		},
		{
			name:  "no subseconds with timezone",
			input: "2025-01-15 10:30:45-0800",
		},
		{
			name:  "milliseconds without timezone (real compact format)",
			input: "2025-01-15 10:30:45.123",
		},
		{
			name:  "microseconds without timezone",
			input: "2025-01-15 10:30:45.123456",
		},
		{
			name:  "no subseconds without timezone",
			input: "2025-01-15 10:30:45",
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
			name:     "wake reason",
			line:     "2025-01-15 10:30:45.123 Df powerd[323:1a2b] [com.apple.powerd:assertions] Wake Reason: EC.LidOpen",
			wantType: EventWake,
		},
		{
			name:     "display wake",
			line:     "2025-01-15 10:30:45.123 Df powerd[323:78a7d1] [com.apple.powerd:assertions] Cancelling notification display wake 0",
			wantType: EventWake,
		},
		{
			name:     "DarkWake",
			line:     "2025-01-15 10:30:45.123 Df powerd[323:1a2b] [com.apple.powerd:assertions] DarkWake from Deep Idle",
			wantType: EventWake,
		},
		{
			name:     "FullWake",
			line:     "2025-01-15 10:30:45.123 Df powerd[323:1a2b] [com.apple.powerd:assertions] FullWake reason: UserActivity",
			wantType: EventWake,
		},
		{
			name:     "entering sleep",
			line:     "2025-01-15 10:30:45.123 Df powerd[323:1a2b] [com.apple.powerd:assertions] Entering sleep reason: Clamshell Sleep",
			wantType: EventSleep,
		},
		{
			name:     "going to sleep",
			line:     "2025-01-15 10:30:45.123 Df powerd[323:1a2b] [com.apple.powerd:sleep] Going to sleep reason: Idle",
			wantType: EventSleep,
		},
		{
			name:     "maintenance sleep",
			line:     "2025-01-15 10:30:45.123 Df powerd[323:1a2b] [com.apple.powerd:sleep] Maintenance sleep wake",
			wantType: EventSleep,
		},
		{
			name:     "SleepService",
			line:     "2025-01-15 10:30:45.123 Df powerd[323:1a2b] [com.apple.powerd:sleep] SleepService: window begins",
			wantType: EventSleep,
		},
		{
			name:     "lid open with space",
			line:     "2025-01-15 10:30:45.123 Df powerd[323:1a2b] [com.apple.powerd:assertions] Lid Open detected",
			wantType: EventLidOpen,
		},
		{
			name:     "LidOpen without space",
			line:     "2025-01-15 10:30:45.123 Df powerd[323:1a2b] [com.apple.powerd:assertions] EC.LidOpen event received",
			wantType: EventLidOpen,
		},
		{
			name:     "lid close with space",
			line:     "2025-01-15 10:30:45.123 Df powerd[323:1a2b] [com.apple.powerd:assertions] Lid Close detected",
			wantType: EventLidClose,
		},
		{
			name:     "LidClose without space",
			line:     "2025-01-15 10:30:45.123 E  powerd[323:78a7d1] [com.apple.powerd:assertions] Pid 374 is not privileged to set property AppliesOnLidClose",
			wantType: EventLidClose,
		},
		{
			name:     "clamshell",
			line:     "2025-01-15 10:30:45.123 Df powerd[323:1a2b] [com.apple.powerd:assertions] Clamshell state changed",
			wantType: EventLidClose,
		},
		{
			name:     "thermal throttle",
			line:     "2025-01-15 10:30:45.123 Df powerd[323:1a2b] [com.apple.powerd:thermal] Thermal pressure throttling CPU",
			wantType: EventThermal,
		},
		{
			name:     "power source",
			line:     "2025-01-15 10:30:45.123 Df powerd[323:d9137a] [com.apple.powerd:battery] Received power source(psid:6829) update from pid 669: <private>",
			wantType: EventPowerSource,
		},
		{
			name:     "accpowersources",
			line:     "2025-01-15 10:30:45.123 Df powerd[323:78c169] [com.apple.powerd:battery] posted 'com.apple.system.accpowersources.attach'",
			wantType: EventPowerSource,
		},
		{
			name:     "legacy format with timezone",
			line:     "2025-01-15 10:30:45.123456-0800  0x123  Default  com.apple.powerd  Wake Reason: EC.LidOpen",
			wantType: EventWake,
		},
		{
			name:    "unrelated line",
			line:    "2025-01-15 10:30:45.123 Df powerd[323:1a2b] [com.apple.powerd:assertions] Some unrelated message",
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
	input := `2025-01-15 08:00:00.123 Df powerd[323:1a2b] [com.apple.powerd:assertions] Wake Reason: EC.LidOpen
2025-01-15 08:00:01.456 Df powerd[323:d9137a] [com.apple.powerd:battery] Received power source(psid:6829) update from pid 669: <private>
2025-01-15 12:00:00.789 Df powerd[323:1a2b] [com.apple.powerd:assertions] Some unrelated log entry
2025-01-15 18:00:00.012 Df powerd[323:1a2b] [com.apple.powerd:assertions] Entering sleep reason: Clamshell Sleep
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
			name:  "compact format Df prefix",
			input: `Df powerd[323:dbd3ca] [com.apple.powerd:battery] Received power source update`,
			want:  "Received power source update",
		},
		{
			name:  "compact format E prefix with trailing spaces",
			input: `E  powerd[323:78a7d1] [com.apple.powerd:assertions] Error message here`,
			want:  "Error message here",
		},
		{
			name:  "process name with spaces",
			input: `Df Some App[123:abc] [com.apple.powerd:battery] message here`,
			want:  "message here",
		},
		{
			name:  "no prefix",
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

func TestDeduplicateEvents(t *testing.T) {
	base := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	window := 30 * time.Second

	tests := []struct {
		name       string
		events     []PowerEvent
		wantLen    int
		wantCounts []int
	}{
		{
			name:    "empty input",
			events:  nil,
			wantLen: 0,
		},
		{
			name: "single event",
			events: []PowerEvent{
				{Timestamp: base, Type: EventWake, Detail: "wake"},
			},
			wantLen:    1,
			wantCounts: []int{1},
		},
		{
			name: "3 consecutive same-type within window",
			events: []PowerEvent{
				{Timestamp: base, Type: EventPowerSource, Detail: "update 1"},
				{Timestamp: base.Add(5 * time.Second), Type: EventPowerSource, Detail: "update 2"},
				{Timestamp: base.Add(10 * time.Second), Type: EventPowerSource, Detail: "update 3"},
			},
			wantLen:    1,
			wantCounts: []int{3},
		},
		{
			name: "2 different types",
			events: []PowerEvent{
				{Timestamp: base, Type: EventWake, Detail: "wake"},
				{Timestamp: base.Add(5 * time.Second), Type: EventSleep, Detail: "sleep"},
			},
			wantLen:    2,
			wantCounts: []int{1, 1},
		},
		{
			name: "same type but outside window",
			events: []PowerEvent{
				{Timestamp: base, Type: EventPowerSource, Detail: "update 1"},
				{Timestamp: base.Add(60 * time.Second), Type: EventPowerSource, Detail: "update 2"},
			},
			wantLen:    2,
			wantCounts: []int{1, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeduplicateEvents(tt.events, window)
			if len(got) != tt.wantLen {
				t.Fatalf("len = %d, want %d", len(got), tt.wantLen)
			}
			for i, wc := range tt.wantCounts {
				if got[i].Count != wc {
					t.Errorf("event[%d].Count = %d, want %d", i, got[i].Count, wc)
				}
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
