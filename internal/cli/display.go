package cli

import (
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/lu-zhengda/macctl/internal/display"
)

var displayCmd = &cobra.Command{
	Use:   "display",
	Short: "Display info, brightness, and Night Shift",
	Long:  `View connected displays, control brightness, and toggle Night Shift.`,
}

var displayListCmd = &cobra.Command{
	Use:   "list",
	Short: "List connected displays",
	RunE: func(cmd *cobra.Command, args []string) error {
		displays, err := display.List()
		if err != nil {
			return fmt.Errorf("failed to list displays: %w", err)
		}

		if jsonFlag {
			return printJSON(displays)
		}

		if len(displays) == 0 {
			fmt.Println("No displays found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tRESOLUTION\tREFRESH\tVENDOR\tMAIN")
		for _, d := range displays {
			main := ""
			if d.Main {
				main = "yes"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				d.Name, d.Resolution, d.RefreshRate, d.Vendor, main)
		}
		w.Flush()
		return nil
	},
}

var displayBrightnessCmd = &cobra.Command{
	Use:   "brightness [level]",
	Short: "Get or set display brightness (0-100)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			// Get brightness.
			b, err := display.GetBrightness()
			if err != nil {
				return fmt.Errorf("failed to get brightness: %w", err)
			}

			if jsonFlag {
				return printJSON(b)
			}

			if b.Level < 0 {
				fmt.Println("Brightness: unavailable (cannot read display brightness)")
			} else {
				fmt.Printf("Brightness: %.0f%%\n", b.Level)
			}
			return nil
		}

		// Set brightness.
		level, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid brightness level: %w", err)
		}
		if err := display.SetBrightness(level); err != nil {
			return fmt.Errorf("failed to set brightness: %w", err)
		}
		fmt.Printf("Brightness set to %d%%\n", level)
		return nil
	},
}

var displayNightShiftCmd = &cobra.Command{
	Use:   "nightshift [on|off|status]",
	Short: "Get or set Night Shift",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 || args[0] == "status" {
			ns, err := display.GetNightShift()
			if err != nil {
				return fmt.Errorf("failed to get night shift status: %w", err)
			}

			if jsonFlag {
				return printJSON(ns)
			}

			fmt.Printf("Night Shift: %s\n", ns.Status)
			return nil
		}

		switch args[0] {
		case "on":
			if err := display.SetNightShift(true); err != nil {
				return fmt.Errorf("failed to enable night shift: %w", err)
			}
			fmt.Println("Night Shift enabled")
		case "off":
			if err := display.SetNightShift(false); err != nil {
				return fmt.Errorf("failed to disable night shift: %w", err)
			}
			fmt.Println("Night Shift disabled")
		default:
			return fmt.Errorf("invalid argument: %s (use on, off, or status)", args[0])
		}
		return nil
	},
}

func init() {
	displayCmd.AddCommand(displayListCmd)
	displayCmd.AddCommand(displayBrightnessCmd)
	displayCmd.AddCommand(displayNightShiftCmd)
	rootCmd.AddCommand(displayCmd)
}
