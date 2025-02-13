// Package chat provides supports for chat activity.
package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ardanlabs/usdl/chat/app/sdk/errs"
	"github.com/ardanlabs/usdl/chat/foundation/logger"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Set of error variables.
var (
	ErrFromNotExists = fmt.Errorf("from user doesn't exists")
	ErrToNotExists   = fmt.Errorf("to user doesn't exists")
)

// Chat represents a chat support.
type Chat struct {
	log   *logger.Logger
	users map[uuid.UUID]User
	mu    sync.RWMutex
}

// New creates a new chat support.
func New(log *logger.Logger) *Chat {
	c := Chat{
		log:   log,
		users: make(map[uuid.UUID]User),
	}

	c.ping()

	return &c
}

// Handshake performs the connection handshake protocol.
func (c *Chat) Handshake(ctx context.Context, w http.ResponseWriter, r *http.Request) (User, error) {
	var ws websocket.Upgrader
	conn, err := ws.Upgrade(w, r, nil)
	if err != nil {
		return User{}, errs.Newf(errs.FailedPrecondition, "unable to upgrade to websocket")
	}

	if err := conn.WriteMessage(websocket.TextMessage, []byte("HELLO")); err != nil {
		return User{}, fmt.Errorf("write message: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	usr := User{
		Conn: conn,
	}

	msg, err := c.readMessage(ctx, usr)
	if err != nil {
		return User{}, fmt.Errorf("read message: %w", err)
	}

	if err := json.Unmarshal(msg, &usr); err != nil {
		return User{}, fmt.Errorf("unmarshal message: %w", err)
	}

	if err := c.addUser(ctx, usr); err != nil {
		defer conn.Close()
		if err := conn.WriteMessage(websocket.TextMessage, []byte("Already Connected")); err != nil {
			return User{}, fmt.Errorf("write message: %w", err)
		}
		return User{}, fmt.Errorf("add user: %w", err)
	}

	v := fmt.Sprintf("WELCOME %s", usr.Name)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(v)); err != nil {
		return User{}, fmt.Errorf("write message: %w", err)
	}

	c.log.Info(ctx, "chat-handshake", "status", "complete", "usr", usr)

	return usr, nil
}

// Listen waits for messages from users.
func (c *Chat) Listen(ctx context.Context, usr User) {
	for {
		msg, err := c.readMessage(ctx, usr)
		if err != nil {
			if c.isCriticalError(ctx, err) {
				return
			}
			continue
		}

		var inMsg inMessage
		if err := json.Unmarshal(msg, &inMsg); err != nil {
			c.log.Info(ctx, "chat-listen-unmarshal", "err", err)
			continue
		}

		if err := c.sendMessage(inMsg); err != nil {
			c.log.Info(ctx, "chat-listen-send", "err", err)
		}
	}
}

// =============================================================================

func (c *Chat) isCriticalError(ctx context.Context, err error) bool {
	switch err.(type) {
	case *websocket.CloseError:
		c.log.Info(ctx, "chat-isCriticalError", "status", "client disconnected")
		return true

	default:
		if errors.Is(err, context.Canceled) {
			c.log.Info(ctx, "chat-isCriticalError", "status", "client canceled")
			return true
		}

		c.log.Info(ctx, "chat-isCriticalError", "err", err)
		return false
	}
}

func (c *Chat) readMessage(ctx context.Context, usr User) ([]byte, error) {
	type response struct {
		msg []byte
		err error
	}

	ch := make(chan response, 1)

	go func() {
		c.log.Info(ctx, "chat-readMessage", "status", "started")
		defer c.log.Info(ctx, "chat-readMessage", "status", "completed")

		_, msg, err := usr.Conn.ReadMessage()
		if err != nil {
			ch <- response{nil, err}
		}
		ch <- response{msg, nil}
	}()

	var resp response

	select {
	case <-ctx.Done():
		c.removeUser(ctx, usr.ID)
		return nil, ctx.Err()

	case resp = <-ch:
		if resp.err != nil {
			c.removeUser(ctx, usr.ID)
			return nil, resp.err
		}
	}

	return resp.msg, nil
}

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
		From: User{
			ID:   from.ID,
			Name: from.Name,
		},
		To: User{
			ID:   to.ID,
			Name: to.Name,
		},
		Msg: msg.Msg,
	}

	if err := to.Conn.WriteJSON(m); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	return nil
}

func (c *Chat) connections() map[uuid.UUID]*websocket.Conn {
	c.mu.RLock()
	defer c.mu.RUnlock()

	m := make(map[uuid.UUID]*websocket.Conn)
	for id, usr := range c.users {
		m[id] = usr.Conn
	}

	return m
}

func (c *Chat) ping() {
	ticker := time.NewTicker(10 * time.Second)

	go func() {
		ctx := context.Background()

		for {
			<-ticker.C

			// Check to see if we got PONGS for the last set of PINGS.

			for k, conn := range c.connections() {
				if err := conn.WriteMessage(websocket.PingMessage, []byte("ping")); err != nil {
					c.log.Info(ctx, "chat-PING", "status", "failed", "id", k, "err", err)
				}
			}
		}
	}()
}

func (c *Chat) addUser(ctx context.Context, usr User) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.users[usr.ID]; exists {
		return fmt.Errorf("user exists")
	}

	c.log.Info(ctx, "chat-adduser", "name", usr.Name, "id", usr.ID)

	c.users[usr.ID] = usr

	return nil
}

func (c *Chat) removeUser(ctx context.Context, userID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()

	usr, exists := c.users[userID]
	if !exists {
		c.log.Info(ctx, "chat-removeuser", "userID", userID, "status", "does not exists")
		return
	}

	c.log.Info(ctx, "chat-removeuser", "name", usr.Name, "id", usr.ID)

	delete(c.users, userID)
	usr.Conn.Close()
}
