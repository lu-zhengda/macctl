package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/lu-zhengda/macctl/internal/disk"
)

var diskCmd = &cobra.Command{
	Use:   "disk",
	Short: "SSD health, I/O stats, and wear trends",
	Long:  `Inspect SSD health, view current I/O rates, and track wear over time.`,
}

var diskStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show SSD health status",
	RunE: func(cmd *cobra.Command, args []string) error {
		h, err := disk.GetHealth()
		if err != nil {
			return fmt.Errorf("failed to get disk health: %w", err)
		}

		if jsonFlag {
			return printJSON(h)
		}

		fmt.Printf("Device:       %s\n", h.Device)
		fmt.Printf("Model:        %s\n", h.Model)
		fmt.Printf("Protocol:     %s\n", h.Protocol)
		fmt.Printf("Size:         %s\n", h.SizeHuman)
		fmt.Printf("SMART Status: %s\n", h.SmartStatus)
		fmt.Printf("Wear Level:   %s\n", h.WearLevel)
		fmt.Printf("Data Written: %s\n", h.DataWritten)
		return nil
	},
}

var diskIOCmd = &cobra.Command{
	Use:   "io",
	Short: "Show current I/O rates",
	Long:  `Display current disk read/write throughput and IOPS.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		stats, err := disk.GetIOStats()
		if err != nil {
			return fmt.Errorf("failed to get I/O stats: %w", err)
		}

		if jsonFlag {
			return printJSON(stats)
		}

		fmt.Printf("Read:   %.2f MB/s  (%.0f IOPS)\n", stats.ReadMBs, stats.ReadIOPS)
		fmt.Printf("Write:  %.2f MB/s  (%.0f IOPS)\n", stats.WriteMBs, stats.WriteIOPS)
		return nil
	},
}

var diskHistoryLast string

var diskHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Show SSD wear trends over time",
	Long:  `Display historical disk health snapshots recorded over time.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		snapshots, err := disk.LoadHistory()
		if err != nil {
			return fmt.Errorf("failed to load disk history: %w", err)
		}

		if snapshots == nil {
			fmt.Println("No disk history recorded yet. Use 'macctl disk record' to capture snapshots.")
			return nil
		}

		if diskHistoryLast != "" {
			dur, err := disk.ParseDuration(diskHistoryLast)
			if err != nil {
				return fmt.Errorf("invalid duration: %w", err)
			}
			snapshots = disk.FilterHistory(snapshots, dur)
		} else if len(snapshots) > disk.DefaultHistoryCount {
			snapshots = snapshots[len(snapshots)-disk.DefaultHistoryCount:]
		}

		if jsonFlag {
			return printJSON(snapshots)
		}

		if len(snapshots) == 0 {
			fmt.Println("No entries match the given filter.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "TIMESTAMP\tMODEL\tSMART\tWEAR\tDATA_WRITTEN")
		for _, s := range snapshots {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				s.Timestamp.Local().Format("2006-01-02 15:04"),
				s.Model, s.SmartStatus, s.WearLevel, s.DataWritten)
		}
		w.Flush()
		return nil
	},
}

var diskRecordCmd = &cobra.Command{
	Use:   "record",
	Short: "Record a disk health snapshot to history",
	Long:  `Capture a snapshot of current disk health and append it to the history file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		snap, err := disk.RecordSnapshot()
		if err != nil {
			return fmt.Errorf("failed to record disk snapshot: %w", err)
		}

		if jsonFlag {
			return printJSON(snap)
		}

		fmt.Printf("Recorded disk snapshot at %s: %s, SMART=%s, wear=%s\n",
			snap.Timestamp.Local().Format("2006-01-02 15:04:05"),
			snap.Model, snap.SmartStatus, snap.WearLevel)
		return nil
	},
}

func init() {
	diskHistoryCmd.Flags().StringVar(&diskHistoryLast, "last", "", "Show entries from last duration (e.g., 24h, 7d)")

	diskCmd.AddCommand(diskStatusCmd)
	diskCmd.AddCommand(diskIOCmd)
	diskCmd.AddCommand(diskHistoryCmd)
	diskCmd.AddCommand(diskRecordCmd)
	rootCmd.AddCommand(diskCmd)
}
