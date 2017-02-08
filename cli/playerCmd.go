package cli

import (
	"fmt"

	"player/config"
	"player/player"
	"player/player/googlemusic"
	"player/player/soundcloud"

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
		err = player.Play(gm.Name(), "Tb56fzzbswo6iqf34dghtaywjni")
		if err != nil {
			fmt.Println(err)
			return
		}
		err = player.Play(sc.Name(), "295322075")
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
}

func Run() error {
	return playerCmd.Execute()
}
