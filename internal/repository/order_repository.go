package repository

import (
	"context"
	"fiction-book/internal/model"

	"gorm.io/gorm"
)

type OrderRepository interface {
	GetOrderByID(ctx context.Context, db *gorm.DB, id string) (*model.Order, error)
	CreateOrder(ctx context.Context, db *gorm.DB, in *model.Order) error
	DeleteOrder(ctx context.Context, db *gorm.DB, orderId string) error
	GetOrderListByUserID(ctx context.Context, db *gorm.DB, userId string, page int, size int) ([]*model.Order, error)
	Count(ctx context.Context, db *gorm.DB, userId string) (int, error)
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

func (r orderRepository) GetOrderByID(ctx context.Context, db *gorm.DB, id string) (*model.Order, error) {
	var order model.Order
	err := db.WithContext(ctx).First(&order, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

func (r orderRepository) CreateOrder(ctx context.Context, db *gorm.DB, in *model.Order) error {
	err := db.WithContext(ctx).Save(in).Error
	if err != nil {
		return err
	}
	return nil
}

func (r orderRepository) DeleteOrder(ctx context.Context, db *gorm.DB, orderId string) error {
	return db.WithContext(ctx).Model(&model.Order{}).Delete("id = ?", orderId).Error
}

func (r orderRepository) GetOrderListByUserID(ctx context.Context, db *gorm.DB, userId string, page int, size int) ([]*model.Order, error) {
	var orders []*model.Order
	err := db.WithContext(ctx).Where("user_id = ?", userId).Offset((page - 1) * size).Limit(size).Find(&orders).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return make([]*model.Order, 0), nil
		}
		return nil, err
	}
	return orders, nil
}

func (r orderRepository) Count(ctx context.Context, db *gorm.DB, userId string) (int, error) {
	var count int64
	err := db.WithContext(ctx).Model(&model.Order{}).Where("user_id = ?", userId).Count(&count).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, err
	}
	return int(count), nil
}
