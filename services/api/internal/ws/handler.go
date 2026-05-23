package ws

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"

	"richtalk/api/internal/httpx"
	"richtalk/api/internal/service"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Origin check is delegated to Nginx in production.
	// In dev, Vite runs on a different port so we allow all origins here.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ServeWS upgrades the HTTP connection to WebSocket after validating the JWT
// passed as the `token` query parameter.
//
// JWT is validated BEFORE the upgrade so we can still return a proper 401 HTTP
// response if the token is invalid. After upgrade it is too late to send HTTP
// status codes.
func ServeWS(hub *Hub, jwtSvc *service.JWTService, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			httpx.Error(w, http.StatusUnauthorized, "unauthorized", "Токен обязателен")
			return
		}

		claims, err := jwtSvc.Validate(token)
		if err != nil {
			httpx.Error(w, http.StatusUnauthorized, "invalid_token", "Недействительный или истёкший токен")
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			// upgrader writes the error response itself
			log.Debug("ws upgrade failed", "error", err)
			return
		}

		client := &Client{
			hub:    hub,
			conn:   conn,
			userID: claims.Subject,
			send:   make(chan []byte, 256),
			log:    log,
		}

		hub.register <- client

		go client.writePump()
		go client.readPump()
	}
}
