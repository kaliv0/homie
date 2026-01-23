package cmd

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	//go:embed scripts/shell_config
	bashConfig string

	generateShellConfigCmd = &cobra.Command{
		Use:   "shell",
		Short: "Generate a shell integration script",
		Long: fmt.Sprintf(`To enable shell integration execute:
$ source <(homie shell | tee -a "$HOME/.bashrc")`),
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.SetOut(os.Stdout)
			cmd.Println(bashConfig)
		},
	}
)

func init() {
	rootCmd.AddCommand(generateShellConfigCmd)
}
