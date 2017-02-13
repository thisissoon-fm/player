package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"player/event"
	"player/sockets/unix"

	"github.com/spf13/cobra"
)

var (
	playCmdProvider string
	playCmdTrackID  string
)

var playCmd = &cobra.Command{
	Use:   "play",
	Short: "Play a track",
	Run: func(cmd *cobra.Command, args []string) {
		config := unix.NewConfig()
		client := unix.NewClient()
		if err := client.Connect(config.Address()); err != nil {
			fmt.Println("Unable to connect to player:", err)
			return
		}
		defer client.Close()
		payload, err := json.Marshal(&event.PlayPayload{
			Provider: playCmdProvider,
			TrackID:  playCmdTrackID,
		})
		if err != nil {
			fmt.Println("Unable to create play payload:", err)
			return
		}
		body, err := json.Marshal(&event.Event{
			Type:    event.PlayEvent,
			Created: time.Now().UTC(),
			Payload: json.RawMessage(payload),
		})
		if err != nil {
			fmt.Println("Unable to create play event:", err)
			return
		}
		fmt.Println("Playing track...")
		if _, err := client.Write(body); err != nil {
			fmt.Println("Unable to send play event:", err)
			return
		}
		fmt.Println("Done")
	},
}

func init() {
	playCmd.PersistentFlags().StringVarP(
		&playCmdProvider,
		"provider",
		"p",
		"",
		"Track Provider (google, soundcloud etc)")
	playCmd.PersistentFlags().StringVarP(
		&playCmdTrackID,
		"track",
		"t",
		"",
		"Track ID")
}
