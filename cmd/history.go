package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kaliv0/homie/internal/clipboard"
	"github.com/kaliv0/homie/internal/config"
	"github.com/kaliv0/homie/internal/finder"
	"github.com/kaliv0/homie/internal/log"
	"github.com/kaliv0/homie/internal/storage"
)

var (
	listHistoryCmd = &cobra.Command{
		Use:   "history",
		Short: "List clipboard history",
		Long: `List clipboard history
  Use <tab> to pin and select multiple entries`,
		Run: func(cmd *cobra.Command, _ []string) {
			limit, err := parseLimit(cmd)
			if err != nil {
				log.Logger().Fatal(err)
			}

			output, err := fetchDisplayHistory(limit)
			if err != nil {
				log.Logger().Fatal(err)
			}
			if len(output) == 0 {
				return
			}

			if err = writeToClipboard(cfg, output); err != nil {
				log.Logger().Fatal(err)
			}

			shouldPaste, err := cmd.Flags().GetBool("paste")
			if err != nil {
				log.Logger().Fatalf("failed to get 'paste' flag: %v", err)
			}
			if !shouldPaste {
				return
			}
			if err := pasteText(output); err != nil {
				log.Logger().Fatal(err)
			}
		},
	}

	clearHistoryCmd = &cobra.Command{
		Use:                   "clear",
		Short:                 "Clear clipboard history",
		DisableFlagsInUseLine: true,
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

			if err := db.Reset(); err != nil {
				_ = db.Close()
				log.Logger().Fatal(err)
			}
		},
	}
)

func parseLimit(cmd *cobra.Command) (int, error) {
	if cmd.Flags().Changed("limit") {
		limit, err := cmd.Flags().GetInt("limit")
		if err != nil {
			return 0, fmt.Errorf("failed to get 'limit' flag: %w", err)
		}
		return limit, nil
	}
	return cfg.Limit, nil
}

func fetchDisplayHistory(limit int) (string, error) {
	dbPath, err := config.DBPath()
	if err != nil {
		return "", err
	}
	return finder.ListHistory(dbPath, limit)
}

func writeToClipboard(cfg *config.Config, text string) error {
	return clipboard.WriteSelection(text, cfg.Tool)
}

func pasteText(text string) error {
	targetPane := os.Getenv("HOMIE_TARGET_PANE")
	if targetPane == "" {
		fmt.Print(text)
		return nil
	}

	// paste inside tmux
	loadBuf := exec.Command("tmux", "load-buffer", "-")
	loadBuf.Stdin = strings.NewReader(text)
	if err := loadBuf.Run(); err != nil {
		return fmt.Errorf("failed to load tmux buffer: %w", err)
	}

	pasteBuf := exec.Command("tmux", "paste-buffer", "-t", targetPane, "-dp")
	if err := pasteBuf.Run(); err != nil {
		_ = exec.Command("tmux", "delete-buffer").Run()
		return fmt.Errorf("failed to paste to tmux pane %q: %w", targetPane, err)
	}
	return nil
}

func init() {
	listHistoryCmd.Flags().IntP(
		"limit",
		"l",
		config.DefaultLimit,
		"Limit the number of clipboard history items displayed",
	)
	listHistoryCmd.Flags().BoolP(
		"paste",
		"p",
		false,
		"Paste selected history item",
	)

	rootCmd.AddCommand(listHistoryCmd)
	rootCmd.AddCommand(clearHistoryCmd)
}
