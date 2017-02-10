package cli

import (
	"fmt"

	"player/build"

	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Print build version and time",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("OS:", build.OS())
		fmt.Println("Architecture:", build.Architecture())
		fmt.Println("Version:", build.Version())
		fmt.Println("Time:", build.TimeStr())
	},
}
