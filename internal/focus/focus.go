package focus

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Status holds the current focus mode status.
type Status struct {
	Active    bool   `json:"active"`
	Mode      string `json:"mode"`
	DnDActive bool   `json:"dnd_active"`
}

// Mode represents a configured focus mode.
type Mode struct {
	Name    string `json:"name"`
	Active  bool   `json:"active"`
	Builtin bool   `json:"builtin"`
}

// GetStatus returns the current focus/DnD status.
func GetStatus() (*Status, error) {
	s := &Status{}

	// Try reading DnD assertions file.
	home, err := os.UserHomeDir()
	if err != nil {
		return s, nil
	}

	assertionsPath := filepath.Join(home, "Library", "DoNotDisturb", "DB", "Assertions.json")
	data, err := os.ReadFile(assertionsPath)
	if err != nil {
		// File may not exist or not be readable. Try alternative methods.
		return getStatusFromDefaults()
	}

	var assertions struct {
		Data []struct {
			StoreAssertionRecords []struct {
				AssertionDetails struct {
					AssertionDetailsModeIdentifier string `json:"assertionDetailsModeIdentifier"`
				} `json:"assertionDetails"`
			} `json:"storeAssertionRecords"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &assertions); err != nil {
		return getStatusFromDefaults()
	}

	// Check if there are any active assertions.
	for _, d := range assertions.Data {
		if len(d.StoreAssertionRecords) > 0 {
			s.Active = true
			s.DnDActive = true
			for _, rec := range d.StoreAssertionRecords {
				modeID := rec.AssertionDetails.AssertionDetailsModeIdentifier
				if modeID != "" {
					s.Mode = extractModeName(modeID)
				}
			}
			break
		}
	}

	if s.Mode == "" && s.Active {
		s.Mode = "Do Not Disturb"
	}

	return s, nil
}

// Enable enables Do Not Disturb / Focus mode.
func Enable(mode string) error {
	// Use shortcuts CLI if available for specific focus modes.
	if mode != "" && mode != "dnd" {
		_, err := exec.Command("shortcuts", "run", mode).CombinedOutput()
		if err == nil {
			return nil
		}
		// Fall through to DnD if shortcuts failed.
	}

	// Enable DnD using defaults and notification center restart.
	script := `
tell application "System Events"
	tell process "Control Center"
		-- Open Control Center
		click menu bar item "Control Center" of menu bar 1
		delay 0.5
		-- Click Focus
		try
			click button "Focus" of group 1 of window "Control Center"
			delay 0.3
			click checkbox "Do Not Disturb" of scroll area 1 of group 1 of window "Control Center"
		end try
		delay 0.3
		-- Close Control Center
		key code 53
	end tell
end tell
`
	_, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to enable focus mode (may require Accessibility permissions): %w", err)
	}
	return nil
}

// Disable disables Do Not Disturb / Focus mode.
func Disable() error {
	script := `
tell application "System Events"
	tell process "Control Center"
		click menu bar item "Control Center" of menu bar 1
		delay 0.5
		try
			click button "Focus" of group 1 of window "Control Center"
			delay 0.3
			-- If Focus is active, clicking it should show options to disable
			click checkbox "Do Not Disturb" of scroll area 1 of group 1 of window "Control Center"
		end try
		delay 0.3
		key code 53
	end tell
end tell
`
	_, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to disable focus mode (may require Accessibility permissions): %w", err)
	}
	return nil
}

// ListModes returns available focus modes.
func ListModes() ([]Mode, error) {
	// Focus modes configuration is stored in a plist that may not be
	// easily accessible. Return known built-in modes and try to detect
	// configured ones.
	modes := []Mode{
		{Name: "Do Not Disturb", Builtin: true},
		{Name: "Sleep", Builtin: true},
		{Name: "Personal", Builtin: true},
		{Name: "Work", Builtin: true},
	}

	// Try to detect which one is active.
	status, err := GetStatus()
	if err == nil && status.Active {
		for i := range modes {
			if strings.EqualFold(modes[i].Name, status.Mode) {
				modes[i].Active = true
			}
		}
	}

	// Try to read user-configured focus modes from preferences.
	home, err := os.UserHomeDir()
	if err == nil {
		configPath := filepath.Join(home, "Library", "DoNotDisturb", "DB", "ModeConfigurations.json")
		data, err := os.ReadFile(configPath)
		if err == nil {
			customModes := parseCustomModes(data)
			modes = append(modes, customModes...)
		}
	}

	return modes, nil
}

func getStatusFromDefaults() (*Status, error) {
	s := &Status{}

	// Try checking via defaults.
	out, err := exec.Command("defaults", "-currentHost", "read", "com.apple.notificationcenterui", "doNotDisturb").Output()
	if err == nil {
		raw := strings.TrimSpace(string(out))
		if raw == "1" {
			s.Active = true
			s.DnDActive = true
			s.Mode = "Do Not Disturb"
		}
	}

	return s, nil
}

func extractModeName(identifier string) string {
	// Mode identifiers are like "com.apple.donotdisturb.mode.default"
	// or "com.apple.focus.mode.custom.uuid"
	parts := strings.Split(identifier, ".")
	if len(parts) > 0 {
		last := parts[len(parts)-1]
		switch last {
		case "default":
			return "Do Not Disturb"
		case "sleep", "sleep-mode":
			return "Sleep"
		case "personal":
			return "Personal"
		case "work":
			return "Work"
		default:
			// Handle hyphenated identifiers (e.g., "sleep-mode" at any position).
			lower := strings.ToLower(last)
			if strings.Contains(lower, "sleep") {
				return "Sleep"
			}
			return last
		}
	}
	return "Unknown"
}

func parseCustomModes(data []byte) []Mode {
	var config struct {
		Data []struct {
			ModeConfigurations map[string]struct {
				Name string `json:"name"`
			} `json:"modeConfigurations"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return nil
	}

	var modes []Mode
	for _, d := range config.Data {
		for _, mc := range d.ModeConfigurations {
			if mc.Name != "" {
				modes = append(modes, Mode{
					Name:    mc.Name,
					Builtin: false,
				})
			}
		}
	}

	return modes
}
