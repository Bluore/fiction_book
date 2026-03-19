package websocket

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	hub := NewHub()
	client := NewClient("id1", "user1", hub, nil)

	assert.Equal(t, "id1", client.ID)
	assert.Equal(t, "user1", client.UserID)
	assert.Equal(t, hub, client.Hub)
	assert.NotNil(t, client.Send)
	assert.False(t, client.HeartbeatTime.IsZero())
}

func TestClientSendMessage(t *testing.T) {
	hub := NewHub()
	client := NewClient("id1", "user1", hub, nil)

	// 发送消息到 channel
	client.SendMessage([]byte("test message"))

	select {
	case msg := <-client.Send:
		assert.Equal(t, "test message", string(msg))
	case <-time.After(100 * time.Millisecond):
		t.Fatal("message not received")
	}
}

func TestClientSendMessageFull(t *testing.T) {
	hub := NewHub()
	client := &Client{
		ID:     "id1",
		UserID: "user1",
		Hub:    hub,
		Send:   make(chan []byte, 1), // 小容量
	}

	// 第一条消息应该成功
	client.SendMessage([]byte("msg1"))

	// 第二条消息应该被丢弃（channel 已满）
	client.SendMessage([]byte("msg2"))

	// 只应该收到第一条
	msg := <-client.Send
	assert.Equal(t, "msg1", string(msg))
}
