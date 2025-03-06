package main

import (
	"fmt"
	"os"

	"github.com/ardanlabs/usdl/chat/api/tooling/client/app"
)

/*
 SAMPLE CONFIG FILE : chat/zarf/config.json
	{
		"user": {
			"id": "<user_id>",
			"name": "<user_name>"
		},
		"contacts": [
			{
				"id": "20723",
				"name": "Kevin Enriquez"
			},
			{
				"id": "58365",
				"name": "Bill Kennedy"
			}
		]
	}
*/

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

	a := app.New(client, cfg)

	uiWrite := func(name string, msg string) {
		a.WriteText(name, msg)
	}

	uiUpdateContact := func(id string, name string) {
		a.UpdateContact(id, name)
	}

	if err := client.Handshake(name, uiWrite, uiUpdateContact); err != nil {
		return fmt.Errorf("handshake: %w", err)
	}

	a.WriteText("system", "CONNECTED")

	if err := a.Run(); err != nil {
		return fmt.Errorf("run: %w", err)
	}

	return nil
}
