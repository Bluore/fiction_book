package tencent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client, err := NewClient(1400000000, "test_secret_key", "admin", 86400)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, 1400000000, client.AppID)
	assert.Equal(t, "admin", client.Admin)
	assert.Equal(t, 86400, client.Expire)
	assert.NotEmpty(t, client.userSig)
}

func TestGetUserSig(t *testing.T) {
	client, err := NewClient(1400000000, "test_secret_key", "admin", 86400)
	require.NoError(t, err)

	userSig, err := client.GetUserSig("test_user")
	assert.NoError(t, err)
	assert.NotEmpty(t, userSig)

	// 不同用户应该生成不同的 sig
	userSig2, err := client.GetUserSig("another_user")
	assert.NoError(t, err)
	assert.NotEqual(t, userSig, userSig2)
}

func TestGenUserSig(t *testing.T) {
	client, err := NewClient(1400000000, "test_secret_key", "admin", 86400)
	require.NoError(t, err)

	sig1, err := client.genUserSig("user1", 3600)
	assert.NoError(t, err)
	assert.NotEmpty(t, sig1)

	sig2, err := client.genUserSig("user2", 3600)
	assert.NoError(t, err)
	assert.NotEmpty(t, sig2)

	// 相同用户多次生成的 sig 应该不同（因为包含时间戳）
	sig3, err := client.genUserSig("user1", 3600)
	assert.NoError(t, err)
	// 时间戳相同时可能相同，这里只验证生成成功
	assert.NotEmpty(t, sig3)
}
