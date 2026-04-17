package cmd

import (
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kaliv0/homie/internal/config"
	"github.com/kaliv0/homie/internal/log"
	"github.com/kaliv0/homie/internal/storage"
)

var writeCmd = &cobra.Command{
	Use:    "write",
	Hidden: true,
	Run: func(cmd *cobra.Command, _ []string) {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Logger().Fatalf("failed to read stdin: %v", err)
		}
		text := strings.TrimRight(string(data), "\n")
		if text == "" {
			return
		}

		if err := writeToClipboard(text); err != nil {
			log.Logger().Fatal(err)
		}

		dbPath, err := config.DBPath()
		if err != nil {
			log.Logger().Fatal(err)
		}
		db, err := storage.NewRepository(dbPath)
		if err != nil {
			log.Logger().Fatal(err)
		}

		defer func() {
			if closeErr := db.Close(); closeErr != nil {
				log.Logger().Println(closeErr)
			}
		}()

		if err := db.Write([]byte(text)); err != nil {
			log.Logger().Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(writeCmd)
}
