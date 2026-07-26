package dto

import "time"

type BuyVIPRequest struct {
	VIPLevel string `json:"vip_level" binding:"required,oneof=vip_1 vip_2"`
	Duration string `json:"duration" binding:"required,oneof=month year"`
	Price    int64  `json:"price" binding:"required"`
}

type BuyVIPResponse struct {
	UserID      string     `json:"user_id"`
	Gold        int64      `json:"gold"`
	VIPMark     string     `json:"vip_mark"`
	VIPLevel    int        `json:"vip_level"`
	VIPActiveAt *time.Time `json:"vip_active_at"`
	Price       int64      `json:"price"`
	Duration    string     `json:"duration"`
}

type UpgradeVIPRequest struct {
	OriginalVIPLevel string `json:"original_vip_level" binding:"required,oneof=vip_1"`
	TargetVIPLevel   string `json:"target_vip_level" binding:"required,oneof=vip_2"`
	Price            int64  `json:"price" binding:"required"`
}

type UpgradeVIPResponse struct {
	UserID           string     `json:"user_id"`
	Gold             int64      `json:"gold"`
	OriginalVIPLevel string     `json:"original_vip_level"`
	TargetVIPLevel   string     `json:"target_vip_level"`
	VIPMark          string     `json:"vip_mark"`
	VIPLevel         int        `json:"vip_level"`
	VIPActiveAt      *time.Time `json:"vip_active_at"`
	Price            int64      `json:"price"`
}

type ShopListOfVIPResponse struct {
	Shop []ShopVIPItem `json:"shop"`
}

type ShopVIPItem struct {
	Type     string     `json:"type"`
	VIPLevel string     `json:"vip_level"`
	Price    int64      `json:"price"`
	Duration string     `json:"duration,omitempty"`
	ActiveAt *time.Time `json:"active_at"`
}
