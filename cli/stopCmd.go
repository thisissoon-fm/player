package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"player/event"
	"player/sockets/unix"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the player if it's playing",
	Run: func(cmd *cobra.Command, args []string) {
		config := unix.NewConfig()
		client := unix.NewClient()
		if err := client.Connect(config.Address()); err != nil {
			fmt.Println("Unable to connect to player:", err)
			return
		}
		defer client.Close()
		eb, err := json.Marshal(&event.Event{
			Type:    event.StopEvent,
			Created: time.Now().UTC(),
		})
		if err != nil {
			fmt.Println("Unable to create stop event:", err)
			return
		}
		fmt.Println("Stop Player...")
		if _, err := client.Write(eb); err != nil {
			fmt.Println("Unable to send stop event:", err)
			return
		}
		fmt.Println("Done")
	},
}
