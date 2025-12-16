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
			if err := internal.StopAllInstances(); err != nil {
				internal.Logger.Print(err)
			}
			if err := exec.Command(cmd.Root().Name(), "run").Start(); err != nil {
				internal.Logger.Fatal(err)
			}
		},
	}

	runCmd = &cobra.Command{
		Use:    "run",
		Hidden: true,
		Run: func(cmd *cobra.Command, _ []string) {
			dbPath, err := internal.DBPath()
			if err != nil {
				internal.Logger.Fatal(err)
			}
			db, err := internal.NewRepository(dbPath, true)
			if err != nil {
				internal.Logger.Fatal(err)
			}

			defer func() {
				if closeErr := db.Close(); closeErr != nil {
					internal.Logger.Print(closeErr)
				}
			}()

			if err := internal.CleanOldHistory(db); err != nil {
				internal.Logger.Print(err)
			}
			if err := internal.TrackClipboard(db); err != nil {
				internal.Logger.Fatal(err)
			}
		},
	}

	stopCmd = &cobra.Command{
		Use:                   "stop",
		Short:                 "Stop clipboard manager",
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, _ []string) {
			if err := internal.StopAllInstances(); err != nil {
				internal.Logger.Fatal(err)
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(startDaemonCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(stopCmd)
}
