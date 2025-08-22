package cmd

import (
	"fmt"
	"os"

	"github.com/kaliv0/homey/cmd/utils"
	"github.com/spf13/cobra"
)

var (
	completionCmd = &cobra.Command{
		Use:   "completion",
		Short: "Generate completion script",
		Long: fmt.Sprintf(`To load completions execute:
$ source <(%s completion | tee -a "$HOME/.bash_completion")`, rootCmd.Root().Name()),
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, _ []string) {
			err := cmd.Root().GenBashCompletion(os.Stdout)
			if err != nil {
				utils.Logger.Fatal(err)
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(completionCmd)
}
