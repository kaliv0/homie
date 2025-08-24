package cmd

import (
	"fmt"

	"github.com/kaliv0/homie/cmd/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.design/x/clipboard"
)

var (
	listHistoryCmd = &cobra.Command{
		Use:   "history",
		Short: "List clipboard history",
		Long: `List clipboard history
  Use <tab> to pin and select multiple entries`,
		Run: func(cmd *cobra.Command, _ []string) {
			// read flags
			utils.ReadConfig()
			// limit is read in order:
			//'--limit <n>' cli flag -> .homierc  -> Flags().IntP() default val
			limit := viper.GetInt("limit")
			shouldPaste, err := cmd.Flags().GetBool("paste")
			if err != nil {
				utils.Logger.Fatal(err)
			}

			// fetch history-to-be-displayed
			dbPath := utils.GetDbPath()
			output := utils.ListHistory(dbPath, limit)

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
			dbPath := utils.GetDbPath()
			db := utils.NewRepository(dbPath, false)
			defer func() {
				db.Close()
			}()
			db.Reset()
		},
	}
)

func init() {
	listHistoryCmd.Flags().IntP(
		"limit",
		"l",
		20,
		"Limit the number of clipboard history items displayed",
	)
	listHistoryCmd.Flags().BoolP(
		"paste",
		"p",
		false,
		"Paste selected history item",
	)
	if err := viper.BindPFlag("limit", listHistoryCmd.Flags().Lookup("limit")); err != nil {
		utils.Logger.Fatal(err)
	}
	rootCmd.AddCommand(listHistoryCmd)
	rootCmd.AddCommand(clearHistoryCmd)
}
