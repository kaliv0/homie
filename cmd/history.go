package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.design/x/clipboard"

	"github.com/kaliv0/homie/internal"
)

var (
	listHistoryCmd = &cobra.Command{
		Use:   "history",
		Short: "List clipboard history",
		Long: `List clipboard history
  Use <tab> to pin and select multiple entries`,
		Run: func(cmd *cobra.Command, _ []string) {
			// read flags
			internal.ReadConfig()
			// limit is read in order:
			//'--limit <n>' cli flag -> .homierc  -> Flags().IntP() default val
			limit := viper.GetInt("limit")
			if limit <= 0 {
				limit = internal.DefaultLimit
			}

			shouldPaste, err := cmd.Flags().GetBool("paste")
			if err != nil {
				internal.Logger.Fatal(err)
			}

			// fetch history-to-be-displayed
			dbPath := internal.DBPath()
			output := internal.ListHistory(dbPath, limit)
			if len(output) == 0 {
				return
			}
			// put output inside clipboard
			clipboard.Write(clipboard.FmtText, []byte(output))
			if shouldPaste {
				fmt.Print(string(clipboard.Read(clipboard.FmtText)))
			} else {
				clipboard.Read(clipboard.FmtText)
			}
		},
	}

	clearHistoryCmd = &cobra.Command{
		Use:                   "clear",
		Short:                 "Clear clipboard history",
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, _ []string) {
			dbPath := internal.DBPath()
			db := internal.NewRepository(dbPath, false)
			defer db.Close()
			db.Reset()
		},
	}
)

func init() {
	listHistoryCmd.Flags().IntP(
		"limit",
		"l",
		internal.DefaultLimit,
		"Limit the number of clipboard history items displayed",
	)
	listHistoryCmd.Flags().BoolP(
		"paste",
		"p",
		false,
		"Paste selected history item",
	)
	if err := viper.BindPFlag("limit", listHistoryCmd.Flags().Lookup("limit")); err != nil {
		internal.Logger.Fatal(err)
	}
	rootCmd.AddCommand(listHistoryCmd)
	rootCmd.AddCommand(clearHistoryCmd)
}
