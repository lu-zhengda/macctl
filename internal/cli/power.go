package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/lu-zhengda/macctl/internal/power"
)

var powerCmd = &cobra.Command{
	Use:   "power",
	Short: "Battery, thermal, and power management",
	Long:  `Inspect battery status, health, thermal state, power assertions, and energy-hungry processes.`,
}

var powerStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show battery status",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := power.GetStatus()
		if err != nil {
			return fmt.Errorf("failed to get power status: %w", err)
		}

		if jsonFlag {
			return printJSON(s)
		}

		chargingState := "discharging"
		if s.IsCharging {
			chargingState = "charging"
		} else if s.ExternalConnected {
			chargingState = "on AC power"
		}

		fmt.Printf("Battery:       %d%%\n", s.Percent)
		fmt.Printf("State:         %s\n", chargingState)
		fmt.Printf("Time:          %s\n", s.TimeRemaining)
		fmt.Printf("Cycles:        %d\n", s.CycleCount)
		fmt.Printf("Temperature:   %.1f C\n", s.Temperature)
		fmt.Printf("Capacity:      %d / %d mAh\n", s.CurrentCapacity, s.MaxCapacity)
		return nil
	},
}

var powerHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Show battery health",
	RunE: func(cmd *cobra.Command, args []string) error {
		h, err := power.GetHealth()
		if err != nil {
			return fmt.Errorf("failed to get battery health: %w", err)
		}

		if jsonFlag {
			return printJSON(h)
		}

		fmt.Printf("Health:          %.1f%%\n", h.HealthPercent)
		fmt.Printf("Condition:       %s\n", h.Condition)
		fmt.Printf("Design Capacity: %d mAh\n", h.DesignCapacity)
		fmt.Printf("Max Capacity:    %d mAh\n", h.MaxCapacity)
		fmt.Printf("Cycle Count:     %d\n", h.CycleCount)
		return nil
	},
}

var powerThermalCmd = &cobra.Command{
	Use:   "thermal",
	Short: "Show thermal status",
	RunE: func(cmd *cobra.Command, args []string) error {
		t, err := power.GetThermal()
		if err != nil {
			return fmt.Errorf("failed to get thermal info: %w", err)
		}

		if jsonFlag {
			return printJSON(t)
		}

		fmt.Printf("Pressure Level: %s\n", t.PressureLevel)
		fmt.Printf("Temperature:    %s\n", t.CPUTemp)
		return nil
	},
}

var powerAssertionsCmd = &cobra.Command{
	Use:   "assertions",
	Short: "List active power assertions",
	RunE: func(cmd *cobra.Command, args []string) error {
		assertions, err := power.GetAssertions()
		if err != nil {
			return fmt.Errorf("failed to get power assertions: %w", err)
		}

		if jsonFlag {
			return printJSON(assertions)
		}

		if len(assertions) == 0 {
			fmt.Println("No active power assertions.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "PID\tPROCESS\tTYPE\tREASON")
		for _, a := range assertions {
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", a.PID, a.Name, a.Type, a.Reason)
		}
		w.Flush()
		return nil
	},
}

var (
	powerHogsN       int
	powerHistoryLast string
)

var powerHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Show battery/thermal history",
	Long:  `Display historical battery and thermal snapshots recorded over time.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		snapshots, err := power.LoadHistory()
		if err != nil {
			return fmt.Errorf("failed to load power history: %w", err)
		}

		if snapshots == nil {
			fmt.Println("No power history recorded yet. Use 'macctl power record' to capture snapshots.")
			return nil
		}

		if powerHistoryLast != "" {
			dur, err := power.ParseDuration(powerHistoryLast)
			if err != nil {
				return fmt.Errorf("invalid duration: %w", err)
			}
			snapshots = power.FilterHistory(snapshots, dur)
		} else if len(snapshots) > power.DefaultHistoryCount {
			snapshots = snapshots[len(snapshots)-power.DefaultHistoryCount:]
		}

		if jsonFlag {
			return printJSON(snapshots)
		}

		if len(snapshots) == 0 {
			fmt.Println("No entries match the given filter.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "TIMESTAMP\tBATTERY\tCHARGING\tCYCLES\tMAX_CAP\tTEMP\tTHERMAL")
		for _, s := range snapshots {
			charging := "no"
			if s.IsCharging {
				charging = "yes"
			}
			fmt.Fprintf(w, "%s\t%d%%\t%s\t%d\t%d mAh\t%.1f C\t%s\n",
				s.Timestamp.Local().Format("2006-01-02 15:04"),
				s.BatteryPct, charging, s.CycleCount, s.MaxCapacity,
				s.Temperature, s.ThermalLevel)
		}
		w.Flush()
		return nil
	},
}

var powerRecordCmd = &cobra.Command{
	Use:   "record",
	Short: "Record a power snapshot to history",
	Long:  `Capture a snapshot of current battery and thermal state and append it to the history file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		snap, err := power.RecordSnapshot()
		if err != nil {
			return fmt.Errorf("failed to record power snapshot: %w", err)
		}

		if jsonFlag {
			return printJSON(snap)
		}

		fmt.Printf("Recorded snapshot at %s: %d%% battery, %s thermal\n",
			snap.Timestamp.Local().Format("2006-01-02 15:04:05"),
			snap.BatteryPct, snap.ThermalLevel)
		return nil
	},
}

var powerHogsCmd = &cobra.Command{
	Use:   "hogs",
	Short: "Show top energy consumers",
	RunE: func(cmd *cobra.Command, args []string) error {
		hogs, err := power.GetEnergyHogs(powerHogsN)
		if err != nil {
			return fmt.Errorf("failed to get energy hogs: %w", err)
		}

		if jsonFlag {
			return printJSON(hogs)
		}

		if len(hogs) == 0 {
			fmt.Println("No energy-consuming processes found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "PID\tCOMMAND\tCPU%")
		for _, h := range hogs {
			fmt.Fprintf(w, "%d\t%s\t%.1f\n", h.PID, h.Command, h.CPU)
		}
		w.Flush()
		return nil
	},
}

func init() {
	powerHogsCmd.Flags().IntVarP(&powerHogsN, "n", "n", 5, "Number of processes to show")
	powerHistoryCmd.Flags().StringVar(&powerHistoryLast, "last", "", "Show entries from last duration (e.g., 24h, 7d)")

	powerCmd.AddCommand(powerStatusCmd)
	powerCmd.AddCommand(powerHealthCmd)
	powerCmd.AddCommand(powerThermalCmd)
	powerCmd.AddCommand(powerAssertionsCmd)
	powerCmd.AddCommand(powerHogsCmd)
	powerCmd.AddCommand(powerHistoryCmd)
	powerCmd.AddCommand(powerRecordCmd)
	rootCmd.AddCommand(powerCmd)
}
