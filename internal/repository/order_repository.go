package repository

import "gorm.io/gorm"

type OrderRepository interface {
}

type orderRepository struct {
	db *gorm.DB
}

// NewOrderRepository 初始化订单仓库
func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{
		db: db,
	}
}
