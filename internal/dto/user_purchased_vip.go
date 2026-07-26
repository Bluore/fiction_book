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
