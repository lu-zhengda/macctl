package display

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Info holds display information.
type Info struct {
	Name        string `json:"name"`
	Resolution  string `json:"resolution"`
	RefreshRate string `json:"refresh_rate"`
	Vendor      string `json:"vendor"`
	Main        bool   `json:"main"`
}

// BrightnessInfo holds brightness level.
type BrightnessInfo struct {
	Level float64 `json:"level"`
}

// NightShiftInfo holds Night Shift status.
type NightShiftInfo struct {
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"`
}

// List returns information about connected displays.
func List() ([]Info, error) {
	out, err := exec.Command("system_profiler", "SPDisplaysDataType", "-json").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get display info: %w", err)
	}

	return parseDisplayJSON(out)
}

// GetBrightness returns the current display brightness.
func GetBrightness() (*BrightnessInfo, error) {
	// Try using osascript to get brightness.
	out, err := exec.Command("osascript", "-e", "tell application \"System Events\" to get the value of slider 1 of group 1 of group 1 of window 1 of application process \"Control Center\"").Output()
	if err != nil {
		// Fallback: try to read from ioreg.
		return getBrightnessFromIoreg()
	}
	raw := strings.TrimSpace(string(out))
	val, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return getBrightnessFromIoreg()
	}
	return &BrightnessInfo{Level: val * 100}, nil
}

// SetBrightness sets the display brightness to a value between 0 and 100.
func SetBrightness(level int) error {
	if level < 0 || level > 100 {
		return fmt.Errorf("brightness must be between 0 and 100")
	}
	// Use osascript with AppleScript to set brightness.
	script := fmt.Sprintf(`tell application "System Preferences"
	reveal anchor "displaysDisplayTab" of pane id "com.apple.preference.displays"
end tell
delay 0.5
tell application "System Events"
	tell process "System Preferences"
		set value of slider 1 of group 2 of tab group 1 of window 1 to %f
	end tell
end tell
tell application "System Preferences" to quit`, float64(level)/100.0)

	// Simpler approach: use brightness CLI if available, otherwise AppleScript.
	_, err := exec.LookPath("brightness")
	if err == nil {
		_, err = exec.Command("brightness", fmt.Sprintf("%.2f", float64(level)/100.0)).Output()
		if err != nil {
			return fmt.Errorf("failed to set brightness: %w", err)
		}
		return nil
	}

	// Try using osascript for setting brightness via System Events.
	_, err = exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return fmt.Errorf("failed to set brightness (install 'brightness' CLI for best results): %w", err)
	}
	return nil
}

// GetNightShift returns the current Night Shift status.
func GetNightShift() (*NightShiftInfo, error) {
	// Check Night Shift via CoreBrightness defaults.
	out, err := exec.Command("defaults", "read", "com.apple.CoreBrightness", "CBBlueReductionStatus").Output()
	if err != nil {
		// Night Shift info may not be available.
		return &NightShiftInfo{
			Enabled: false,
			Status:  "unavailable (cannot read CoreBrightness preferences)",
		}, nil
	}

	raw := string(out)
	enabled := strings.Contains(raw, "BlueLightReductionEnabled = 1") || strings.Contains(raw, "BlueReductionEnabled = 1")

	status := "off"
	if enabled {
		status = "on"
	}

	return &NightShiftInfo{
		Enabled: enabled,
		Status:  status,
	}, nil
}

// SetNightShift enables or disables Night Shift.
func SetNightShift(enable bool) error {
	// Night Shift can be toggled via keyboard shortcut or using the private
	// CoreBrightness framework. The most reliable approach without private
	// frameworks is using a shortcut or AppleScript.
	var script string
	if enable {
		script = `
do shell script "
defaults write com.apple.CoreBrightness CBBlueReductionStatus -dict BlueLightReductionEnabled -bool true
"
`
	} else {
		script = `
do shell script "
defaults write com.apple.CoreBrightness CBBlueReductionStatus -dict BlueLightReductionEnabled -bool false
"
`
	}

	_, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set night shift (may require System Preferences): %w", err)
	}
	return nil
}

func getBrightnessFromIoreg() (*BrightnessInfo, error) {
	out, err := exec.Command("ioreg", "-r", "-c", "AppleBacklightDisplay", "-w", "0").Output()
	if err != nil {
		return &BrightnessInfo{Level: -1}, nil
	}

	raw := string(out)
	re := regexp.MustCompile(`"brightness"\s*=\s*(\d+)`)
	m := re.FindStringSubmatch(raw)
	if len(m) > 1 {
		val, _ := strconv.ParseFloat(m[1], 64)
		// ioreg brightness is typically 0-1024.
		return &BrightnessInfo{Level: val / 1024.0 * 100.0}, nil
	}

	return &BrightnessInfo{Level: -1}, nil
}

type systemProfilerDisplay struct {
	SPDisplaysDataType []struct {
		Name   string `json:"_name"`
		Vendor string `json:"sppci_vendor"`
		Ndrvs  []struct {
			Name        string `json:"_name"`
			Resolution  string `json:"_spdisplays_resolution"`
			RefreshRate string `json:"spdisplays_refresh_rate"`
			Main        string `json:"spdisplays_main"`
		} `json:"spdisplays_ndrvs"`
	} `json:"SPDisplaysDataType"`
}

func parseDisplayJSON(data []byte) ([]Info, error) {
	var sp systemProfilerDisplay
	if err := json.Unmarshal(data, &sp); err != nil {
		// Try parsing the array directly.
		return parseDisplayJSONArray(data)
	}

	var displays []Info
	for _, gpu := range sp.SPDisplaysDataType {
		for _, d := range gpu.Ndrvs {
			displays = append(displays, Info{
				Name:        d.Name,
				Resolution:  d.Resolution,
				RefreshRate: d.RefreshRate,
				Vendor:      gpu.Vendor,
				Main:        d.Main == "spdisplays_yes",
			})
		}
	}
	return displays, nil
}

func parseDisplayJSONArray(data []byte) ([]Info, error) {
	var items []struct {
		Name   string `json:"_name"`
		Vendor string `json:"sppci_vendor"`
		Ndrvs  []struct {
			Name        string `json:"_name"`
			Resolution  string `json:"_spdisplays_resolution"`
			RefreshRate string `json:"spdisplays_refresh_rate"`
			Main        string `json:"spdisplays_main"`
		} `json:"spdisplays_ndrvs"`
	}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("failed to parse display JSON: %w", err)
	}

	var displays []Info
	for _, gpu := range items {
		for _, d := range gpu.Ndrvs {
			displays = append(displays, Info{
				Name:        d.Name,
				Resolution:  d.Resolution,
				RefreshRate: d.RefreshRate,
				Vendor:      gpu.Vendor,
				Main:        d.Main == "spdisplays_yes",
			})
		}
	}
	return displays, nil
}
