package repository

import (
	"context"
	"errors"
	"fiction-book/internal/model"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrIntExpected  = errors.New("int expected")
	ErrGoldNegative = errors.New("gold value will be negative")

	maxInt64 = int64(^uint64(0) >> 1)
	minInt64 = -maxInt64
)

type UserAssetRepository interface {
	GetByID(ctx context.Context, db *gorm.DB, id string) (*model.UserAsset, error)
	// GetByIDForUpdate 在事务中对用户资产行加锁读取，用于金币等金额的校验与变更
	GetByIDForUpdate(ctx context.Context, db *gorm.DB, id string) (*model.UserAsset, error)
	UpdateByID(ctx context.Context, db *gorm.DB, id string, data *model.UserAsset) error
	AddGoldByID(ctx context.Context, db *gorm.DB, id string, gold int64) (*model.UserAsset, error)
	SubGoldByID(ctx context.Context, db *gorm.DB, id string, gold int64) (*model.UserAsset, error)
	GetGoldByID(ctx context.Context, db *gorm.DB, id string) (int64, error)
	CreateUserAsset(ctx context.Context, db *gorm.DB, id string) error
	DeleteByID(ctx context.Context, db *gorm.DB, id string) error
}

type userAssetRepository struct {
	db *gorm.DB
}

func NewUserAssetRepository(db *gorm.DB) UserAssetRepository {
	return &userAssetRepository{db: db}
}

func (u userAssetRepository) GetByID(ctx context.Context, db *gorm.DB, id string) (*model.UserAsset, error) {
	return u.firstByID(ctx, db, id, false)
}

func (u userAssetRepository) GetByIDForUpdate(ctx context.Context, db *gorm.DB, id string) (*model.UserAsset, error) {
	return u.firstByID(ctx, db, id, true)
}

func (u userAssetRepository) firstByID(ctx context.Context, db *gorm.DB, id string, forUpdate bool) (*model.UserAsset, error) {
	var userAsset model.UserAsset
	query := db.WithContext(ctx)
	if forUpdate {
		// 行锁需在事务内使用，SQLite 会忽略该子句
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := query.First(&userAsset, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &userAsset, nil
}

func (u userAssetRepository) UpdateByID(ctx context.Context, db *gorm.DB, id string, data *model.UserAsset) error {
	return db.WithContext(ctx).Where("id = ?", id).Save(data).Error
}

func (u userAssetRepository) AddGoldByID(ctx context.Context, db *gorm.DB, id string, gold int64) (*model.UserAsset, error) {
	// 不允许增加负资产
	if gold < 0 {
		return nil, errors.New("forbidden gold value negative")
	}

	// 加行锁读取，避免并发发放金币丢失更新
	userAsset, err := u.GetByIDForUpdate(ctx, db, id)
	if err != nil {
		return nil, err
	}
	if userAsset == nil {
		return nil, nil
	}

	userAsset.Gold, err = safeAddInt64(userAsset.Gold, gold)
	if err != nil {
		return nil, err
	}
	// 资产为负数
	if userAsset.Gold < 0 {
		return nil, ErrGoldNegative
	}

	err = db.WithContext(ctx).Save(userAsset).Error
	if err != nil {
		return nil, err
	}
	return userAsset, nil
}

func (u userAssetRepository) SubGoldByID(ctx context.Context, db *gorm.DB, id string, gold int64) (*model.UserAsset, error) {
	// 不允许减少负数
	if gold < 0 {
		return nil, errors.New("forbidden gold value negative")
	}

	// 加行锁读取，避免并发扣减金币丢失更新
	userAsset, err := u.GetByIDForUpdate(ctx, db, id)
	if err != nil {
		return nil, err
	}
	if userAsset == nil {
		return nil, nil
	}

	addGold := -gold
	userAsset.Gold, err = safeAddInt64(userAsset.Gold, addGold)
	if err != nil {
		return nil, err
	}
	// 资产为负数
	if userAsset.Gold < 0 {
		return nil, ErrGoldNegative
	}

	err = db.WithContext(ctx).Save(userAsset).Error
	if err != nil {
		return nil, err
	}
	return userAsset, nil
}

func (u userAssetRepository) GetGoldByID(ctx context.Context, db *gorm.DB, id string) (int64, error) {
	userAsset, err := u.GetByID(ctx, db, id)
	if userAsset == nil || err != nil {
		return 0, err
	}
	return userAsset.Gold, err
}

func (u userAssetRepository) CreateUserAsset(ctx context.Context, db *gorm.DB, id string) error {
	exist, err := u.GetByID(ctx, db, id)
	if err != nil {
		return err
	}
	if exist != nil {
		return errors.New("user asset exist")
	}

	nowTime := time.Now()

	userAsset := &model.UserAsset{
		ID:          id,
		Gold:        0,
		VIPLevel:    "vip_0",
		VIPActiveAt: &nowTime,
	}
	err = db.WithContext(ctx).Save(userAsset).Error
	if err != nil {
		return err
	}

	exist, err = u.GetByID(ctx, db, id)
	if err != nil {
		return err
	}
	if exist == nil {
		return errors.New("user_asset create failed")
	}
	return nil
}

// DeleteByID 软删除用户资产记录（DeletedAt 置位）
func (u userAssetRepository) DeleteByID(ctx context.Context, db *gorm.DB, id string) error {
	return db.WithContext(ctx).Where("id = ?", id).Delete(&model.UserAsset{}).Error
}

func safeAddInt64(a int64, b int64) (int64, error) {
	if a >= 0 && b <= 0 {
		return a + b, nil
	}
	if a <= 0 && b >= 0 {
		return a + b, nil
	}
	if a > 0 && b > 0 {
		if a > maxInt64-b {
			return 0, ErrIntExpected
		}
		return a + b, nil
	}
	if a < 0 && b < 0 {
		if a < minInt64-b {
			return 0, ErrIntExpected
		}
		return a + b, nil
	}
	return 0, ErrIntExpected
}
