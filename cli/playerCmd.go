package cli

import (
	"fmt"

	"player/config"
	"player/event"
	"player/logger"
	"player/player"
	"player/providers/googlemusic"
	"player/providers/soundcloud"
	"player/providers/spotify"
	"player/run"
	"player/sockets/unix"
	"player/sockets/web"

	"github.com/spf13/cobra"
)

// "Tatvqym3fp2pklprl3ll7top4ru"
// "Taw3eomospsfhb2r5qyyewlrc2q"
// "Tnfy7jvelislcpk2ahmbqicqxca"

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
		defer logger.Info("exit")
		// Event Hub
		go event.ProcessEvents()
		defer event.Close()
		// Websocket Client
		websocket := web.New(web.NewConfig())
		go websocket.Connect()
		defer websocket.Close()
		// Start a unix socket server for IPC
		unixsock := unix.NewServer(unix.NewConfig())
		go unixsock.Listen()
		defer unixsock.Close()
		// Google Music Provider
		gmp, err := googlemusic.New(googlemusic.NewConfig())
		if err != nil {
			fmt.Println(err)
			return
		}
		player.AddProvider(gmp)
		// // SoundCloud Provider
		scp := soundcloud.New(soundcloud.NewConfig())
		player.AddProvider(scp)
		// Spotify Provider
		sp, err := spotify.New(spotify.NewConfig())
		if err != nil {
			fmt.Println(err)
			return
		}
		defer sp.Close()
		player.AddProvider(sp)
		// Close the player on exit
		defer player.Close()
		// Run until os signal
		sig := <-run.UntilQuit()
		logger.WithField("sig", sig).Debug("received os signal")
	},
}

func init() {
	playerCmd.PersistentFlags().StringVarP(
		&configPath,
		"config",
		"c",
		"",
		"Optional absolute path to toml config file")
	playerCmd.AddCommand(buildCmd, playCmd, stopCmd, pauseCmd, resumeCmd)
}

func Run() error {
	return playerCmd.Execute()
}
