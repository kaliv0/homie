package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	completionCmd = &cobra.Command{
		Use:   "completion",
		Short: "Generate completion script",
		Long: `To load completions execute:
$ source <(homie completion | tee -a "$HOME/.bash_completion")`,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := cmd.Root().GenBashCompletion(os.Stdout); err != nil {
				return fmt.Errorf("failed to generate bash completion: %w", err)
			}
			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(completionCmd)
}
