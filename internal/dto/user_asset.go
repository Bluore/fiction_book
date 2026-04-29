package dto

import "time"

type GetUserAssetRequest struct {
}

type UserAssetResponse struct {
	ID          string     `json:"id"`
	Gold        int64      `json:"gold"`
	VIPMark     string     `json:"vip_mark"`
	VIPLevel    int        `json:"vip_level"`
	VIPActiveAt *time.Time `json:"vip_active_at"`
}
