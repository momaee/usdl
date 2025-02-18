package chatapp

import (
	"net/http"

	"github.com/ardanlabs/usdl/chat/app/sdk/chat"
	"github.com/ardanlabs/usdl/chat/app/sdk/chat/users"
	"github.com/ardanlabs/usdl/chat/foundation/logger"
	"github.com/ardanlabs/usdl/chat/foundation/web"
)

// Routes adds specific routes for this group.
func Routes(app *web.App, log *logger.Logger) {
	chat := chat.New(log, users.New(log))

	api := newApp(log, chat)

	app.HandlerFunc(http.MethodGet, "", "/connect", api.connect)
}
