package cli

import (
	"fmt"

	"player/config"
	"player/logger"
	"player/player"
	"player/providers/googlemusic"
	"player/providers/soundcloud"

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
			logger.WithError(err).Warn("unable to read config")
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
		player.AddStreamer(gm)
		sc := soundcloud.New(soundcloud.NewConfig())
		player.AddStreamer(sc)
		defer player.Close()
		tracks := []string{
			"T2bzzzgnjq3asx433qed2ip77iu",
			"Toxylvghchv3irywxuchgb2yrhe",
			"T7bqkzir7gjkazqcpyfgb2ifjua",
		}
		for _, t := range tracks {
			err = player.Play(gm.Name(), t)
			if err != nil {
				fmt.Println(err)
				return
			}
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
