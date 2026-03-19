package websocket

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(_ *http.Request) bool {
		return true // 允许所有来源，生产环境需要限制
	},
}

// Handler WebSocket Handler
type Handler struct {
	Hub            *Hub
	MessageHandler func(client *Client, message []byte)
}

// NewHandler 创建 Handler
func NewHandler(hub *Hub, msgHandler func(*Client, []byte)) *Handler {
	return &Handler{
		Hub:            hub,
		MessageHandler: msgHandler,
	}
}

// ServeWS 处理 WebSocket 连接
func (h *Handler) ServeWS(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id required"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := NewClient(uuid.New().String(), userID, h.Hub, conn)
	h.Hub.Register(client)

	go client.WritePump()
	go client.ReadPump(h.MessageHandler)
}
