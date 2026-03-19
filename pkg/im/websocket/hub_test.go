package websocket

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()
	assert.NotNil(t, hub)
	assert.NotNil(t, hub.clients)
	assert.NotNil(t, hub.broadcast)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
}

func TestHubRegisterUnregister(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// 创建模拟客户端
	client := &Client{
		ID:     "test-client-1",
		UserID: "user1",
		Send:   make(chan []byte, 256),
	}

	// 注册
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	assert.True(t, hub.IsOnline("test-client-1"))

	// 注销
	hub.Unregister(client)
	time.Sleep(10 * time.Millisecond)

	assert.False(t, hub.IsOnline("test-client-1"))
}

func TestHubGetOnlineUsers(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client1 := &Client{ID: "client1", UserID: "user1", Send: make(chan []byte, 256)}
	client2 := &Client{ID: "client2", UserID: "user2", Send: make(chan []byte, 256)}

	hub.Register(client1)
	hub.Register(client2)
	time.Sleep(10 * time.Millisecond)

	users := hub.GetOnlineUsers()
	assert.Len(t, users, 2)
	assert.Contains(t, users, "client1")
	assert.Contains(t, users, "client2")
}

func TestHubSendToUser(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{ID: "client1", UserID: "user1", Send: make(chan []byte, 256)}
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	// 发送消息给存在的用户
	ok := hub.SendToUser("client1", []byte("hello"))
	assert.True(t, ok)

	// 验证消息
	select {
	case msg := <-client.Send:
		assert.Equal(t, "hello", string(msg))
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected message not received")
	}

	// 发送消息给不存在的用户
	ok = hub.SendToUser("nonexistent", []byte("hello"))
	assert.False(t, ok)
}

func TestHubIsOnline(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{ID: "online-client", UserID: "user1", Send: make(chan []byte, 256)}

	// 未注册时
	assert.False(t, hub.IsOnline("online-client"))

	// 注册后
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)
	assert.True(t, hub.IsOnline("online-client"))
}
