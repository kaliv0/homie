package cmd

import (
	"fmt"
	"os/exec"
	"runtime"

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
				log.Logger().Fatal(err)
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

			// put output inside clipboard
			goos := runtime.GOOS
			useXclip := viper.GetBool("use_xclip")
			if _, err := exec.LookPath("xclip"); err != nil {
				log.Logger().Println("xclip not found, falling back to \"golang.design/x/clipboard\"")
				useXclip = false
			}
			if goos == "linux" && useXclip {
				// NB since golang.design/x/clipboard doesn't always
				// write successfully to the clipboard and supports only x11 (but not Wayland)
				// we use this custom working-around based on xclip instead
				err = clipboard.Write(output)
				if err != nil {
					log.Logger().Fatal(err)
				}

				text, err := clipboard.Read()
				if err != nil {
					log.Logger().Fatal(err)
				}
				if shouldPaste {
					fmt.Print(text)
				}
			} else {
				if err := gclip.Init(); err != nil {
					log.Logger().Fatal(err)
				}
				gclip.Write(gclip.FmtText, []byte(output))
				text := gclip.Read(gclip.FmtText)
				if shouldPaste {
					fmt.Print(string(text))
				}
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
			db, err := storage.NewRepository(dbPath, false)
			if err != nil {
				log.Logger().Fatal(err)
			}

			defer func() {
				if closeErr := db.Close(); closeErr != nil {
					log.Logger().Println(closeErr)
				}
			}()

			if err := db.Reset(); err != nil {
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
		log.Logger().Fatal(err)
	}
	viper.SetDefault("use_xclip", true)

	rootCmd.AddCommand(listHistoryCmd)
	rootCmd.AddCommand(clearHistoryCmd)
}
