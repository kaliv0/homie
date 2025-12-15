package cmd

import (
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/kaliv0/homie/internal"
)

var (
	startDaemonCmd = &cobra.Command{
		Use:                   "start",
		Short:                 "Start clipboard manager",
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, _ []string) {
			internal.StopAllInstances()
			if err := exec.Command(cmd.Root().Name(), "run").Start(); err != nil {
				internal.Logger.Fatal(err)
			}
		},
	}

	runCmd = &cobra.Command{
		Use:    "run",
		Hidden: true,
		Run: func(cmd *cobra.Command, _ []string) {
			dbPath := internal.DBPath()
			db := internal.NewRepository(dbPath, true)
			internal.CleanOldHistory(db)
			internal.TrackClipboard(db)
		},
	}

	stopCmd = &cobra.Command{
		Use:                   "stop",
		Short:                 "Stop clipboard manager",
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, _ []string) {
			internal.StopAllInstances()
		},
	}
)

func init() {
	rootCmd.AddCommand(startDaemonCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(stopCmd)
}
