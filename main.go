package main

import (
	"fmt"

	"player/cli"
)

// Application Entry Point
func main() {
	if err := cli.Run(); err != nil {
		fmt.Println(err)
	}
}
