package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
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
		Run: func(cmd *cobra.Command, _ []string) {
			running, _, err := daemon.Status(cfg)
			if err != nil {
				log.Logger().Fatal(err)
			}
			if running {
				if log.Verbose() {
					log.Logger().Println("homie daemon is already running")
				}
				return
			}
			spawnDaemon(cmd)
			if log.Verbose() {
				log.Logger().Println("homie daemon started")
			}
		},
	}

	restartDaemonCmd = &cobra.Command{
		Use:                   "restart",
		Short:                 "Restart clipboard manager",
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, _ []string) {
			if err := daemon.Stop(cfg); err != nil {
				log.Logger().Fatal(err)
			}
			spawnDaemon(cmd)
			if log.Verbose() {
				log.Logger().Println("homie daemon restarted")
			}
		},
	}

	runCmd = &cobra.Command{
		Use:    "run",
		Hidden: true,
		Run: func(_ *cobra.Command, _ []string) {
			if err := runProcess(cfg); err != nil {
				if errors.Is(err, daemon.ErrAlreadyRunning) {
					if log.Verbose() {
						log.Logger().Println("homie daemon is already running")
					}
					os.Exit(1)
				}
				log.Logger().Fatal(err)
			}
		},
	}

	stopCmd = &cobra.Command{
		Use:                   "stop",
		Short:                 "Stop clipboard manager",
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, _ []string) {
			if err := daemon.Stop(cfg); err != nil {
				log.Logger().Fatal(err)
			}
			if log.Verbose() {
				log.Logger().Println("homie daemon stopped")
			}
		},
	}

	statusCmd = &cobra.Command{
		Use:                   "status",
		Short:                 "Show clipboard manager daemon status",
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, _ []string) {
			running, pid, err := daemon.Status(cfg)
			if err != nil {
				log.Logger().Fatal(err)
			}
			if running {
				fmt.Printf("running (pid %d)\n", pid)
				return
			}
			fmt.Println("not running")
		},
	}
)

func spawnDaemon(cmd *cobra.Command) {
	if err := daemon.Start(cfg, cmd.Root().Name(), "run"); err != nil {
		log.Logger().Fatal(err)
	}
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

	dbPath, err := config.DBPath()
	if err != nil {
		return err
	}
	db, err := storage.NewRepository(dbPath)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			log.Logger().Println(closeErr)
		}
	}()

	if err := db.AutoMigrate(); err != nil {
		return err
	}
	if err := db.SetDBFilesPermissions(); err != nil {
		return err
	}

	cleanup := storage.CleanupConfig{
		TTL:     cfg.TTL,
		MaxSize: cfg.MaxSize,
		MinSize: cfg.MinSize,
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
