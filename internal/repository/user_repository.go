package repository

import (
	"context"

	"gorm.io/gorm"

	"fiction-book/internal/model"
)

// UserRepository 用户仓储接口
type UserRepository interface {
	Create(ctx context.Context, db *gorm.DB, user *model.User) error
	GetByID(ctx context.Context, db *gorm.DB, id string) (*model.User, error)
	GetByUsername(ctx context.Context, db *gorm.DB, username string) (*model.User, error)
	GetByEmail(ctx context.Context, db *gorm.DB, email string) (*model.User, error)
	Update(ctx context.Context, db *gorm.DB, user *model.User) error
	Delete(ctx context.Context, db *gorm.DB, id string) error
	List(ctx context.Context, db *gorm.DB, offset, limit int) ([]*model.User, error)
}

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户仓储实例
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, db *gorm.DB, user *model.User) error {
	return db.WithContext(ctx).Create(user).Error
}

func (r *userRepository) GetByID(ctx context.Context, db *gorm.DB, id string) (*model.User, error) {
	var user model.User
	err := db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByUsername(ctx context.Context, db *gorm.DB, username string) (*model.User, error) {
	var user model.User
	err := db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, db *gorm.DB, email string) (*model.User, error) {
	var user model.User
	err := db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) Update(ctx context.Context, db *gorm.DB, user *model.User) error {
	return db.WithContext(ctx).Save(user).Error
}

func (r *userRepository) Delete(ctx context.Context, db *gorm.DB, id string) error {
	return db.WithContext(ctx).Delete(&model.User{}, "id = ?", id).Error
}

func (r *userRepository) List(ctx context.Context, db *gorm.DB, offset, limit int) ([]*model.User, error) {
	var users []*model.User
	err := db.WithContext(ctx).Offset(offset).Limit(limit).Find(&users).Error
	return users, err
}
