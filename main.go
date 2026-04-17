package main

import (
	"github.com/spf13/cobra"

	"github.com/kaliv0/homie/cmd"
)

func main() {
	cobra.CheckErr(cmd.Execute())
}
