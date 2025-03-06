package main

import (
	"fmt"
	"os"

	"github.com/ardanlabs/usdl/chat/api/tooling/client/app"
)

const (
	url            = "ws://localhost:3000/connect"
	configFilePath = "chat/zarf/"
)

func main() {
	if err := run(); err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := app.NewConfig(configFilePath)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	id := cfg.User().ID
	name := cfg.User().Name

	client := app.NewClient(id, url, cfg)
	defer client.Close()

	app := app.New(client, cfg)

	uiScreenWrite := func(name string, msg string) {
		app.WriteText(name, msg)
	}

	if err := client.Handshake(name, uiScreenWrite); err != nil {
		return fmt.Errorf("handshake: %w", err)
	}

	app.WriteText("system", "CONNECTED")

	if err := app.Run(); err != nil {
		return fmt.Errorf("run: %w", err)
	}

	return nil
}
