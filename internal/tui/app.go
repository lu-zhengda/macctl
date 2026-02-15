package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/lu-zhengda/macctl/internal/audio"
	"github.com/lu-zhengda/macctl/internal/display"
	"github.com/lu-zhengda/macctl/internal/focus"
	"github.com/lu-zhengda/macctl/internal/power"
)

type tickMsg time.Time

type statusMsg struct {
	battery *power.Status
	health  *power.Health
	thermal *power.ThermalInfo
	volume  *audio.VolumeInfo
	output  string
	focus   *focus.Status
	displays []display.Info
	err     error
}

// keyMap defines key bindings for the TUI.
type keyMap struct {
	Quit    key.Binding
	Refresh key.Binding
	Help    key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Refresh, k.Quit, k.Help}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Refresh, k.Quit, k.Help},
	}
}

// Model is the Bubble Tea model for macctl.
type Model struct {
	version  string
	keys     keyMap
	help     help.Model
	width    int
	height   int
	battery  *power.Status
	health   *power.Health
	thermal  *power.ThermalInfo
	volume   *audio.VolumeInfo
	output   string
	focus    *focus.Status
	displays []display.Info
	showHelp bool
	err      error
}

// New creates a new TUI model.
func New(version string) Model {
	return Model{
		version: version,
		keys:    newKeyMap(),
		help:    help.New(),
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func fetchStatus() tea.Cmd {
	return func() tea.Msg {
		msg := statusMsg{}

		// Fetch all status concurrently would be ideal, but for simplicity
		// we do it sequentially. Each call is fast since it's local.
		bat, err := power.GetStatus()
		if err != nil {
			msg.err = err
			return msg
		}
		msg.battery = bat

		health, _ := power.GetHealth()
		msg.health = health

		thermal, _ := power.GetThermal()
		msg.thermal = thermal

		vol, _ := audio.GetVolume()
		msg.volume = vol

		out, _ := audio.GetCurrentOutput()
		msg.output = out

		foc, _ := focus.GetStatus()
		msg.focus = foc

		disps, _ := display.List()
		msg.displays = disps

		return msg
	}
}

// Init initializes the TUI.
func (m Model) Init() tea.Cmd {
	return tea.Batch(fetchStatus(), tickCmd())
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		return m, nil

	case tickMsg:
		return m, tea.Batch(fetchStatus(), tickCmd())

	case statusMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.battery = msg.battery
		m.health = msg.health
		m.thermal = msg.thermal
		m.volume = msg.volume
		m.output = msg.output
		m.focus = msg.focus
		m.displays = msg.displays
		m.err = nil
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showHelp {
		m.showHelp = false
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Refresh):
		return m, fetchStatus()
	case key.Matches(msg, m.keys.Help):
		m.showHelp = true
	}

	return m, nil
}

// View renders the TUI.
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Title.
	b.WriteString(titleStyle.Render(fmt.Sprintf("macctl %s", m.version)))
	b.WriteString("  ")
	b.WriteString(dimStyle.Render("macOS Environment Controller"))
	b.WriteString("\n\n")

	// Help view.
	if m.showHelp {
		b.WriteString(m.help.View(m.keys))
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("Press any key to return"))
		return b.String()
	}

	// Battery section.
	b.WriteString(sectionStyle.Render("Battery"))
	b.WriteString("\n")
	if m.battery != nil {
		b.WriteString(renderBatteryGauge(m.battery))
		b.WriteString("\n")
	} else {
		b.WriteString(dimStyle.Render("  loading..."))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Health section.
	if m.health != nil {
		b.WriteString(fmt.Sprintf("  Health: %.0f%% (%s)  Cycles: %d\n",
			m.health.HealthPercent, m.health.Condition, m.health.CycleCount))
	}

	// Thermal section.
	if m.thermal != nil {
		b.WriteString(fmt.Sprintf("  Thermal: %s  %s\n", m.thermal.PressureLevel, m.thermal.CPUTemp))
	}
	b.WriteString("\n")

	// Display section.
	b.WriteString(sectionStyle.Render("Display"))
	b.WriteString("\n")
	if len(m.displays) > 0 {
		for _, d := range m.displays {
			main := ""
			if d.Main {
				main = " (main)"
			}
			b.WriteString(fmt.Sprintf("  %s%s  %s\n", d.Name, main, d.Resolution))
		}
	} else {
		b.WriteString(dimStyle.Render("  loading..."))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Audio section.
	b.WriteString(sectionStyle.Render("Audio"))
	b.WriteString("\n")
	if m.volume != nil {
		muteStr := ""
		if m.volume.Muted {
			muteStr = " [MUTED]"
		}
		b.WriteString(fmt.Sprintf("  Output: %s\n", m.output))
		b.WriteString(fmt.Sprintf("  Volume: %s%s\n", renderVolumeBar(m.volume.OutputVolume), muteStr))
	} else {
		b.WriteString(dimStyle.Render("  loading..."))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Focus section.
	b.WriteString(sectionStyle.Render("Focus"))
	b.WriteString("\n")
	if m.focus != nil {
		if m.focus.Active {
			b.WriteString(fmt.Sprintf("  %s (%s)\n",
				critStyle.Render("ACTIVE"), m.focus.Mode))
		} else {
			b.WriteString(fmt.Sprintf("  %s\n", goodStyle.Render("off")))
		}
	} else {
		b.WriteString(dimStyle.Render("  loading..."))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Error.
	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	}

	// Help.
	b.WriteString(m.help.View(m.keys))

	return b.String()
}

func renderBatteryGauge(s *power.Status) string {
	var b strings.Builder
	pct := s.Percent
	barLen := 20
	filled := pct * barLen / 100

	b.WriteString("  [")
	for i := 0; i < barLen; i++ {
		if i < filled {
			switch {
			case pct <= 20:
				b.WriteString(critStyle.Render(string(gaugeChars[3])))
			case pct <= 50:
				b.WriteString(warnStyle.Render(string(gaugeChars[3])))
			default:
				b.WriteString(goodStyle.Render(string(gaugeChars[3])))
			}
		} else {
			b.WriteString(dimStyle.Render(string(gaugeChars[0])))
		}
	}
	b.WriteString("] ")

	// Percentage.
	pctStr := fmt.Sprintf("%d%%", pct)
	switch {
	case pct <= 20:
		b.WriteString(critStyle.Render(pctStr))
	case pct <= 50:
		b.WriteString(warnStyle.Render(pctStr))
	default:
		b.WriteString(goodStyle.Render(pctStr))
	}

	// Charging status.
	if s.IsCharging {
		b.WriteString(statusStyle.Render(" charging"))
	} else if s.ExternalConnected {
		b.WriteString(statusStyle.Render(" on AC"))
	}

	// Time remaining.
	if s.TimeRemaining != "" && s.TimeRemaining != "unknown" {
		b.WriteString(dimStyle.Render(fmt.Sprintf(" (%s)", s.TimeRemaining)))
	}

	return b.String()
}

func renderVolumeBar(vol int) string {
	var b strings.Builder
	barLen := 20
	filled := vol * barLen / 100

	for i := 0; i < barLen; i++ {
		if i < filled {
			b.WriteString(statusStyle.Render(string(gaugeChars[3])))
		} else {
			b.WriteString(dimStyle.Render(string(gaugeChars[0])))
		}
	}
	b.WriteString(fmt.Sprintf(" %d%%", vol))
	return b.String()
}
