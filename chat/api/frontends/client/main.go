package main

import (
	"fmt"
	"os"

	"github.com/ardanlabs/usdl/chat/api/frontends/client/app"
	"github.com/ardanlabs/usdl/chat/api/frontends/client/app/storage/dbfile"
)

const (
	url            = "ws://localhost:3000/connect"
	configFilePath = "chat/zarf/client"
)

func main() {
	if err := run(); err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	id, privateKey, err := app.NewID(configFilePath)
	if err != nil {
		return fmt.Errorf("id: %w", err)
	}

	db, err := dbfile.NewDB(configFilePath, id)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	client := app.NewClient(id, privateKey, url, db)
	defer client.Close()

	a := app.New(client, db)

	if err := client.Handshake(db.MyAccount().Name, a.WriteText, a.UpdateContact); err != nil {
		return fmt.Errorf("handshake: %w", err)
	}

	if err := a.Run(); err != nil {
		return fmt.Errorf("run: %w", err)
	}

	return nil
}
