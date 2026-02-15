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

var powerHogsN int

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

	powerCmd.AddCommand(powerStatusCmd)
	powerCmd.AddCommand(powerHealthCmd)
	powerCmd.AddCommand(powerThermalCmd)
	powerCmd.AddCommand(powerAssertionsCmd)
	powerCmd.AddCommand(powerHogsCmd)
	rootCmd.AddCommand(powerCmd)
}
