package chatapp

import (
	"net/http"

	"github.com/ardanlabs/usdl/chat/foundation/logger"
	"github.com/ardanlabs/usdl/chat/foundation/web"
)

// Routes adds specific routes for this group.
func Routes(app *web.App, log *logger.Logger) {
	api := newApp(log)

	app.HandlerFunc(http.MethodGet, "", "/connect", api.connect)
}
