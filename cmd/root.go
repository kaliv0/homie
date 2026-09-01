package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kaliv0/homie/internal/config"
	"github.com/kaliv0/homie/internal/log"
)

var cfg *config.Config

var rootCmd = &cobra.Command{
	Use:   "homie",
	Short: "Terminal-based clipboard manager",
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := config.ReadConfig(); err != nil {
			return err
		}

		var err error
		cfg, err = config.Parse(viper.GetViper())
		if err != nil {
			return err
		}
		log.Configure(cfg.Verbose, cfg.LogFile)
		return nil
	},
	Long: `
		⣿⣿⣿⣿⣿⣿⣿⣿⣿⠟⠏⢀⣀⣤⣤⣤⣤⣤⣤⣄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
		⣿⣿⣿⣿⣿⣿⡿⣿⣴⢶⣶⣿⣟⣶⣿⣭⠿⠦⠤⠽⣷⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
		⣿⣿⣿⣿⡿⢫⣿⢋⣠⣿⣿⡶⢻⡏⠀⠀⠀⠀⠀⠀⠀⠉⠙⢦⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
		⣿⣿⡿⠋⠈⣸⣿⣿⣿⡿⠿⠀⠈⠃⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠙⣄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
		⢯⠋⠈⠀⣴⣿⣿⣿⣿⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠸⡄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
		⠒⠀⠀⢰⣿⣿⣿⣿⡇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢷⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
		⠀⠀⠀⢼⣿⣿⣿⣿⡇⠀⠀⠀⠀⠀⢀⡀⠤⠤⠤⣀⠀⢀⡀⠤⠤⠤⣀⣱⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
		⠀⠀⠀⢸⣿⣿⣿⣿⣿⠀⠀⠀⢀⡖⠁⠀⠀⠀⠀⠀⠱⡏⠀⠀⠀⠀⠈⠱⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⠀⠀⠀⠀
		⠀⠀⠀⠈⣿⣿⣿⣿⣿⡆⠀⠀⢸⠀⠀⠴⠆⠀⠀⠀⠀⠀⠀⠀⠀⠀⠶⠀⡇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣀⢀⡴⠋⠉⢹⡶⠶⢤
		⠀⠀⠀⠀⢸⣿⣿⣿⣿⣿⣸⢻⡜⡄⠀⠀⠀⠀⠀⠀⢀⠶⠒⠒⠀⠐⣄⡼⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣾⣿⣿⠟⠛⠓⠶⣏⠀⠀⣀
		⠀⠀⠀⢸⢧⣿⣿⣿⣿⣿⡿⠀⠷⠙⠲⠄⠀⡀⠠⠔⠁⠀⠀⠀⢀⣠⡇⡧⠀⠀⠀⠀⠀⠀⠀⠀⠀⣴⣿⣿⣿⡇⠀⡀⠀⠀⠈⢦⠞⠁
		⠀⠀⠀⢸⡈⢻⣿⣿⣿⡿⠧⣄⠀⠀⠀⢀⡴⠖⠒⠚⠛⠛⠛⠛⠉⠀⠈⠙⠦⣀⣠⣀⠀⠀⠀⠀⢰⣿⣿⣿⣿⠄⠀⠈⢢⡀⠠⢾⠀⠀
		⠀⠀⠀⠀⣴⣾⣿⣿⣿⢲⡶⡄⠀⢀⡶⠋⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢁⠀⢸⠀⠀⠀⠀⣸⣿⣿⠟⠋⠀⠀⠀⡎⠀⠀⠈⠉⠀
		⡀⠀⠀⠘⣿⣿⣿⣿⣿⣦⣤⡴⠀⣾⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢰⡷⢰⠃⠀⠀⣠⣾⣿⠟⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠
		⠙⢦⡀⠀⠈⠛⢻⣿⣿⣿⣿⡇⠀⣿⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢦⣀⣠⠎⢀⣤⣾⡿⠋⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⠞⠀
		⠀⠀⠈⠳⢄⠀⢸⣿⣿⣿⣿⡇⠀⠙⣆⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⡴⠃⠀⣠⣴⣿⠟⠋⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⡰⠁⠀⠀
		⠀⠀⠀⠀⠈⠙⢾⣿⣿⣿⣿⡇⠀⠀⠈⠳⠤⣀⡀⠀⠀⢀⣀⠤⡖⠋⢀⡤⠾⠿⣏⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⠎⠀⠀⠀⠀
		⠀⠀⠀⠀⠀⠀⠀⢨⠇⠙⢿⠷⠖⠒⠛⠓⠒⠚⠛⠯⡉⠉⠀⠀⡷⠶⠯⡁⠀⠀⠀⠙⠢⡀⠀⠀⠀⠀⠀⠀⠀⠀⢀⠜⠁⠀⠀⠀⠀⠀
		⠀⠀⠀⠀⠀⠀⢀⡌⠀⠀⠀⠳⡀⠀⠀⠀⠀⠀⡌⠀⠙⢆⠀⠀⡧⠂⠀⢡⠀⠀⠀⠀⠀⠈⠢⡀⠀⠀⠀⠀⢀⡔⠁⠀⠀⠀⠀⠀⠀⠀
		⠀⠀⠀⠀⠀⢀⠌⠀⠀⠀⠀⠀⠰⡀⠀⠀⠀⢰⠃⠀⠀⠈⠣⡴⠉⠡⡀⠈⡆⠀⠀⠀⠀⠀⠀⠘⠄⠀⢀⡴⠊⠀⠀⠀⠀⠀⠀⠀⠀⠀
		⠀⠀⠀⠀⡠⠊⠀⠀⠀⠀⠀⠀⠀⢩⠉⠉⠉⠉⠀⠀⠀⠀⠀⠀⠀⠀⠱⡀⠁⠀⠀⠀⠀⠀⠀⠀⠈⣶⠊⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
		⣷⣄⣠⠞⠁⠀⠀⠀⠀⠀⠀⠀⠀⠈⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢡⠀⠀⠀⠀⠀⠀⠀⠀⢠⠎⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
		⣿⣎⡁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⡆⠀⠀⠀⠀⢀⡠⠞⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
		⣿⣿⣿⣶⣤⣀⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠠⢤⡤⠴⠒⠊⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
		⠻⠿⢿⣿⣿⣿⣿⠏⠉⠉⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠉⠢⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
`}

func init() {
	rootCmd.PersistentFlags().BoolP(config.KeyVerbose, "v", false, "enable verbose diagnostic messages")
	rootCmd.PersistentFlags().String(config.FlagLogFile, "", "append logs to file")

	config.BindFlag(config.KeyVerbose, rootCmd.PersistentFlags().Lookup(config.KeyVerbose))
	config.BindFlag(config.KeyLogFile, rootCmd.PersistentFlags().Lookup(config.FlagLogFile))
}

// Execute runs the root cobra command.
func Execute() error {
	return rootCmd.Execute()
}
