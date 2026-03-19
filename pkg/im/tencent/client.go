package tencent

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"
)

const (
	baseURL = "https://console.tim.qq.com"
	version = "v4"
)

// Client 腾讯云 IM 客户端
type Client struct {
	AppID      int
	SecretKey  string
	Admin      string
	Expire     int
	userSig    string
	httpClient *http.Client
}

// NewClient 创建客户端
func NewClient(appID int, secretKey, admin string, expire int) (*Client, error) { // pragma: allowlist secret
	c := &Client{
		AppID:      appID,
		SecretKey:  secretKey, // pragma: allowlist secret
		Admin:      admin,
		Expire:     expire,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}

	sig, err := c.genUserSig(admin, expire)
	if err != nil {
		return nil, err
	}
	c.userSig = sig

	return c, nil
}

// genUserSig 生成 UserSig
func (c *Client) genUserSig(identifier string, expire int) (string, error) {
	currTime := time.Now().Unix()
	sigDoc := map[string]interface{}{
		"TLS.ver":        "2.0",
		"TLS.identifier": identifier,
		"TLS.sdkappid":   c.AppID,
		"TLS.expire":     expire,
		"TLS.time":       currTime,
	}

	data, err := json.Marshal(sigDoc)
	if err != nil {
		return "", fmt.Errorf("failed to marshal sig doc: %w", err)
	}
	h := hmac.New(sha256.New, []byte(c.SecretKey))
	h.Write(data)
	sig := base64.StdEncoding.EncodeToString(h.Sum(nil))

	sigDoc["TLS.sig"] = sig
	jsonData, err := json.Marshal(sigDoc)
	if err != nil {
		return "", fmt.Errorf("failed to marshal sig doc with signature: %w", err)
	}

	return base64.StdEncoding.EncodeToString(jsonData), nil
}

// GetUserSig 获取用户 UserSig（用于客户端登录）
func (c *Client) GetUserSig(userID string) (string, error) {
	return c.genUserSig(userID, c.Expire)
}

// request 发送请求
func (c *Client) request(path string, body interface{}) ([]byte, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/%s/%s?sdkappid=%d&identifier=%s&usersig=%s&random=%d&contenttype=json",
		baseURL, version, path, c.AppID, c.Admin, c.userSig, rand.Intn(4294967294))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		ErrorCode int    `json:"ErrorCode"`
		ErrorInfo string `json:"ErrorInfo"`
	}
	if err = json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	if result.ErrorCode != 0 {
		return nil, fmt.Errorf("tencent im error: %d - %s", result.ErrorCode, result.ErrorInfo)
	}

	return respBody, nil
}

// AccountImport 导入账号
func (c *Client) AccountImport(userID, nick, faceURL string) error {
	body := map[string]string{
		"Identifier": userID,
		"Nick":       nick,
		"FaceUrl":    faceURL,
	}
	_, err := c.request("im_open_login_svc/account_import", body)
	return err
}

// SendMessage 发送单聊消息
func (c *Client) SendMessage(fromUser, toUser, content string) (int64, string, error) {
	body := map[string]interface{}{
		"From_Account": fromUser,
		"To_Account":   toUser,
		"MsgRandom":    rand.Intn(4294967294),
		"MsgBody": []map[string]interface{}{
			{
				"MsgType": "TIMTextElem",
				"MsgContent": map[string]string{
					"Text": content,
				},
			},
		},
	}

	respBody, err := c.request("openim/sendmsg", body)
	if err != nil {
		return 0, "", err
	}

	var resp struct {
		MsgTime int64  `json:"MsgTime"`
		MsgKey  string `json:"MsgKey"`
	}
	if err = json.Unmarshal(respBody, &resp); err != nil {
		return 0, "", err
	}

	return resp.MsgTime, resp.MsgKey, nil
}

// QueryState 查询用户在线状态
func (c *Client) QueryState(userIDs []string) (map[string]string, error) {
	body := map[string]interface{}{
		"To_Account": userIDs,
	}

	respBody, err := c.request("openim/querystate", body)
	if err != nil {
		return nil, err
	}

	var resp struct {
		QueryResult []struct {
			ToAccount string `json:"To_Account"`
			State     string `json:"State"`
		} `json:"QueryResult"`
	}
	if err = json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, r := range resp.QueryResult {
		result[r.ToAccount] = r.State
	}
	return result, nil
}
