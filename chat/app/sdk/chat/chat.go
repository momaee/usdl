// Package chat provides supports for chat activity.
package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ardanlabs/usdl/chat/foundation/logger"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var ErrFromNotExists = fmt.Errorf("from user doesn't exists")
var ErrToNotExists = fmt.Errorf("to user doesn't exists")

// Chat represents a chat support.
type Chat struct {
	log   *logger.Logger
	users map[uuid.UUID]connection
	mu    sync.RWMutex
}

// New creates a new chat support.
func New(log *logger.Logger) *Chat {
	c := Chat{
		log:   log,
		users: make(map[uuid.UUID]connection),
	}

	c.ping()

	return &c
}

// Handshake performs the connection handshake protocol.
func (c *Chat) Handshake(ctx context.Context, conn *websocket.Conn) error {
	err := conn.WriteMessage(websocket.TextMessage, []byte("HELLO"))
	if err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	msg, err := c.readMessage(ctx, conn)
	if err != nil {
		return fmt.Errorf("read message: %w", err)
	}

	var usr user
	if err := json.Unmarshal(msg, &usr); err != nil {
		return fmt.Errorf("unmarshal message: %w", err)
	}

	if err := c.addUser(usr, conn); err != nil {
		defer conn.Close()
		if err := conn.WriteMessage(websocket.TextMessage, []byte("Already Connected")); err != nil {
			return fmt.Errorf("write message: %w", err)
		}
		return fmt.Errorf("add user: %w", err)
	}

	v := fmt.Sprintf("WELCOME %s", usr.Name)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(v)); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	c.log.Info(ctx, "handshake complete", "usr", usr)

	return nil
}

// Listen waits for messages from users.
func (c *Chat) Listen(ctx context.Context, conn *websocket.Conn) {
	for {
		msg, err := c.readMessage(ctx, conn)
		if err != nil {
			c.log.Info(ctx, "listen-read", "err", err)
			return
		}

		var inMsg inMessage
		if err := json.Unmarshal(msg, &inMsg); err != nil {
			c.log.Info(ctx, "listen-unmarshal", "err", err)
			return
		}

		if err := c.sendMessage(inMsg); err != nil {
			c.log.Info(ctx, "listen-send", "err", err)
		}
	}
}

// =============================================================================

func (c *Chat) sendMessage(msg inMessage) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	from, exists := c.users[msg.FromID]
	if !exists {
		return ErrFromNotExists
	}

	to, exists := c.users[msg.ToID]
	if !exists {
		return ErrToNotExists
	}

	m := outMessage{
		From: user{
			ID:   from.id,
			Name: from.name,
		},
		To: user{
			ID:   to.id,
			Name: to.name,
		},
		Msg: msg.Msg,
	}

	if err := to.conn.WriteJSON(m); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	return nil
}

func (c *Chat) connections() map[uuid.UUID]connection {
	c.mu.RLock()
	defer c.mu.RUnlock()

	m := make(map[uuid.UUID]connection)
	for k, v := range c.users {
		m[k] = v
	}

	return m
}

func (c *Chat) ping() {
	ticker := time.NewTicker(10 * time.Second)

	go func() {
		for {
			<-ticker.C

			c.log.Info(context.Background(), "PING", "status", "started")

			for k, v := range c.connections() {
				c.log.Info(context.Background(), "PING", "name", v.name, "id", v.id)

				if err := v.conn.WriteMessage(websocket.PingMessage, []byte("ping")); err != nil {
					c.removeUser(k)
				}
			}

			c.log.Info(context.Background(), "PING", "status", "completed")
		}
	}()
}

func (c *Chat) addUser(usr user, conn *websocket.Conn) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.users[usr.ID]; exists {
		return fmt.Errorf("user exists")
	}

	c.log.Info(context.Background(), "add user", "name", usr.Name, "id", usr.ID)

	c.users[usr.ID] = connection{
		id:   usr.ID,
		name: usr.Name,
		conn: conn,
	}

	return nil
}

func (c *Chat) removeUser(userID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()

	v, exists := c.users[userID]
	if !exists {
		c.log.Info(context.Background(), "remove user", "userID", userID, "status", "does not exists")
		return
	}

	c.log.Info(context.Background(), "remove user", "name", v.name, "id", v.id)

	delete(c.users, userID)
	v.conn.Close()
}

func (c *Chat) readMessage(ctx context.Context, conn *websocket.Conn) ([]byte, error) {
	type response struct {
		msg []byte
		err error
	}

	ch := make(chan response, 1)

	go func() {
		c.log.Info(ctx, "chat", "status", "starting handshake read")
		defer c.log.Info(ctx, "chat", "status", "completed handshake read")

		_, msg, err := conn.ReadMessage()
		if err != nil {
			ch <- response{nil, err}
		}
		ch <- response{msg, nil}
	}()

	var resp response

	select {
	case <-ctx.Done():
		conn.Close()
		return nil, ctx.Err()

	case resp = <-ch:
		if resp.err != nil {
			return nil, fmt.Errorf("empty message")
		}
	}

	return resp.msg, nil
}
