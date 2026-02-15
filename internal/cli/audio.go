package cli

import (
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/lu-zhengda/macctl/internal/audio"
)

var audioCmd = &cobra.Command{
	Use:   "audio",
	Short: "Audio devices, volume, and mute control",
	Long:  `Manage audio devices, control volume, and toggle mute.`,
}

var audioListCmd = &cobra.Command{
	Use:   "list",
	Short: "List audio devices",
	RunE: func(cmd *cobra.Command, args []string) error {
		devices, err := audio.ListDevices()
		if err != nil {
			return fmt.Errorf("failed to list audio devices: %w", err)
		}

		if jsonFlag {
			return printJSON(devices)
		}

		if len(devices) == 0 {
			fmt.Println("No audio devices found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tTYPE\tACTIVE")
		for _, d := range devices {
			active := ""
			if d.Active {
				active = "yes"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", d.Name, d.Type, active)
		}
		w.Flush()
		return nil
	},
}

var audioOutputCmd = &cobra.Command{
	Use:   "output [device]",
	Short: "Get or switch audio output device",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			name, err := audio.GetCurrentOutput()
			if err != nil {
				return fmt.Errorf("failed to get current output: %w", err)
			}

			if jsonFlag {
				return printJSON(map[string]string{"output_device": name})
			}

			fmt.Printf("Output: %s\n", name)
			return nil
		}

		if err := audio.SetOutput(args[0]); err != nil {
			return fmt.Errorf("failed to set output device: %w", err)
		}
		fmt.Printf("Output device set to: %s\n", args[0])
		return nil
	},
}

var audioInputCmd = &cobra.Command{
	Use:   "input [device]",
	Short: "Get or switch audio input device",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			name, err := audio.GetCurrentInput()
			if err != nil {
				return fmt.Errorf("failed to get current input: %w", err)
			}

			if jsonFlag {
				return printJSON(map[string]string{"input_device": name})
			}

			fmt.Printf("Input: %s\n", name)
			return nil
		}

		if err := audio.SetInput(args[0]); err != nil {
			return fmt.Errorf("failed to set input device: %w", err)
		}
		fmt.Printf("Input device set to: %s\n", args[0])
		return nil
	},
}

var audioVolumeCmd = &cobra.Command{
	Use:   "volume [level]",
	Short: "Get or set volume (0-100)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			vol, err := audio.GetVolume()
			if err != nil {
				return fmt.Errorf("failed to get volume: %w", err)
			}

			if jsonFlag {
				return printJSON(vol)
			}

			muteStr := ""
			if vol.Muted {
				muteStr = " (muted)"
			}
			fmt.Printf("Output Volume: %d%%%s\n", vol.OutputVolume, muteStr)
			fmt.Printf("Input Volume:  %d%%\n", vol.InputVolume)
			return nil
		}

		level, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid volume level: %w", err)
		}
		if err := audio.SetVolume(level); err != nil {
			return fmt.Errorf("failed to set volume: %w", err)
		}
		fmt.Printf("Volume set to %d%%\n", level)
		return nil
	},
}

var audioMuteCmd = &cobra.Command{
	Use:   "mute [on|off|toggle]",
	Short: "Control mute state",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 || args[0] == "toggle" {
			if err := audio.ToggleMute(); err != nil {
				return fmt.Errorf("failed to toggle mute: %w", err)
			}
			fmt.Println("Mute toggled")
			return nil
		}

		switch args[0] {
		case "on":
			if err := audio.SetMute(true); err != nil {
				return fmt.Errorf("failed to mute: %w", err)
			}
			fmt.Println("Muted")
		case "off":
			if err := audio.SetMute(false); err != nil {
				return fmt.Errorf("failed to unmute: %w", err)
			}
			fmt.Println("Unmuted")
		default:
			return fmt.Errorf("invalid argument: %s (use on, off, or toggle)", args[0])
		}
		return nil
	},
}

func init() {
	audioCmd.AddCommand(audioListCmd)
	audioCmd.AddCommand(audioOutputCmd)
	audioCmd.AddCommand(audioInputCmd)
	audioCmd.AddCommand(audioVolumeCmd)
	audioCmd.AddCommand(audioMuteCmd)
	rootCmd.AddCommand(audioCmd)
}
