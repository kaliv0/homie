package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kaliv0/homie/assets/ascii"
	"github.com/kaliv0/homie/internal/config"
	"github.com/kaliv0/homie/internal/log"
)

var cfg *config.Config

var rootCmd = &cobra.Command{
	Use:           "homie",
	Short:         "Terminal-based clipboard manager",
	Version:       "1.16.1",
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		switch runtime.GOOS {
		case "linux", "darwin":
		default:
			return fmt.Errorf("homie requires Linux or macOS (running on %s)", runtime.GOOS)
		}

		if err := config.ReadConfig(); err != nil {
			return err
		}

		cfg = config.Parse(viper.GetViper())
		log.Configure(cfg.Verbose, cfg.LogFile)
		return nil
	},
}

// Execute runs the root cobra command. Fatal stays here so RunE handlers can return errors.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	help := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd == rootCmd {
			// print Banner only for 'homie -h'
			fmt.Fprintln(cmd.OutOrStdout(), ascii.Banner)
		}
		help(cmd, args)
	})
}
