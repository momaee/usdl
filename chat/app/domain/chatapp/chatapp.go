package chatapp

import (
	"context"
	"net/http"

	"github.com/ardanlabs/usdl/chat/foundation/web"
)

type app struct {
}

func newApp() *app {
	return &app{}
}

func (a *app) test(_ context.Context, _ *http.Request) web.Encoder {
	// Web socket implemented here
	// Just perform basic echo
	// Make sure we are handling connection drops/issues (context)
	// How we will map a connection to a user

	return status{
		Status: "ok",
	}
}
