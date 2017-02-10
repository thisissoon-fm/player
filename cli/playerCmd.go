package cli

import (
	"fmt"

	"player/config"
	"player/logger"
	"player/player"
	"player/services/googlemusic"
	"player/services/soundcloud"

	"github.com/spf13/cobra"
)

var (
	configPath string
)

var playerCmd = &cobra.Command{
	Use:   "player",
	Short: "SOON_ FM Music Player",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if err := config.Read(configPath); err != nil {
			fmt.Println(err) // TODO: Warning Log
		}
		logger.SetGlobalLogger(logger.New(logger.NewConfig()))
	},
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		gm, err := googlemusic.New(googlemusic.NewConfig())
		if err != nil {
			fmt.Println(err)
			return
		}
		sc := soundcloud.New(soundcloud.NewConfig())
		player, err := player.New(gm, sc)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer player.Close()
		err = player.Play(gm.Name(), "Tjqq2dasnnsnjwxjvu7s2hqdkpq")
		if err != nil {
			fmt.Println(err)
			return
		}
	},
}

func init() {
	playerCmd.PersistentFlags().StringVarP(
		&configPath,
		"config",
		"c",
		"",
		"Optional absolute path to toml config file")
	playerCmd.AddCommand(buildCmd)
}

func Run() error {
	return playerCmd.Execute()
}
