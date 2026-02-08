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
			if err := config.ReadConfig(); err != nil {
				log.Logger().Println(err)
			}

			output, err := fetchDisplayHistory()
			if err != nil {
				log.Logger().Fatal(err)
			}
			if len(output) == 0 {
				return
			}

			if err = writeToClipboard(output); err != nil {
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

func fetchDisplayHistory() (string, error) {
	// limit is read in order:
	//'--limit <n>' cli flag -> .homierc  -> Flags().IntP() default val
	limit := viper.GetInt("limit")
	if limit <= 0 {
		limit = storage.DefaultLimit
	}

	dbPath, err := config.DBPath()
	if err != nil {
		return "", err
	}
	return finder.ListHistory(dbPath, limit)
}

func writeToClipboard(text string) error {
	if tool := clipboardTool(); tool != "" {
		return clipboard.Write(text, tool)
	}

	if err := gclip.Init(); err != nil {
		return fmt.Errorf("failed to initialize clipboard: %w", err)
	}
	gclip.Write(gclip.FmtText, []byte(text))
	return nil
}

func clipboardTool() string {
	if runtime.GOOS != "linux" {
		return ""
	}
	for _, tool := range []string{"xclip", "xsel"} {
		if viper.GetBool("use_" + tool) {
			if _, err := exec.LookPath(tool); err == nil {
				log.Logger().Printf("using %s as clipboardTool", tool)
				return tool
			}
			log.Logger().Printf("%s not found", tool)
		}
	}
	log.Logger().Println(`falling back to "golang.design/x/clipboard"`)
	return ""
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
