package cmd

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

//go:embed scripts/tmux_config
var tmuxConfig string

var generateTmuxConfigCmd = &cobra.Command{
	Use:   "tmux",
	Short: "Generate a tmux integration script",
	Long: fmt.Sprintf(`To enable tmux integration append to your .tmux.conf:
$ %s tmux >> "$HOME/.tmux.conf"

Then reload from inside a running tmux session:
$ tmux source-file "$HOME/.tmux.conf"

Requires tmux 3.2+ (for display-popup)`, rootCmd.Root().Name()),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, _ []string) {
		cmd.SetOut(os.Stdout)
		cmd.Println(tmuxConfig)
	},
}

func init() {
	rootCmd.AddCommand(generateTmuxConfigCmd)
}
