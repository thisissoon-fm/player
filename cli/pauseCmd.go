package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"player/event"
	"player/sockets/unix"

	"github.com/spf13/cobra"
)

var pauseCmd = &cobra.Command{
	Use:   "pause",
	Short: "Pauses the player",
	Run: func(cmd *cobra.Command, args []string) {
		config := unix.NewConfig()
		client := unix.NewClient()
		if err := client.Connect(config.Address()); err != nil {
			fmt.Println("Unable to connect to player:", err)
			return
		}
		defer client.Close()
		eb, err := json.Marshal(&event.Event{
			Type:    event.PauseEvent,
			Created: time.Now().UTC(),
		})
		if err != nil {
			fmt.Println("Unable to create pause event:", err)
			return
		}
		fmt.Println("Pausing Player...")
		if _, err := client.Write(eb); err != nil {
			fmt.Println("Unable to send pause event:", err)
			return
		}
		fmt.Println("Done")
	},
}
