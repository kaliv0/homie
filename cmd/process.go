package cmd

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	gclip "golang.design/x/clipboard"

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
		RunE: func(cmd *cobra.Command, _ []string) error {
			running, _, err := daemon.Status(cfg)
			if err != nil {
				return err
			}
			if running {
				if log.Verbose() {
					log.Logger().Println("homie daemon is already running")
				}
				return nil
			}
			if err := spawnDaemon(cmd); err != nil {
				return err
			}
			if log.Verbose() {
				log.Logger().Println("homie daemon started")
			}
			return nil
		},
	}

	restartDaemonCmd = &cobra.Command{
		Use:                   "restart",
		Short:                 "Restart clipboard manager",
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := daemon.Stop(cfg); err != nil {
				return err
			}
			if err := spawnDaemon(cmd); err != nil {
				return err
			}
			if log.Verbose() {
				log.Logger().Println("homie daemon restarted")
			}
			return nil
		},
	}

	runCmd = &cobra.Command{
		Use:    "run",
		Hidden: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runProcess(cfg)
		},
	}

	stopCmd = &cobra.Command{
		Use:                   "stop",
		Short:                 "Stop clipboard manager",
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := daemon.Stop(cfg); err != nil {
				return err
			}
			if log.Verbose() {
				log.Logger().Println("homie daemon stopped")
			}
			return nil
		},
	}

	statusCmd = &cobra.Command{
		Use:                   "status",
		Short:                 "Show clipboard manager daemon status",
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			running, pid, err := daemon.Status(cfg)
			if err != nil {
				return err
			}
			if running {
				fmt.Printf("running (pid %d)\n", pid)
				return nil
			}
			fmt.Println("not running")
			return nil
		},
	}
)

func spawnDaemon(cmd *cobra.Command) error {
	return daemon.Start(cfg, cmd.Root().Name(), "run")
}

func runProcess(cfg *config.Config) error {
	lock, err := daemon.Acquire(cfg)
	if err != nil {
		return err
	}
	defer func() {
		if releaseErr := lock.Release(); releaseErr != nil {
			log.Logger().Println(releaseErr)
		}
	}()

	db, err := openDB()
	if err != nil {
		return err
	}
	defer closeDB(db)

	if err := db.AutoMigrate(); err != nil {
		return err
	}
	if err := db.SetDBFilesPermissions(); err != nil {
		return err
	}

	cleanup := storage.CleanupConfig{
		TTL:       cfg.TTL,
		Keep:      cfg.Keep,
		Threshold: cfg.Threshold,
	}
	if err := storage.CleanOldHistory(db, cleanup); err != nil {
		log.Logger().Println(err)
	}

	// Ignore SIGHUP so the daemon survives terminal/session closure (e.g. tmux exit)
	signal.Ignore(syscall.SIGHUP)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := gclip.Init(); err != nil {
		return fmt.Errorf("failed to initialize clipboard: %w", err)
	}
	return clipboard.TrackClipboard(ctx, db, gclip.Watch(ctx, gclip.FmtText))
}

func init() {
	rootCmd.AddCommand(startDaemonCmd)
	rootCmd.AddCommand(restartDaemonCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)
}
