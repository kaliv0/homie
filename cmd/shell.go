package cmd

import (
	_ "embed"
	"os"

	"github.com/spf13/cobra"
)

var (
	//go:embed scripts/shell_config
	bashConfig string

	generateShellConfigCmd = &cobra.Command{
		Use:   "shell",
		Short: "Generate a shell integration script",
		Long: `To enable shell integration execute:
$ source <(homie shell | tee -a "$HOME/.bashrc")`,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SetOut(os.Stdout)
			cmd.Println(bashConfig)
			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(generateShellConfigCmd)
}
