package events

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/lu-zhengda/macctl/internal/power"
)

// PowerEvent represents a power-related system event.
type PowerEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Detail    string    `json:"detail"`
}

// EventType constants for categorizing events.
const (
	EventWake          = "wake"
	EventSleep         = "sleep"
	EventLidOpen       = "lid_open"
	EventLidClose      = "lid_close"
	EventThermal       = "thermal_throttle"
	EventPowerSource   = "power_source_change"
	EventPowerUnknown  = "power_event"
)

// GetEvents queries the system log for power-related events.
func GetEvents(lastDuration string) ([]PowerEvent, error) {
	if lastDuration == "" {
		lastDuration = "24h"
	}

	out, err := exec.Command("log", "show",
		"--predicate", `subsystem == "com.apple.powerd"`,
		"--style", "compact",
		"--last", lastDuration,
	).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to query system log: %w", err)
	}

	return parseLogOutput(string(out)), nil
}

func parseLogOutput(output string) []PowerEvent {
	var events []PowerEvent

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		event := parseLine(line)
		if event != nil {
			events = append(events, *event)
		}
	}

	return events
}

// timestampRe matches the compact log timestamp format.
// Real compact format: "2025-01-15 10:30:45.123" (no timezone offset).
// Also supports legacy format with timezone: "2025-01-15 10:30:45.123456-0800".
var timestampRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}\.\d+(?:[+-]\d{4})?)\s+`)

func parseLine(line string) *PowerEvent {
	m := timestampRe.FindStringSubmatch(line)
	if len(m) < 2 {
		return nil
	}

	ts, err := parseTimestamp(m[1])
	if err != nil {
		return nil
	}

	rest := line[len(m[0]):]
	lower := strings.ToLower(rest)

	event := &PowerEvent{
		Timestamp: ts,
	}

	switch {
	case strings.Contains(lower, "wake reason") || strings.Contains(lower, "waking") ||
		strings.Contains(lower, "display wake") || strings.Contains(lower, "darkwake") || strings.Contains(lower, "fullwake"):
		event.Type = EventWake
		event.Detail = extractDetail(rest)
	case strings.Contains(lower, "sleep reason") || strings.Contains(lower, "entering sleep") ||
		strings.Contains(lower, "going to sleep") || strings.Contains(lower, "maintenance sleep") ||
		strings.Contains(lower, "sleepservice"):
		event.Type = EventSleep
		event.Detail = extractDetail(rest)
	case strings.Contains(lower, "lidopen") || strings.Contains(lower, "lid open"):
		event.Type = EventLidOpen
		event.Detail = extractDetail(rest)
	case strings.Contains(lower, "lidclose") || strings.Contains(lower, "lid close") || strings.Contains(lower, "clamshell"):
		event.Type = EventLidClose
		event.Detail = extractDetail(rest)
	case strings.Contains(lower, "thermal") && (strings.Contains(lower, "throttl") || strings.Contains(lower, "pressure")):
		event.Type = EventThermal
		event.Detail = extractDetail(rest)
	case strings.Contains(lower, "power source") || strings.Contains(lower, "ac power") || strings.Contains(lower, "battery power") ||
		strings.Contains(lower, "accpowersources"):
		event.Type = EventPowerSource
		event.Detail = extractDetail(rest)
	default:
		// Skip lines that don't match any known event type.
		return nil
	}

	return event
}

func extractDetail(s string) string {
	// Remove the process/subsystem prefix to get a cleaner detail string.
	// Compact format often has: "0x123 Default com.apple.powerd ... message"
	parts := strings.SplitN(s, "  ", 2)
	if len(parts) > 1 {
		return strings.TrimSpace(parts[1])
	}
	// Truncate long details.
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return strings.TrimSpace(s)
}

func parseTimestamp(s string) (time.Time, error) {
	layouts := []string{
		"2006-01-02 15:04:05.000000-0700",
		"2006-01-02 15:04:05.000-0700",
		"2006-01-02 15:04:05-0700",
		"2006-01-02 15:04:05.000000",
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("failed to parse timestamp: %q", s)
}

// ParseDuration delegates to power.ParseDuration for consistent duration parsing.
var ParseDuration = power.ParseDuration
