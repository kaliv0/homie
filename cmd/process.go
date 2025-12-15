package cmd

import (
	"os/exec"

	"github.com/kaliv0/homie/internal"
	"github.com/spf13/cobra"
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
			db, err := internal.NewRepository(dbPath, true)
			if err != nil {
				internal.Logger.Fatal(err)
			}
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
