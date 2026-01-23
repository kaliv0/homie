package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
			// read flags
			if err := config.ReadConfig(); err != nil {
				log.Logger().Println(err)
			}

			// limit is read in order:
			//'--limit <n>' cli flag -> .homierc  -> Flags().IntP() default val
			limit := viper.GetInt("limit")
			if limit <= 0 {
				limit = storage.DefaultLimit
			}

			shouldPaste, err := cmd.Flags().GetBool("paste")
			if err != nil {
				log.Logger().Fatalf("failed to get 'paste' flag: %v", err)
			}

			// fetch history-to-be-displayed
			dbPath, err := config.DBPath()
			if err != nil {
				log.Logger().Fatal(err)
			}
			output, err := finder.ListHistory(dbPath, limit)
			if err != nil {
				log.Logger().Fatal(err)
			}
			if len(output) == 0 {
				return
			}

			writeToClipboard(output)
			if !shouldPaste {
				return
			}

			targetPane := os.Getenv("HOMIE_TARGET_PANE")
			if targetPane != "" {
				if err := pasteToTmuxPane(output, targetPane); err != nil {
					log.Logger().Fatal(err)
				}
				return
			}
			fmt.Print(output)
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

func init() {
	listHistoryCmd.Flags().IntP(
		"limit",
		"l",
		storage.DefaultLimit,
		"Limit the number of clipboard history items displayed",
	)
	listHistoryCmd.Flags().BoolP(
		"paste",
		"p",
		false,
		"Paste selected history item",
	)

	if err := viper.BindPFlag("limit", listHistoryCmd.Flags().Lookup("limit")); err != nil {
		log.Logger().Fatalf("failed to bind 'limit' flag to viper: %v", err)
	}
	viper.SetDefault("use_xclip", true)

	rootCmd.AddCommand(listHistoryCmd)
	rootCmd.AddCommand(clearHistoryCmd)
}

func writeToClipboard(text string) {
	useXclip := viper.GetBool("use_xclip")
	if _, err := exec.LookPath("xclip"); err != nil {
		log.Logger().Println("xclip not found, falling back to \"golang.design/x/clipboard\"")
		useXclip = false
	}

	if runtime.GOOS == "linux" && useXclip {
		if err := clipboard.Write(text); err != nil {
			log.Logger().Fatal(err)
		}
		return
	}

	if err := gclip.Init(); err != nil {
		log.Logger().Fatalf("failed to initialize clipboard: %v", err)
	}
	gclip.Write(gclip.FmtText, []byte(text))
}

func pasteToTmuxPane(text, paneID string) error {
	loadBuf := exec.Command("tmux", "load-buffer", "-")
	loadBuf.Stdin = strings.NewReader(text)
	if err := loadBuf.Run(); err != nil {
		return fmt.Errorf("failed to load tmux buffer: %w", err)
	}

	pasteBuf := exec.Command("tmux", "paste-buffer", "-t", paneID, "-dp")
	if err := pasteBuf.Run(); err != nil {
		_ = exec.Command("tmux", "delete-buffer").Run()
		return fmt.Errorf("failed to paste to tmux pane: %w", err)
	}
	return nil
}
