package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kaliv0/homie/internal/clipboard"
	"github.com/kaliv0/homie/internal/config"
	"github.com/kaliv0/homie/internal/finder"
	"github.com/kaliv0/homie/internal/runtime"
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
				runtime.Logger.Print(err)
			}

			// limit is read in order:
			//'--limit <n>' cli flag -> .homierc  -> Flags().IntP() default val
			limit := viper.GetInt("limit")
			if limit <= 0 {
				limit = storage.DefaultLimit
			}

			shouldPaste, err := cmd.Flags().GetBool("paste")
			if err != nil {
				runtime.Logger.Fatal(err)
			}

			// fetch history-to-be-displayed
			dbPath, err := config.DBPath()
			if err != nil {
				runtime.Logger.Fatal(err)
			}
			output, err := finder.ListHistory(dbPath, limit)
			if err != nil {
				runtime.Logger.Fatal(err)
			}
			if len(output) == 0 {
				return
			}

			// put output inside clipboard
			err = clipboard.WriteText(output)
			if err != nil {
				runtime.Logger.Fatal(err)
			}

			text, err := clipboard.Read()
			if err != nil {
				runtime.Logger.Fatal(err)
			}
			if shouldPaste {
				fmt.Print(text)
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
				runtime.Logger.Fatal(err)
			}
			db, err := storage.NewRepository(dbPath, false)
			if err != nil {
				runtime.Logger.Fatal(err)
			}

			defer func() {
				if closeErr := db.Close(); closeErr != nil {
					runtime.Logger.Print(closeErr)
				}
			}()

			if err := db.Reset(); err != nil {
				runtime.Logger.Fatal(err)
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
		runtime.Logger.Fatal(err)
	}
	rootCmd.AddCommand(listHistoryCmd)
	rootCmd.AddCommand(clearHistoryCmd)
}
