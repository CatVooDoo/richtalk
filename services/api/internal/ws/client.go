package ws

import (
	"log/slog"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10 // slightly less than pongWait
	maxMsgSize = 4096
)

// Client represents a single WebSocket connection for one authenticated user.
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	userID string
	send   chan []byte // buffered outbound messages
	log    *slog.Logger
}

// readPump reads incoming messages from the browser and forwards them to the hub.
// Runs in its own goroutine. Closes the connection on any error.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMsgSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.log.Debug("ws read error", "user_id", c.userID, "error", err)
			}
			break
		}
		c.hub.incoming <- incomingMessage{client: c, data: msg}
	}
}

// writePump drains the send channel and writes messages to the WebSocket.
// Also sends periodic pings to detect dead connections.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				c.log.Debug("ws write error", "user_id", c.userID, "error", err)
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
