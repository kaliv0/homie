package cmd

import (
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kaliv0/homie/internal/clipboard"
	"github.com/kaliv0/homie/internal/log"
)

// used as a workaround to enable copying inside tmux session
var writeCmd = &cobra.Command{
	Use:    "write",
	Hidden: true,
	Run: func(cmd *cobra.Command, _ []string) {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Logger().Fatalf("failed to read stdin: %v", err)
		}
		// tmux copy-pipe appends a trailing newline -> strip it before persist
		text := strings.TrimRight(string(data), "\n")
		// check if we need to persist at all
		if strings.TrimSpace(text) == "" {
			return
		}

		if err := clipboard.WriteSelection(text); err != nil {
			log.Logger().Fatal(err)
		}

		db, err := openDB()
		if err != nil {
			log.Logger().Fatal(err)
		}
		defer closeDB(db)

		if err := db.Write([]byte(text)); err != nil {
			log.Logger().Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(writeCmd)
}
