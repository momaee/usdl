package chatapp

import (
	"net/http"

	"github.com/ardanlabs/usdl/chat/app/sdk/chat"
	"github.com/ardanlabs/usdl/chat/foundation/logger"
	"github.com/ardanlabs/usdl/chat/foundation/web"
)

// Routes adds specific routes for this group.
func Routes(app *web.App, log *logger.Logger, chat *chat.Chat) {
	api := newApp(log, chat)

	app.HandlerFunc(http.MethodGet, "", "/connect", api.connect)
}
