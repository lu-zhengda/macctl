package preset

import "testing"

func TestBuiltinPresets(t *testing.T) {
	presets := BuiltinPresets()
	if len(presets) == 0 {
		t.Fatal("expected at least one builtin preset")
	}

	names := map[string]bool{}
	for _, p := range presets {
		if p.Name == "" {
			t.Error("preset has empty name")
		}
		if p.Description == "" {
			t.Errorf("preset %q has empty description", p.Name)
		}
		if len(p.Actions) == 0 {
			t.Errorf("preset %q has no actions", p.Name)
		}
		names[p.Name] = true
	}

	expected := []string{"deep-work", "meeting", "present", "chill", "battery-saver"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("missing expected preset: %s", name)
		}
	}
}

func TestGet(t *testing.T) {
	p := Get("deep-work")
	if p == nil {
		t.Fatal("expected to find 'deep-work' preset")
	}
	if p.Name != "deep-work" {
		t.Errorf("Name = %q, want %q", p.Name, "deep-work")
	}

	p = Get("DEEP-WORK")
	if p == nil {
		t.Fatal("expected case-insensitive match")
	}

	p = Get("nonexistent")
	if p != nil {
		t.Error("expected nil for nonexistent preset")
	}
}

func TestDryRun(t *testing.T) {
	p := Get("deep-work")
	if p == nil {
		t.Fatal("expected to find 'deep-work' preset")
	}

	results := DryRun(p)
	if len(results) != len(p.Actions) {
		t.Errorf("expected %d results, got %d", len(p.Actions), len(results))
	}

	for _, r := range results {
		if !r.Success {
			t.Errorf("dry run should always succeed, got failure for %v", r.Action)
		}
		if r.Message == "" {
			t.Error("expected non-empty message")
		}
	}
}

func TestDescribeAction(t *testing.T) {
	tests := []struct {
		name   string
		action Action
		want   string
	}{
		{
			name:   "with args",
			action: Action{Domain: "display", Command: "brightness", Args: []string{"50"}},
			want:   "[display] brightness 50",
		},
		{
			name:   "without args",
			action: Action{Domain: "focus", Command: "off"},
			want:   "[focus] off",
		},
		{
			name:   "multiple args",
			action: Action{Domain: "audio", Command: "mute", Args: []string{"on"}},
			want:   "[audio] mute on",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := describeAction(tt.action)
			if got != tt.want {
				t.Errorf("describeAction() = %q, want %q", got, tt.want)
			}
		})
	}
}
