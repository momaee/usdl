package app

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ardanlabs/usdl/chat/api/frontends/client/app/storage/dbfile"
	"github.com/ardanlabs/usdl/chat/foundation/signature"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/websocket"
)

type UIScreenWrite func(id string, msg string)
type UIUpdateContact func(id string, name string)

// =============================================================================

type outgoingMessage struct {
	ToID  common.Address `json:"toID"`
	Msg   string         `json:"msg"`
	Nonce uint64         `json:"nonce"`
	V     *big.Int       `json:"v"`
	R     *big.Int       `json:"r"`
	S     *big.Int       `json:"s"`
}

type user struct {
	ID   common.Address `json:"id"`
	Name string         `json:"name"`
}

type incomingMessage struct {
	From user   `json:"from"`
	Msg  string `json:"msg"`
}

// =============================================================================

type Client struct {
	id         common.Address
	privateKey *ecdsa.PrivateKey
	url        string
	db         *dbfile.DB
	conn       *websocket.Conn
	uiWrite    UIScreenWrite
}

func NewClient(id common.Address, privateKey *ecdsa.PrivateKey, url string, db *dbfile.DB) *Client {
	return &Client{
		id:         id,
		privateKey: privateKey,
		url:        url,
		db:         db,
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

			var inMsg incomingMessage
			if err := json.Unmarshal(msg, &inMsg); err != nil {
				uiWrite("system", fmt.Sprintf("unmarshal: %s", err))
				return
			}

			user, err := c.db.QueryContactByID(inMsg.From.ID)
			switch {
			case err != nil:
				user, err = c.db.InsertContact(inMsg.From.ID, inMsg.From.Name)
				if err != nil {
					uiWrite("system", fmt.Sprintf("add contact: %s", err))
					return
				}

				uiUpdateContact(inMsg.From.ID.Hex(), inMsg.From.Name)

			default:
				inMsg.From.Name = user.Name
			}

			msg := formatMessage(user.Name, inMsg.Msg)

			if err := c.db.InsertMessage(inMsg.From.ID, msg); err != nil {
				uiWrite("system", fmt.Sprintf("add message: %s", err))
				return
			}

			uiWrite(inMsg.From.ID.Hex(), msg)
		}
	}()

	return nil
}

func (c *Client) Send(to common.Address, msg string) error {
	if c.conn == nil {
		return fmt.Errorf("no connection")
	}

	dataToSign := struct {
		ToID  common.Address
		Msg   string
		Nonce uint64
	}{
		ToID:  to,
		Msg:   msg,
		Nonce: 1,
	}

	v, r, s, err := signature.Sign(dataToSign, c.privateKey)
	if err != nil {
		return fmt.Errorf("signing: %w", err)
	}

	outMsg := outgoingMessage{
		ToID:  to,
		Msg:   msg,
		Nonce: 1,
		V:     v,
		R:     r,
		S:     s,
	}

	data, err := json.Marshal(outMsg)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	msg = formatMessage("You", msg)

	if err := c.db.InsertMessage(to, msg); err != nil {
		return fmt.Errorf("add message: %w", err)
	}

	c.uiWrite(to.Hex(), msg)

	return nil
}
