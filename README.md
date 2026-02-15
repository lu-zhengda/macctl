# macctl

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Platform: macOS](https://img.shields.io/badge/Platform-macOS-lightgrey.svg)](https://github.com/lu-zhengda/macctl)
[![Homebrew](https://img.shields.io/badge/Homebrew-tap-orange.svg)](https://github.com/lu-zhengda/homebrew-tap)

macOS environment controller — power, display, audio, and focus management from the terminal.

## Install

```bash
brew tap lu-zhengda/tap
brew install macctl
```

## Usage

Launch without subcommands for an interactive TUI, or use subcommands directly.

### Power

```
$ macctl power status
Battery:       100%
State:         on AC power
Time:          fully charged
Cycles:        351
Temperature:   30.1 C
Capacity:      5059 / 5209 mAh

$ macctl power health
Health:          85.7%
Condition:       Normal
Design Capacity: 6075 mAh
Max Capacity:    5209 mAh
Cycle Count:     351

$ macctl power hogs
PID    COMMAND       CPU%
382    WindowServer  34.7
12422  claude        30.3
5647   claude        22.5
71898  Magnet        12.0
612    iTerm2        9.4
```

### Display

```
$ macctl display list
NAME       RESOLUTION             REFRESH  VENDOR  MAIN
Color LCD  1512 x 982 @ 120.00Hz                   yes

$ macctl display brightness 80
Brightness: set to 80%
```

### Audio

```
$ macctl audio list
NAME                    TYPE    ACTIVE
MacBook Pro Microphone  input   yes
MacBook Pro Speakers    output  yes

$ macctl audio volume
Output Volume: 63%
Input Volume:  50%
```

### Focus

```
$ macctl focus status
Focus:  off

$ macctl focus on
Focus:  enabled
```

### Presets

```
$ macctl preset
NAME           DESCRIPTION
deep-work      Focus on, display brightness 50%, audio mute
meeting        Focus on (allow calls), audio unmute
present        Focus on, display brightness 100%
chill          Focus off, Night Shift on, display brightness 40%, audio volume 30%
battery-saver  Display brightness 30%, show power hogs

$ macctl preset deep-work --dry-run
Would apply preset: deep-work
  Focus:       on
  Brightness:  50%
  Audio:       mute
```

## Commands

| Command | Description |
|---------|-------------|
| `macctl power status` | Battery status, state, temperature |
| `macctl power health` | Battery health and cycle count |
| `macctl power thermal` | Thermal pressure state |
| `macctl power hogs` | Top energy-consuming processes |
| `macctl power assertions` | Active power assertions |
| `macctl display list` | Connected displays |
| `macctl display brightness [n]` | Get or set brightness (0-100) |
| `macctl display nightshift [on\|off]` | Get or toggle Night Shift |
| `macctl audio list` | Audio input/output devices |
| `macctl audio volume [n]` | Get or set volume (0-100) |
| `macctl audio mute [on\|off]` | Control mute state |
| `macctl audio input [name]` | Get or switch input device |
| `macctl audio output [name]` | Get or switch output device |
| `macctl focus status` | Current focus/DnD state |
| `macctl focus on` | Enable Focus/DnD |
| `macctl focus off` | Disable Focus/DnD |
| `macctl focus list` | Configured focus modes |
| `macctl preset [name]` | List or apply presets |

All commands support `--json` for machine-readable output.

## TUI

Launch `macctl` without arguments for an interactive dashboard.

## Claude Code

macctl is part of the [macos-toolkit](https://github.com/lu-zhengda/macos-toolkit) Claude Code plugin. Install the plugin to let Claude manage your Mac environment using natural language.

## License

[MIT](LICENSE)
