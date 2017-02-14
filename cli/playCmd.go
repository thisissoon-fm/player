package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"player/event"
	"player/run"
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
		if playCmdProvider == "" || playCmdTrackID == "" {
			fmt.Println("Need a provider and a track ID")
			return
		}
		payload, err := json.Marshal(&event.PlayPayload{
			ProviderID: playCmdProvider,
			TrackID:    playCmdTrackID,
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
		if _, err := client.Write(body); err != nil {
			fmt.Println("Unable to send play event:", err)
			return
		}
		exitC := make(chan bool)
		go func() {
			defer close(exitC)
			for {
				b, err := client.Read()
				if err != nil {
					return
				}
				e := &event.Event{}
				if err := json.Unmarshal(b, e); err != nil {
					fmt.Println("error reading event:", err)
				}
				switch e.Type {
				case event.PlayingEvent:
					fmt.Println("Playing track")
					return
				case event.ErrorEvent:
					payload := &event.ErrorPayload{}
					if err := json.Unmarshal(e.Payload, payload); err != nil {
						fmt.Println("Unable to process error")
					}
					fmt.Println("Error playing track:", payload.Error)
					return
				}
			}
		}()
		deadline := time.Second * 30
		select {
		case <-exitC:
			return
		case <-run.UntilQuit():
			return
		case <-time.After(deadline):
			fmt.Println("no response from player after", deadline)
			return
		}
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
