package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func main() {
	if err := hack1(); err != nil {
		log.Fatal(err)
	}
}

func hack1() error {
	const url = "ws://localhost:3000/connect"
	req := make(http.Header)

	users := []uuid.UUID{
		uuid.MustParse("8ce5af7a-788c-4c83-8e70-4500b775b359"),
		uuid.MustParse("8a45ec7a-273c-430a-9d90-ac30f94000cd"),
	}

	var ID uuid.UUID

	switch os.Args[1] {
	case "0":
		ID = users[0]
	case "1":
		ID = users[1]
	}

	fmt.Println("ID:", ID)

	socket, _, err := websocket.DefaultDialer.Dial(url, req)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	defer socket.Close()

	// -------------------------------------------------------------------------

	_, msg, err := socket.ReadMessage()
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	if string(msg) != "HELLO" {
		return fmt.Errorf("unexpected message: %s", msg)
	}

	// -------------------------------------------------------------------------

	user := struct {
		ID   uuid.UUID
		Name string
	}{
		ID:   ID,
		Name: "Bill Kennedy",
	}

	data, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	if err := socket.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	// -------------------------------------------------------------------------

	_, msg, err = socket.ReadMessage()
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	fmt.Println(string(msg))

	// -------------------------------------------------------------------------

	go func() {
		for {
			_, msg, err = socket.ReadMessage()
			if err != nil {
				fmt.Printf("read: %s\n", err)
				return
			}

			var outMsg outMessage
			if err := json.Unmarshal(msg, &outMsg); err != nil {
				fmt.Printf("Unmarshal: %s\n", err)
				return
			}

			fmt.Printf("\n%s\n", outMsg.Msg)
		}
	}()

	// -------------------------------------------------------------------------

	for {
		fmt.Print("\n\n")
		fmt.Print("message >")

		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read input: %w", err)
		}

		var from uuid.UUID
		var to uuid.UUID

		switch os.Args[1] {
		case "0":
			from = users[0]
			to = users[1]
		case "1":
			from = users[1]
			to = users[0]
		}

		inMsg := inMessage{
			FromID: from,
			ToID:   to,
			Msg:    input,
		}

		data2, err := json.Marshal(inMsg)
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}

		if err := socket.WriteMessage(websocket.TextMessage, data2); err != nil {
			return fmt.Errorf("write: %w", err)
		}
	}
}

type inMessage struct {
	FromID uuid.UUID `json:"fromID"`
	ToID   uuid.UUID `json:"toID"`
	Msg    string    `json:"msg"`
}

type user struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type outMessage struct {
	From user   `json:"from"`
	To   user   `json:"to"`
	Msg  string `json:"msg"`
}
