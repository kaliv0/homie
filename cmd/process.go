package cmd

import (
	"context"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/kaliv0/homie/internal/clipboard"
	"github.com/kaliv0/homie/internal/config"
	"github.com/kaliv0/homie/internal/daemon"
	"github.com/kaliv0/homie/internal/log"
	"github.com/kaliv0/homie/internal/storage"
)

var (
	startDaemonCmd = &cobra.Command{
		Use:                   "start",
		Short:                 "Start clipboard manager",
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, _ []string) {
			if err := daemon.StopAllInstances(); err != nil {
				log.Logger().Println(err)
			}

			cmdName := cmd.Root().Name()
			daemonCmd := exec.Command(cmdName, "run")
			if err := daemonCmd.Start(); err != nil {
				log.Logger().Fatalf("failed to start daemon process (command=%q run): %v", cmdName, err)
			}
			if err := daemonCmd.Process.Release(); err != nil {
				log.Logger().Printf("failed to release daemon process: %v\n", err)
			}
		},
	}

	runCmd = &cobra.Command{
		Use:    "run",
		Hidden: true,
		Run: func(cmd *cobra.Command, _ []string) {
			dbPath, err := config.DBPath()
			if err != nil {
				log.Logger().Fatal(err)
			}
			db, err := storage.NewRepository(dbPath)
			if err != nil {
				log.Logger().Fatal(err)
			}

			defer func() {
				if closeErr := db.Close(); closeErr != nil {
					log.Logger().Println(closeErr)
				}
			}()

			if err := db.AutoMigrate(); err != nil {
				_ = db.Close()
				log.Logger().Fatal(err)
			}

			if err := storage.CleanOldHistory(db); err != nil {
				log.Logger().Println(err)
			}

			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			if err := clipboard.TrackClipboard(ctx, db); err != nil {
				_ = db.Close()
				log.Logger().Fatal(err)
			}
		},
	}

	stopCmd = &cobra.Command{
		Use:                   "stop",
		Short:                 "Stop clipboard manager",
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, _ []string) {
			if err := daemon.StopAllInstances(); err != nil {
				log.Logger().Fatal(err)
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(startDaemonCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(stopCmd)
}
