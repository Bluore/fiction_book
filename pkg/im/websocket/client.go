package websocket

import (
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

// Client WebSocket 客户端
type Client struct {
	ID            string
	UserID        string
	Hub           *Hub
	Conn          *websocket.Conn
	Send          chan []byte
	HeartbeatTime time.Time
}

// NewClient 创建客户端
func NewClient(id, userID string, hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		ID:            id,
		UserID:        userID,
		Hub:           hub,
		Conn:          conn,
		Send:          make(chan []byte, 256),
		HeartbeatTime: time.Now(),
	}
}

// ReadPump 读取消息
func (c *Client) ReadPump(handler func(client *Client, message []byte)) {
	defer func() {
		c.Hub.Unregister(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	//nolint:errcheck // deadline errors are non-critical for websocket
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		//nolint:errcheck // deadline errors are non-critical for websocket
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		c.HeartbeatTime = time.Now()
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
		if handler != nil {
			handler(c, message)
		}
	}
}

// WritePump 发送消息
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			//nolint:errcheck // deadline errors are non-critical for websocket
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				//nolint:errcheck // close message errors are non-critical
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			//nolint:errcheck // deadline errors are non-critical for websocket
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SendMessage 发送消息
func (c *Client) SendMessage(message []byte) {
	select {
	case c.Send <- message:
	default:
	}
}
