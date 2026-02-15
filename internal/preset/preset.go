package preset

import (
	"fmt"
	"strings"

	"github.com/lu-zhengda/macctl/internal/audio"
	"github.com/lu-zhengda/macctl/internal/display"
	"github.com/lu-zhengda/macctl/internal/focus"
	"github.com/lu-zhengda/macctl/internal/power"
)

// Preset defines a compound action preset.
type Preset struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Actions     []Action `json:"actions"`
}

// Action represents a single action within a preset.
type Action struct {
	Domain  string   `json:"domain"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// Result holds the result of applying a single action.
type Result struct {
	Action  Action `json:"action"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// BuiltinPresets returns all built-in presets.
func BuiltinPresets() []Preset {
	return []Preset{
		{
			Name:        "deep-work",
			Description: "Focus on, display brightness 50%, audio mute",
			Actions: []Action{
				{Domain: "focus", Command: "on", Args: []string{"dnd"}},
				{Domain: "display", Command: "brightness", Args: []string{"50"}},
				{Domain: "audio", Command: "mute", Args: []string{"on"}},
			},
		},
		{
			Name:        "meeting",
			Description: "Focus on (allow calls), audio unmute",
			Actions: []Action{
				{Domain: "focus", Command: "on", Args: []string{"dnd"}},
				{Domain: "audio", Command: "mute", Args: []string{"off"}},
			},
		},
		{
			Name:        "present",
			Description: "Focus on, display brightness 100%",
			Actions: []Action{
				{Domain: "focus", Command: "on", Args: []string{"dnd"}},
				{Domain: "display", Command: "brightness", Args: []string{"100"}},
			},
		},
		{
			Name:        "chill",
			Description: "Focus off, Night Shift on, display brightness 40%, audio volume 30%",
			Actions: []Action{
				{Domain: "focus", Command: "off"},
				{Domain: "display", Command: "nightshift", Args: []string{"on"}},
				{Domain: "display", Command: "brightness", Args: []string{"40"}},
				{Domain: "audio", Command: "volume", Args: []string{"30"}},
			},
		},
		{
			Name:        "battery-saver",
			Description: "Display brightness 30%, show power hogs",
			Actions: []Action{
				{Domain: "display", Command: "brightness", Args: []string{"30"}},
				{Domain: "power", Command: "hogs"},
			},
		},
	}
}

// Get returns a preset by name, or nil if not found.
func Get(name string) *Preset {
	for _, p := range BuiltinPresets() {
		if strings.EqualFold(p.Name, name) {
			return &p
		}
	}
	return nil
}

// Apply executes all actions in a preset.
func Apply(p *Preset) []Result {
	var results []Result
	for _, action := range p.Actions {
		result := executeAction(action)
		results = append(results, result)
	}
	return results
}

// DryRun returns descriptions of what each action would do.
func DryRun(p *Preset) []Result {
	var results []Result
	for _, action := range p.Actions {
		results = append(results, Result{
			Action:  action,
			Success: true,
			Message: describeAction(action),
		})
	}
	return results
}

func executeAction(a Action) Result {
	var err error

	switch a.Domain {
	case "focus":
		err = executeFocusAction(a)
	case "display":
		err = executeDisplayAction(a)
	case "audio":
		err = executeAudioAction(a)
	case "power":
		return executePowerAction(a)
	default:
		return Result{Action: a, Success: false, Message: fmt.Sprintf("unknown domain: %s", a.Domain)}
	}

	if err != nil {
		return Result{Action: a, Success: false, Message: err.Error()}
	}
	return Result{Action: a, Success: true, Message: describeAction(a) + " - done"}
}

func executeFocusAction(a Action) error {
	switch a.Command {
	case "on":
		mode := ""
		if len(a.Args) > 0 {
			mode = a.Args[0]
		}
		return focus.Enable(mode)
	case "off":
		return focus.Disable()
	default:
		return fmt.Errorf("unknown focus command: %s", a.Command)
	}
}

func executeDisplayAction(a Action) error {
	switch a.Command {
	case "brightness":
		if len(a.Args) == 0 {
			return fmt.Errorf("brightness level required")
		}
		level := 0
		_, err := fmt.Sscanf(a.Args[0], "%d", &level)
		if err != nil {
			return fmt.Errorf("invalid brightness level: %w", err)
		}
		return display.SetBrightness(level)
	case "nightshift":
		if len(a.Args) == 0 {
			return fmt.Errorf("nightshift state required (on/off)")
		}
		return display.SetNightShift(a.Args[0] == "on")
	default:
		return fmt.Errorf("unknown display command: %s", a.Command)
	}
}

func executeAudioAction(a Action) error {
	switch a.Command {
	case "volume":
		if len(a.Args) == 0 {
			return fmt.Errorf("volume level required")
		}
		level := 0
		_, err := fmt.Sscanf(a.Args[0], "%d", &level)
		if err != nil {
			return fmt.Errorf("invalid volume level: %w", err)
		}
		return audio.SetVolume(level)
	case "mute":
		if len(a.Args) == 0 {
			return audio.ToggleMute()
		}
		return audio.SetMute(a.Args[0] == "on")
	default:
		return fmt.Errorf("unknown audio command: %s", a.Command)
	}
}

func executePowerAction(a Action) Result {
	switch a.Command {
	case "hogs":
		hogs, err := power.GetEnergyHogs(5)
		if err != nil {
			return Result{Action: a, Success: false, Message: err.Error()}
		}
		var lines []string
		lines = append(lines, "Top energy consumers:")
		for _, h := range hogs {
			lines = append(lines, fmt.Sprintf("  PID %d: %s (%.1f%% CPU)", h.PID, h.Command, h.CPU))
		}
		return Result{Action: a, Success: true, Message: strings.Join(lines, "\n")}
	default:
		return Result{Action: a, Success: false, Message: fmt.Sprintf("unknown power command: %s", a.Command)}
	}
}

func describeAction(a Action) string {
	args := strings.Join(a.Args, " ")
	if args != "" {
		return fmt.Sprintf("[%s] %s %s", a.Domain, a.Command, args)
	}
	return fmt.Sprintf("[%s] %s", a.Domain, a.Command)
}
