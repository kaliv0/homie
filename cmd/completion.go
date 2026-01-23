package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kaliv0/homie/internal/log"
)

var (
	completionCmd = &cobra.Command{
		Use:   "completion",
		Short: "Generate completion script",
		Long: fmt.Sprintf(`To load completions execute:
$ source <(homie completion | tee -a "$HOME/.bash_completion")`),
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, _ []string) {
			if err := cmd.Root().GenBashCompletion(os.Stdout); err != nil {
				log.Logger().Fatalf("failed to generate bash completion: %v", err)
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(completionCmd)
}
