package im

// Config IM 配置
type Config struct {
	TencentIM TencentIMConfig `mapstructure:"tencent_im"`
	WebSocket WebSocketConfig `mapstructure:"websocket"`
}

// TencentIMConfig 腾讯云 IM 配置
type TencentIMConfig struct {
	AppID     int    `mapstructure:"app_id"`
	SecretKey string `mapstructure:"secret_key"`
	Admin     string `mapstructure:"admin"`
	Expire    int    `mapstructure:"expire"`
}

// WebSocketConfig WebSocket 配置
type WebSocketConfig struct {
	ReadBufferSize  int `mapstructure:"read_buffer_size"`
	WriteBufferSize int `mapstructure:"write_buffer_size"`
	HeartbeatSec    int `mapstructure:"heartbeat_sec"`
}
