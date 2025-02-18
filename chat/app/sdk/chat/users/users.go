package users

import (
	"context"
	"sync"
	"time"

	"github.com/ardanlabs/usdl/chat/app/sdk/chat"
	"github.com/ardanlabs/usdl/chat/foundation/logger"
	"github.com/google/uuid"
)

// Users provides user storage management.
type Users struct {
	log     *logger.Logger
	users   map[uuid.UUID]chat.User
	muUsers sync.RWMutex
}

// New creates a new user storage.
func New(log *logger.Logger) *Users {
	u := Users{
		log:   log,
		users: make(map[uuid.UUID]chat.User),
	}

	return &u
}

// Add adds a new user to the storage.
func (u *Users) Add(ctx context.Context, usr chat.User) error {
	u.muUsers.Lock()
	defer u.muUsers.Unlock()

	if _, exists := u.users[usr.ID]; exists {
		return chat.ErrExists
	}

	u.users[usr.ID] = usr

	u.log.Info(ctx, "chat-adduser", "name", usr.Name, "id", usr.ID)

	return nil
}

// UpdateLastPong updates a user value's pong date/time.
func (u *Users) UpdateLastPong(ctx context.Context, userID uuid.UUID) error {
	u.muUsers.Lock()
	defer u.muUsers.Unlock()

	usr, exists := u.users[userID]
	if !exists {
		return chat.ErrNotExists
	}

	usr.LastPong = time.Now()
	u.users[usr.ID] = usr

	u.log.Info(ctx, "chat-upduser", "name", usr.Name, "id", usr.ID, "lastPong", usr.LastPong)

	return nil
}

// Remove removes a user from the storage.
func (u *Users) Remove(ctx context.Context, userID uuid.UUID) {
	u.muUsers.Lock()
	defer u.muUsers.Unlock()

	usr, exists := u.users[userID]
	if !exists {
		u.log.Info(ctx, "chat-removeuser", "userID", userID, "status", "does not exists")
		return
	}

	delete(u.users, userID)

	u.log.Info(ctx, "chat-removeuser", "name", usr.Name, "id", usr.ID)
}

// Connections returns all the know users with their connections. A connection
// that is not valid shouldn't be used.
func (u *Users) Connections() map[uuid.UUID]chat.Connection {
	u.muUsers.RLock()
	defer u.muUsers.RUnlock()

	m := make(map[uuid.UUID]chat.Connection)
	for id, usr := range u.users {
		m[id] = chat.Connection{
			Conn:     usr.Conn,
			LastPong: usr.LastPong,
		}
	}

	return m
}

// Retrieve retrieves a user from the storage.
func (u *Users) Retrieve(ctx context.Context, userID uuid.UUID) (chat.User, error) {
	u.muUsers.RLock()
	defer u.muUsers.RUnlock()

	usr, exists := u.users[userID]
	if !exists {
		return chat.User{}, chat.ErrNotExists
	}

	return usr, nil
}
