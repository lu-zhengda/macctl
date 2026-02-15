package power

import "testing"

func TestExtractInt(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		pattern string
		want    int
	}{
		{
			name:    "current capacity",
			input:   `"CurrentCapacity" = 4523`,
			pattern: `"CurrentCapacity"\s*=\s*(\d+)`,
			want:    4523,
		},
		{
			name:    "no match",
			input:   `"SomeOtherKey" = 100`,
			pattern: `"CurrentCapacity"\s*=\s*(\d+)`,
			want:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractInt(tt.input, tt.pattern)
			if got != tt.want {
				t.Errorf("extractInt() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestExtractBool(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		pattern string
		want    bool
	}{
		{
			name:    "charging yes",
			input:   `"IsCharging" = Yes`,
			pattern: `"IsCharging"\s*=\s*(Yes|No)`,
			want:    true,
		},
		{
			name:    "charging no",
			input:   `"IsCharging" = No`,
			pattern: `"IsCharging"\s*=\s*(Yes|No)`,
			want:    false,
		},
		{
			name:    "no match",
			input:   `"Other" = Yes`,
			pattern: `"IsCharging"\s*=\s*(Yes|No)`,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBool(tt.input, tt.pattern)
			if got != tt.want {
				t.Errorf("extractBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractTimeRemaining(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "time remaining",
			input: `Now drawing from 'Battery Power'\n -InternalBattery-0 (id=1234)\t85%; discharging; 3:45 remaining present: true`,
			want:  "3:45",
		},
		{
			name:  "fully charged",
			input: `Now drawing from 'AC Power'\n -InternalBattery-0 (id=1234)\t100%; charged; 0:00 remaining present: true`,
			want:  "fully charged",
		},
		{
			name:  "no estimate",
			input: `Now drawing from 'Battery Power'\n -InternalBattery-0 (id=1234)\t85%; discharging; (no estimate) present: true`,
			want:  "calculating",
		},
		{
			name:  "unknown",
			input: `some unknown output`,
			want:  "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTimeRemaining(tt.input)
			if got != tt.want {
				t.Errorf("extractTimeRemaining() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseAssertions(t *testing.T) {
	input := `Assertion status system-wide:
   BackgroundTask                 1
   PreventUserIdleDisplaySleep    1
Listed by owning process:
   pid 312(coreaudiod): [0x000012340000004e] 00:12:34 PreventUserIdleSystemSleep named: "com.apple.audio.context"
   pid 456(Safari): [0x000012340000005f] 01:23:45 PreventUserIdleDisplaySleep named: "Playing video"
`
	assertions := parseAssertions(input)
	if len(assertions) != 2 {
		t.Fatalf("expected 2 assertions, got %d", len(assertions))
	}

	if assertions[0].PID != 312 {
		t.Errorf("first assertion PID = %d, want 312", assertions[0].PID)
	}
	if assertions[0].Name != "coreaudiod" {
		t.Errorf("first assertion Name = %q, want %q", assertions[0].Name, "coreaudiod")
	}
	if assertions[0].Type != "PreventUserIdleSystemSleep" {
		t.Errorf("first assertion Type = %q, want %q", assertions[0].Type, "PreventUserIdleSystemSleep")
	}
	if assertions[0].Reason != "com.apple.audio.context" {
		t.Errorf("first assertion Reason = %q, want %q", assertions[0].Reason, "com.apple.audio.context")
	}

	if assertions[1].PID != 456 {
		t.Errorf("second assertion PID = %d, want 456", assertions[1].PID)
	}
	if assertions[1].Reason != "Playing video" {
		t.Errorf("second assertion Reason = %q, want %q", assertions[1].Reason, "Playing video")
	}
}

func TestParseEnergyHogs(t *testing.T) {
	input := `  PID  %CPU COMM
  123  45.2 /usr/bin/some_process
  456  12.3 /Applications/App.app/Contents/MacOS/App
  789   5.1 /usr/sbin/daemon
  101   2.0 /bin/bash
`
	hogs := parseEnergyHogs(input, 3)
	if len(hogs) != 3 {
		t.Fatalf("expected 3 hogs, got %d", len(hogs))
	}

	if hogs[0].PID != 123 {
		t.Errorf("first hog PID = %d, want 123", hogs[0].PID)
	}
	if hogs[0].CPU != 45.2 {
		t.Errorf("first hog CPU = %f, want 45.2", hogs[0].CPU)
	}
	if hogs[0].Command != "some_process" {
		t.Errorf("first hog Command = %q, want %q", hogs[0].Command, "some_process")
	}

	if hogs[1].Command != "App" {
		t.Errorf("second hog Command = %q, want %q", hogs[1].Command, "App")
	}
}
