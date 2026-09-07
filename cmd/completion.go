package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	completionCmd = &cobra.Command{
		Use:   "completion [bash|zsh]",
		Short: "Generate completion script",
		Long: `To load completions execute:

  bash:
    echo 'source <(homie completion bash)' >> ~/.bashrc

  zsh:
    echo 'source <(homie completion zsh)' >> ~/.zshrc

Then reload your shell.`,
		DisableFlagsInUseLine: true,
		Args:                  requireShellArg,
		ValidArgs:             []string{"bash", "zsh"},
		RunE: func(cmd *cobra.Command, args []string) error {
			shell, err := resolveShell(args[0])
			if err != nil {
				return err
			}

			switch shell {
			case "bash":
				if err := cmd.Root().GenBashCompletion(os.Stdout); err != nil {
					return fmt.Errorf("failed to generate bash completion: %w", err)
				}
			case "zsh":
				if err := cmd.Root().GenZshCompletion(os.Stdout); err != nil {
					return fmt.Errorf("failed to generate zsh completion: %w", err)
				}
			}
			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(completionCmd)
}
