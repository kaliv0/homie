package cmd

import (
	"os/exec"

	"github.com/kaliv0/homie/cmd/utils"
	"github.com/spf13/cobra"
)

var (
	startDaemonCmd = &cobra.Command{
		Use:                   "start",
		Short:                 "Start clipboard manager",
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, _ []string) {
			utils.StopAllInstances()
			if err := exec.Command(cmd.Root().Name(), "run").Start(); err != nil {
				utils.Logger.Fatal(err)
			}
		},
	}

	runCmd = &cobra.Command{
		Use:    "run",
		Hidden: true,
		Run: func(cmd *cobra.Command, _ []string) {
			dbPath := utils.GetDbPath()
			db := utils.NewRepository(dbPath, true)
			utils.CleanOldHistory(db)
			utils.TrackClipboard(db)
		},
	}

	stopCmd = &cobra.Command{
		Use:                   "stop",
		Short:                 "Stop clipboard manager",
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, _ []string) {
			utils.StopAllInstances()
		},
	}
)

func init() {
	rootCmd.AddCommand(startDaemonCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(stopCmd)
}
