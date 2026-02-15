package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/lu-zhengda/macctl/internal/preset"
)

var presetDryRun bool

var presetCmd = &cobra.Command{
	Use:   "preset [name]",
	Short: "Apply or list presets",
	Long: `Apply a compound preset or list available presets.
Without arguments, lists all available presets.
With a preset name, applies that preset.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			// List presets.
			presets := preset.BuiltinPresets()

			if jsonFlag {
				return printJSON(presets)
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tDESCRIPTION")
			for _, p := range presets {
				fmt.Fprintf(w, "%s\t%s\n", p.Name, p.Description)
			}
			w.Flush()
			return nil
		}

		// Apply preset.
		p := preset.Get(args[0])
		if p == nil {
			return fmt.Errorf("unknown preset: %s", args[0])
		}

		if presetDryRun {
			results := preset.DryRun(p)

			if jsonFlag {
				return printJSON(results)
			}

			fmt.Printf("Preset: %s\n", p.Name)
			fmt.Printf("Description: %s\n\n", p.Description)
			fmt.Println("Would execute:")
			for _, r := range results {
				fmt.Printf("  %s\n", r.Message)
			}
			return nil
		}

		results := preset.Apply(p)

		if jsonFlag {
			return printJSON(results)
		}

		fmt.Printf("Applying preset: %s\n\n", p.Name)
		for _, r := range results {
			status := "OK"
			if !r.Success {
				status = "FAIL"
			}
			fmt.Printf("  [%s] %s\n", status, r.Message)
		}
		return nil
	},
}

func init() {
	presetCmd.Flags().BoolVar(&presetDryRun, "dry-run", false, "Show what would be applied without executing")
	rootCmd.AddCommand(presetCmd)
}
