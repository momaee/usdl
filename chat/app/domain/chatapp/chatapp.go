package chatapp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ardanlabs/usdl/chat/app/sdk/errs"
	"github.com/ardanlabs/usdl/chat/foundation/logger"
	"github.com/ardanlabs/usdl/chat/foundation/web"
	"github.com/gorilla/websocket"
)

type app struct {
	log *logger.Logger
	WS  websocket.Upgrader
}

func newApp(log *logger.Logger) *app {
	return &app{
		log: log,
	}
}

func (a *app) connect(ctx context.Context, r *http.Request) web.Encoder {
	// We have handshake working!!
	// We want test handshake errors including server times out
	// Router Package, map socket to user, and then send a message to the user
	// Finish this code.

	c, err := a.WS.Upgrade(web.GetWriter(ctx), r, nil)
	if err != nil {
		return errs.Newf(errs.FailedPrecondition, "unable to upgrade to websocket")
	}
	defer c.Close()

	usr, err := a.handshake(c)
	if err != nil {
		return errs.Newf(errs.FailedPrecondition, "handshake failed: %s", err)
	}

	a.log.Info(ctx, "handshake complete", "usr", usr)

	// var wg sync.WaitGroup
	// wg.Add(3)

	// ticker := time.NewTicker(time.Second)

	// go func() {
	// 	wg.Done()

	// 	select {
	// 	case <-ticker.C:
	// 		if err := c.WriteMessage(websocket.PingMessage, []byte("ping")); err != nil {
	// 			return
	// 		}
	// 	}
	// }()

	// go func() {
	// 	wg.Done()

	// 	for {
	// 		_, msg, err := c.ReadMessage()
	// 		if err != nil {
	// 			return
	// 		}
	// 	}
	// }()

	// go func() {
	// 	wg.Done()

	// 	for {
	// 		msg, wd := <-ch

	// 		// If the channel is closed, release the websocket.
	// 		if !wd {
	// 			return
	// 		}

	// 		if err := c.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
	// 			return
	// 		}
	// 	}
	// }()

	// wg.Wait()

	return web.NewNoResponse()
}

func (a *app) handshake(c *websocket.Conn) (user, error) {
	if err := c.WriteMessage(websocket.TextMessage, []byte("HELLO")); err != nil {
		return user{}, fmt.Errorf("write message: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	msg, err := a.readMessage(ctx, c)
	if err != nil {
		return user{}, fmt.Errorf("read message: %w", err)
	}

	var usr user
	if err := json.Unmarshal(msg, &usr); err != nil {
		return user{}, fmt.Errorf("unmarshal message: %w", err)
	}

	v := fmt.Sprintf("WELCOME %s", usr.Name)
	if err := c.WriteMessage(websocket.TextMessage, []byte(v)); err != nil {
		return user{}, fmt.Errorf("write message: %w", err)
	}

	return usr, nil
}

func (a *app) readMessage(ctx context.Context, c *websocket.Conn) ([]byte, error) {
	type response struct {
		msg []byte
		err error
	}

	ch := make(chan response, 1)

	go func() {
		a.log.Info(ctx, "starting handshake read")
		defer a.log.Info(ctx, "completed handshake read")

		_, msg, err := c.ReadMessage()
		if err != nil {
			ch <- response{nil, err}
		}
		ch <- response{msg, nil}
	}()

	var resp response

	select {
	case <-ctx.Done():
		c.Close()
		return nil, ctx.Err()

	case resp = <-ch:
		if resp.err != nil {
			return nil, fmt.Errorf("empty message")
		}
	}

	return resp.msg, nil
}
