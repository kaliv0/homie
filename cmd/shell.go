package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	//go:embed scripts/bash_config
	bashConfig string
	//go:embed scripts/zsh_config
	zshConfig string

	generateShellConfigCmd = &cobra.Command{
		Use:   "shell [bash|zsh]",
		Short: "Generate a shell integration script",
		Long: `To enable shell integration execute:

  bash:
    echo 'source <(homie shell bash)' >> ~/.bashrc

  zsh:
    echo 'source <(homie shell zsh)' >> ~/.zshrc

Then reload your shell.`,
		DisableFlagsInUseLine: true,
		Args:                  requireShellArg,
		ValidArgs:             []string{"bash", "zsh"},
		RunE: func(cmd *cobra.Command, args []string) error {
			shell, err := resolveShell(args[0])
			if err != nil {
				return err
			}

			cmd.SetOut(os.Stdout)
			switch shell {
			case "bash":
				cmd.Println(bashConfig)
			case "zsh":
				cmd.Println(zshConfig)
			}
			return nil
		},
	}
)

func requireShellArg(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("shell required: bash or zsh")
	} else if len(args) > 1 {
		return fmt.Errorf("only single argument required")
	}
	return nil
}

func resolveShell(name string) (string, error) {
	switch strings.ToLower(name) {
	case "bash", "zsh":
		return strings.ToLower(name), nil
	default:
		return "", fmt.Errorf("unsupported shell %q (want bash or zsh)", name)
	}
}

func init() {
	rootCmd.AddCommand(generateShellConfigCmd)
}
