package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	gclip "golang.design/x/clipboard"

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
			output, err := fetchDisplayHistory(cfg)
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

func fetchDisplayHistory(cfg *config.Config) (string, error) {
	dbPath, err := config.DBPath()
	if err != nil {
		return "", err
	}
	return finder.ListHistory(dbPath, cfg.Limit)
}

func writeToClipboard(cfg *config.Config, text string) error {
	tool, err := clipboardTool(cfg)
	if err != nil {
		return err
	}
	if tool != "" {
		return clipboard.Write(text, tool)
	}

	if err = gclip.Init(); err != nil {
		return fmt.Errorf("failed to initialize clipboard: %w", err)
	}
	gclip.Write(gclip.FmtText, []byte(text))
	return nil
}

func clipboardTool(cfg *config.Config) (string, error) {
	if runtime.GOOS != "linux" {
		return "", nil
	}

	tool := cfg.Tool
	if tool == "" {
		return "", fmt.Errorf(
			"clipboard tool not set: add %s, %s, or %s to ~/.homierc",
			config.ClipboardXclip, config.ClipboardXsel, config.ClipboardWLClipboard,
		)
	}

	bin, err := clipboard.Binary(tool)
	if err != nil {
		return "", err
	}
	if _, err := exec.LookPath(bin); err != nil {
		return "", fmt.Errorf("%s not found: %w", tool, err)
	}
	return tool, nil
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
		config.KeyLimit,
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

	config.BindFlag(config.KeyLimit, listHistoryCmd.Flags().Lookup(config.KeyLimit))

	rootCmd.AddCommand(listHistoryCmd)
	rootCmd.AddCommand(clearHistoryCmd)
}
