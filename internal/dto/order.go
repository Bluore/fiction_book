package dto

import (
	"time"
)

type CreateOrderRequest struct {
	Price       float64 `json:"price" binding:"required"`
	RewardGold  int64   `json:"reward_gold" binding:"required"`
	RequestUUID string  `json:"request_uuid" binding:"required"`
}

type CreateOrderResponse struct {
	OrderID string  `json:"order_id"`
	PayURL  string  `json:"pay_url"`
	Price   float64 `json:"price"`
}

type GetOrderRequest struct {
	ID string `uri:"id" binding:"required"`
}
type GetOrderListRequest struct {
	PageListRequest
}

type GetOrderListResponse struct {
	Orders []*OrderResponse `json:"orders"`
	PageListResponse
}

type OrderResponse struct {
	OrderID       string    `json:"order_id"`
	UserID        string    `json:"user_id"`
	PayPrice      float64   `json:"pay_price"`
	OriginalPrice float64   `json:"original_price"`
	RewordGold    int64     `json:"reword_gold"`
	PastAt        time.Time `json:"past_at"`
	Status        int       `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}
