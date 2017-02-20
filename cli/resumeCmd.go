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

var resumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume the player",
	Run: func(cmd *cobra.Command, args []string) {
		defer fmt.Println("Done")
		config := unix.NewConfig()
		client := unix.NewClient()
		if err := client.Connect(config.Address()); err != nil {
			fmt.Println("Unable to connect to player:", err)
			return
		}
		defer client.Close()
		eb, err := json.Marshal(&event.Event{
			Type:    event.ResumeEvent,
			Created: time.Now().UTC(),
		})
		if err != nil {
			fmt.Println("Unable to create resume event:", err)
			return
		}
		fmt.Println("Resume Player...")
		if _, err := client.Write(eb); err != nil {
			fmt.Println("Unable to send resume event:", err)
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
					fmt.Println("Playback resumed")
					return
				case event.ErrorEvent:
					payload := &event.ErrorPayload{}
					if err := json.Unmarshal(e.Payload, payload); err != nil {
						fmt.Println("Unable to process error")
					}
					fmt.Println("Error resuming player:", payload.Error)
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
