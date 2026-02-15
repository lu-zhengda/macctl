package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/lu-zhengda/macctl/internal/focus"
)

var focusCmd = &cobra.Command{
	Use:   "focus",
	Short: "Focus modes and Do Not Disturb",
	Long: `Control Focus modes and Do Not Disturb.
Note: Some Focus APIs are restricted on macOS and may require Accessibility permissions.`,
}

var focusStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current focus mode status",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := focus.GetStatus()
		if err != nil {
			return fmt.Errorf("failed to get focus status: %w", err)
		}

		if jsonFlag {
			return printJSON(s)
		}

		if s.Active {
			fmt.Printf("Focus:  active (%s)\n", s.Mode)
			fmt.Printf("DnD:    %v\n", s.DnDActive)
		} else {
			fmt.Println("Focus:  off")
		}
		return nil
	},
}

var focusOnCmd = &cobra.Command{
	Use:   "on [mode]",
	Short: "Enable Focus/DnD",
	Long: `Enable Do Not Disturb or a specific Focus mode.
Without arguments, enables Do Not Disturb.
With a mode name, attempts to activate that Focus mode via Shortcuts.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mode := ""
		if len(args) > 0 {
			mode = args[0]
		}

		if err := focus.Enable(mode); err != nil {
			return fmt.Errorf("failed to enable focus mode: %w", err)
		}

		if mode == "" || mode == "dnd" {
			fmt.Println("Do Not Disturb enabled")
		} else {
			fmt.Printf("Focus mode '%s' enabled\n", mode)
		}
		return nil
	},
}

var focusOffCmd = &cobra.Command{
	Use:   "off",
	Short: "Disable Focus/DnD",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := focus.Disable(); err != nil {
			return fmt.Errorf("failed to disable focus mode: %w", err)
		}
		fmt.Println("Focus mode disabled")
		return nil
	},
}

var focusListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured focus modes",
	RunE: func(cmd *cobra.Command, args []string) error {
		modes, err := focus.ListModes()
		if err != nil {
			return fmt.Errorf("failed to list focus modes: %w", err)
		}

		if jsonFlag {
			return printJSON(modes)
		}

		if len(modes) == 0 {
			fmt.Println("No focus modes found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tBUILTIN\tACTIVE")
		for _, m := range modes {
			builtin := ""
			if m.Builtin {
				builtin = "yes"
			}
			active := ""
			if m.Active {
				active = "yes"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", m.Name, builtin, active)
		}
		w.Flush()
		return nil
	},
}

func init() {
	focusCmd.AddCommand(focusStatusCmd)
	focusCmd.AddCommand(focusOnCmd)
	focusCmd.AddCommand(focusOffCmd)
	focusCmd.AddCommand(focusListCmd)
	rootCmd.AddCommand(focusCmd)
}
