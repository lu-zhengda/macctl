package audio

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Device holds audio device information.
type Device struct {
	Name   string `json:"name"`
	Type   string `json:"type"` // "input" or "output"
	Active bool   `json:"active"`
}

// VolumeInfo holds volume information.
type VolumeInfo struct {
	OutputVolume int  `json:"output_volume"`
	InputVolume  int  `json:"input_volume"`
	Muted        bool `json:"muted"`
}

// ListDevices returns all audio input and output devices.
func ListDevices() ([]Device, error) {
	out, err := exec.Command("system_profiler", "SPAudioDataType", "-json").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get audio device info: %w", err)
	}

	return parseAudioJSON(out)
}

// GetVolume returns the current volume settings.
func GetVolume() (*VolumeInfo, error) {
	out, err := exec.Command("osascript", "-e", "get volume settings").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get volume settings: %w", err)
	}

	return parseVolumeSettings(string(out))
}

// SetVolume sets the output volume (0-100).
func SetVolume(level int) error {
	if level < 0 || level > 100 {
		return fmt.Errorf("volume must be between 0 and 100")
	}
	_, err := exec.Command("osascript", "-e",
		fmt.Sprintf("set volume output volume %d", level)).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set volume: %w", err)
	}
	return nil
}

// SetMute controls the mute state.
func SetMute(mute bool) error {
	state := "true"
	if !mute {
		state = "false"
	}
	_, err := exec.Command("osascript", "-e",
		fmt.Sprintf("set volume output muted %s", state)).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set mute: %w", err)
	}
	return nil
}

// ToggleMute toggles the mute state.
func ToggleMute() error {
	vol, err := GetVolume()
	if err != nil {
		return fmt.Errorf("failed to get current mute state: %w", err)
	}
	return SetMute(!vol.Muted)
}

// GetCurrentOutput returns the name of the current output device.
func GetCurrentOutput() (string, error) {
	// Try SwitchAudioSource if available.
	if _, err := exec.LookPath("SwitchAudioSource"); err == nil {
		out, err := exec.Command("SwitchAudioSource", "-c").Output()
		if err == nil {
			return strings.TrimSpace(string(out)), nil
		}
	}

	// Fallback: parse system_profiler output.
	devices, err := ListDevices()
	if err != nil {
		return "", err
	}
	for _, d := range devices {
		if d.Type == "output" && d.Active {
			return d.Name, nil
		}
	}
	return "unknown", nil
}

// GetCurrentInput returns the name of the current input device.
func GetCurrentInput() (string, error) {
	if _, err := exec.LookPath("SwitchAudioSource"); err == nil {
		out, err := exec.Command("SwitchAudioSource", "-c", "-t", "input").Output()
		if err == nil {
			return strings.TrimSpace(string(out)), nil
		}
	}

	devices, err := ListDevices()
	if err != nil {
		return "", err
	}
	for _, d := range devices {
		if d.Type == "input" && d.Active {
			return d.Name, nil
		}
	}
	return "unknown", nil
}

// SetOutput switches the output device by name.
func SetOutput(name string) error {
	if _, err := exec.LookPath("SwitchAudioSource"); err == nil {
		_, err := exec.Command("SwitchAudioSource", "-s", name).CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to switch output device: %w", err)
		}
		return nil
	}
	return fmt.Errorf("SwitchAudioSource not installed (brew install switchaudio-osx)")
}

// SetInput switches the input device by name.
func SetInput(name string) error {
	if _, err := exec.LookPath("SwitchAudioSource"); err == nil {
		_, err := exec.Command("SwitchAudioSource", "-s", name, "-t", "input").CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to switch input device: %w", err)
		}
		return nil
	}
	return fmt.Errorf("SwitchAudioSource not installed (brew install switchaudio-osx)")
}

func parseVolumeSettings(output string) (*VolumeInfo, error) {
	info := &VolumeInfo{}
	output = strings.TrimSpace(output)

	// Format: "output volume:50, input volume:75, alert volume:100, output muted:false"
	parts := strings.Split(output, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, ":", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])

		switch key {
		case "output volume":
			v, err := strconv.Atoi(val)
			if err == nil {
				info.OutputVolume = v
			}
		case "input volume":
			v, err := strconv.Atoi(val)
			if err == nil {
				info.InputVolume = v
			}
		case "output muted":
			info.Muted = val == "true"
		}
	}

	return info, nil
}

type systemProfilerAudio struct {
	SPAudioDataType []struct {
		Name   string `json:"_name"`
		Items  []struct {
			Name          string `json:"_name"`
			DefaultOutput string `json:"coreaudio_default_audio_output_device"`
			DefaultInput  string `json:"coreaudio_default_audio_input_device"`
			OutputSource  string `json:"coreaudio_output_source"`
			InputSource   string `json:"coreaudio_input_source"`
		} `json:"_items"`
	} `json:"SPAudioDataType"`
}

func parseAudioJSON(data []byte) ([]Device, error) {
	var sp systemProfilerAudio
	if err := json.Unmarshal(data, &sp); err != nil {
		return nil, fmt.Errorf("failed to parse audio JSON: %w", err)
	}

	var devices []Device
	for _, group := range sp.SPAudioDataType {
		for _, item := range group.Items {
			devType := "output"
			active := false

			if item.DefaultOutput == "spaudio_yes" {
				devType = "output"
				active = true
			} else if item.DefaultInput == "spaudio_yes" {
				devType = "input"
				active = true
			}

			// Determine type from name if not default.
			name := strings.ToLower(item.Name)
			if strings.Contains(name, "input") || strings.Contains(name, "microphone") || strings.Contains(name, "mic") {
				devType = "input"
			}

			devices = append(devices, Device{
				Name:   item.Name,
				Type:   devType,
				Active: active,
			})
		}
	}

	return devices, nil
}
