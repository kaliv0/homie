package cmd

import (
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/kaliv0/homie/internal/clipboard"
	"github.com/kaliv0/homie/internal/config"
	"github.com/kaliv0/homie/internal/runtime"
	"github.com/kaliv0/homie/internal/storage"
)

var (
	startDaemonCmd = &cobra.Command{
		Use:                   "start",
		Short:                 "Start clipboard manager",
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, _ []string) {
			if err := runtime.StopAllInstances(); err != nil {
				runtime.Logger().Println(err)
			}
			if err := exec.Command(cmd.Root().Name(), "run").Start(); err != nil {
				runtime.Logger().Fatal(err)
			}
		},
	}

	runCmd = &cobra.Command{
		Use:    "run",
		Hidden: true,
		Run: func(cmd *cobra.Command, _ []string) {
			dbPath, err := config.DBPath()
			if err != nil {
				runtime.Logger().Fatal(err)
			}
			db, err := storage.NewRepository(dbPath, true)
			if err != nil {
				runtime.Logger().Fatal(err)
			}

			defer func() {
				if closeErr := db.Close(); closeErr != nil {
					runtime.Logger().Println(closeErr)
				}
			}()

			if err := storage.CleanOldHistory(db); err != nil {
				runtime.Logger().Println(err)
			}
			if err := clipboard.TrackClipboard(db); err != nil {
				runtime.Logger().Fatal(err)
			}
		},
	}

	stopCmd = &cobra.Command{
		Use:                   "stop",
		Short:                 "Stop clipboard manager",
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, _ []string) {
			if err := runtime.StopAllInstances(); err != nil {
				runtime.Logger().Fatal(err)
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(startDaemonCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(stopCmd)
}
