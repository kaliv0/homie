package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kaliv0/homie/internal"
)

var (
	completionCmd = &cobra.Command{
		Use:   "completion",
		Short: "Generate completion script",
		Long: fmt.Sprintf(`To load completions execute:
$ source <(%s completion | tee -a "$HOME/.bash_completion")`, rootCmd.Root().Name()),
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, _ []string) {
			if err := cmd.Root().GenBashCompletion(os.Stdout); err != nil {
				internal.Logger.Fatal(err)
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(completionCmd)
}
