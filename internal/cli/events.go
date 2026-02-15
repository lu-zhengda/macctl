package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/lu-zhengda/macctl/internal/events"
)

var eventsLast string

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Show power events from system log",
	Long: `Query the macOS system log for power-related events such as
wake/sleep, lid open/close, thermal throttling, and power source changes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		duration := eventsLast
		if duration == "" {
			duration = "24h"
		}

		powerEvents, err := events.GetEvents(duration)
		if err != nil {
			return fmt.Errorf("failed to get power events: %w", err)
		}

		if jsonFlag {
			return printJSON(powerEvents)
		}

		if len(powerEvents) == 0 {
			fmt.Printf("No power events found in the last %s.\n", duration)
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "TIMESTAMP\tTYPE\tDETAIL")
		for _, e := range powerEvents {
			detail := e.Detail
			if len(detail) > 80 {
				detail = detail[:80] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n",
				e.Timestamp.Local().Format("2006-01-02 15:04:05"),
				e.Type, detail)
		}
		w.Flush()
		return nil
	},
}

func init() {
	eventsCmd.Flags().StringVar(&eventsLast, "last", "", "Duration to look back (e.g., 24h, 7d; default: 24h)")
	rootCmd.AddCommand(eventsCmd)
}
