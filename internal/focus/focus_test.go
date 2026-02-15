package focus

import "testing"

func TestExtractModeName(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		want       string
	}{
		{
			name:       "default dnd",
			identifier: "com.apple.donotdisturb.mode.default",
			want:       "Do Not Disturb",
		},
		{
			name:       "sleep mode",
			identifier: "com.apple.focus.mode.sleep",
			want:       "Sleep",
		},
		{
			name:       "work mode",
			identifier: "com.apple.focus.mode.work",
			want:       "Work",
		},
		{
			name:       "personal mode",
			identifier: "com.apple.focus.mode.personal",
			want:       "Personal",
		},
		{
			name:       "sleep mode hyphenated",
			identifier: "com.apple.sleep.sleep-mode",
			want:       "Sleep",
		},
		{
			name:       "custom mode",
			identifier: "com.apple.focus.mode.custom.abc123",
			want:       "abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractModeName(tt.identifier)
			if got != tt.want {
				t.Errorf("extractModeName(%q) = %q, want %q", tt.identifier, got, tt.want)
			}
		})
	}
}

func TestParseCustomModes(t *testing.T) {
	input := []byte(`{
		"data": [
			{
				"modeConfigurations": {
					"uuid-1": {"name": "Coding"},
					"uuid-2": {"name": "Reading"}
				}
			}
		]
	}`)

	modes := parseCustomModes(input)
	if len(modes) != 2 {
		t.Fatalf("expected 2 modes, got %d", len(modes))
	}

	names := map[string]bool{}
	for _, m := range modes {
		names[m.Name] = true
		if m.Builtin {
			t.Errorf("custom mode %q should not be builtin", m.Name)
		}
	}

	if !names["Coding"] {
		t.Error("expected 'Coding' mode")
	}
	if !names["Reading"] {
		t.Error("expected 'Reading' mode")
	}
}

func TestParseCustomModesInvalid(t *testing.T) {
	modes := parseCustomModes([]byte("not valid json"))
	if modes != nil {
		t.Errorf("expected nil for invalid JSON, got %v", modes)
	}
}
