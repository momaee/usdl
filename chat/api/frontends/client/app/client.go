package app

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/websocket"
)

type UIScreenWrite func(id string, msg string)
type UIUpdateContact func(id string, name string)

// =============================================================================

type inMessage struct {
	ToID common.Address `json:"toID"`
	Msg  string         `json:"msg"`
}

type user struct {
	ID   common.Address `json:"id"`
	Name string         `json:"name"`
}

type outMessage struct {
	From user   `json:"from"`
	Msg  string `json:"msg"`
}

// =============================================================================

type Client struct {
	id       common.Address
	url      string
	contacts *Contacts
	conn     *websocket.Conn
	uiWrite  UIScreenWrite
}

func NewClient(id common.Address, url string, contacts *Contacts) *Client {
	return &Client{
		id:       id,
		url:      url,
		contacts: contacts,
	}
}

func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}

	return c.conn.Close()
}

func (c *Client) Handshake(name string, uiWrite UIScreenWrite, uiUpdateContact UIUpdateContact) error {
	conn, _, err := websocket.DefaultDialer.Dial(c.url, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	c.conn = conn
	c.uiWrite = uiWrite

	// -------------------------------------------------------------------------

	_, msg, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	if string(msg) != "HELLO" {
		return fmt.Errorf("unexpected message: %s", msg)
	}

	// -------------------------------------------------------------------------

	user := struct {
		ID   common.Address
		Name string
	}{
		ID:   c.id,
		Name: name,
	}

	data, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	// -------------------------------------------------------------------------

	_, msg, err = conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	// -------------------------------------------------------------------------

	go func() {
		for {
			_, msg, err = conn.ReadMessage()
			if err != nil {
				uiWrite("system", fmt.Sprintf("read: %s", err))
				return
			}

			var outMsg outMessage
			if err := json.Unmarshal(msg, &outMsg); err != nil {
				uiWrite("system", fmt.Sprintf("unmarshal: %s", err))
				return
			}

			user, err := c.contacts.LookupContact(outMsg.From.ID)
			switch {
			case err != nil:
				if err := c.contacts.AddContact(outMsg.From.ID, outMsg.From.Name); err != nil {
					uiWrite("system", fmt.Sprintf("add contact: %s", err))
					return
				}

				uiUpdateContact(outMsg.From.ID.Hex(), outMsg.From.Name)

			default:
				outMsg.From.Name = user.Name
			}

			msg := formatMessage(user.Name, outMsg.Msg)

			if err := c.contacts.AddMessage(outMsg.From.ID, msg); err != nil {
				uiWrite("system", fmt.Sprintf("add message: %s", err))
				return
			}

			uiWrite(outMsg.From.ID.Hex(), msg)
		}
	}()

	return nil
}

func (c *Client) Send(to common.Address, msg string) error {
	if c.conn == nil {
		return fmt.Errorf("no connection")
	}

	inMsg := inMessage{
		ToID: to,
		Msg:  msg,
	}

	data, err := json.Marshal(inMsg)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	msg = formatMessage("You", msg)

	if err := c.contacts.AddMessage(to, msg); err != nil {
		return fmt.Errorf("add message: %w", err)
	}

	c.uiWrite(to.Hex(), msg)

	return nil
}
