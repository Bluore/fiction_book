package service

import (
	"context"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
	"time"

	"gorm.io/gorm"
)

const (
	vipLevel0 = "vip_0"
	vipLevel1 = "vip_1"
	vipLevel2 = "vip_2"

	vipDurationMonth = "month"
	vipDurationYear  = "year"

	vipShopTypeBuy     = "buy"
	vipShopTypeRenew   = "renew"
	vipShopTypeUpgrade = "upgrade"
)

type BuyVIP interface {
	BuyVIP(ctx context.Context, userID string, req *dto.BuyVIPRequest) (*dto.BuyVIPResponse, error)
	UpgradeVIP(ctx context.Context, userID string, req *dto.UpgradeVIPRequest) (*dto.UpgradeVIPResponse, error)
	ShopListOfVIP(ctx context.Context, userID string) (*dto.ShopListOfVIPResponse, error)
}

type buyVIPService struct {
	userRepo      repository.UserRepository
	userAssetRepo repository.UserAssetRepository
	db            *gorm.DB
	cfg           *config.Config
}

func NewBuyVIPService(
	userRepo repository.UserRepository,
	userAssetRepo repository.UserAssetRepository,
	db *gorm.DB,
	cfg *config.Config,
) BuyVIP {
	return &buyVIPService{
		userRepo:      userRepo,
		userAssetRepo: userAssetRepo,
		db:            db,
		cfg:           cfg,
	}
}

func (s *buyVIPService) BuyVIP(ctx context.Context, userID string, req *dto.BuyVIPRequest) (*dto.BuyVIPResponse, error) {
	if req == nil || req.Price <= 0 {
		return nil, ErrParam
	}

	var purchasedVIP *model.UserAsset
	var price int64
	var activeAt time.Time
	err := s.db.Transaction(func(tx *gorm.DB) error {
		user, err := s.userRepo.GetByID(ctx, tx, userID)
		if err != nil {
			return err
		}
		if user == nil {
			return ErrUserNotFound
		}

		userAsset, err := s.userAssetRepo.GetByID(ctx, tx, userID)
		if err != nil {
			return err
		}
		if userAsset == nil {
			return ErrUserNotFound
		}

		currentVIP := effectiveVIPLevel(userAsset, time.Now())
		price, err = s.vipPrice(currentVIP, req.VIPLevel, req.Duration)
		if err != nil {
			return err
		}
		if price <= 0 || req.Price != price {
			return ErrParam
		}

		if userAsset.Gold < price {
			return ErrSufficientFunds
		}

		userAssetNow, err := s.userAssetRepo.SubGoldByID(ctx, tx, userID, price)
		if err != nil {
			return err
		}
		if userAssetNow == nil {
			return ErrUserNotFound
		}

		if currentVIP == req.VIPLevel {
			activeAt, err = vipActiveAt(req.Duration, *userAsset.VIPActiveAt)
		} else {
			activeAt, err = vipActiveAt(req.Duration, time.Now())
		}
		if err != nil {
			return err
		}
		userAssetNow.VIPLevel = req.VIPLevel
		userAssetNow.VIPActiveAt = &activeAt

		err = s.userAssetRepo.UpdateByID(ctx, tx, userID, userAssetNow)
		if err != nil {
			return err
		}
		purchasedVIP = userAssetNow
		return nil
	})
	if err != nil {
		return nil, err
	}

	return toBuyVIPResponse(purchasedVIP, price, req.Duration), nil
}

func (s *buyVIPService) UpgradeVIP(ctx context.Context, userID string, req *dto.UpgradeVIPRequest) (*dto.UpgradeVIPResponse, error) {
	if req == nil || req.Price <= 0 || req.OriginalVIPLevel != vipLevel1 || req.TargetVIPLevel != vipLevel2 {
		return nil, ErrParam
	}

	var upgradedVIP *model.UserAsset
	var upgradePrice int64

	err := s.db.Transaction(func(tx *gorm.DB) error {
		user, err := s.userRepo.GetByID(ctx, tx, userID)
		if err != nil {
			return err
		}
		if user == nil {
			return ErrUserNotFound
		}

		userAsset, err := s.userAssetRepo.GetByID(ctx, tx, userID)
		if err != nil {
			return err
		}
		if userAsset == nil {
			return ErrUserNotFound
		}

		currentVIP := effectiveVIPLevel(userAsset, time.Now())
		if currentVIP != vipLevel1 || currentVIP != req.OriginalVIPLevel {
			return ErrParam
		}

		upgradePrice, err = s.upgradeVIPPrice(*userAsset.VIPActiveAt)
		if err != nil {
			return err
		}
		if req.Price != upgradePrice {
			return ErrParam
		}

		if userAsset.Gold < upgradePrice {
			return ErrSufficientFunds
		}

		userAssetNow, err := s.userAssetRepo.SubGoldByID(ctx, tx, userID, upgradePrice)
		if err != nil {
			return err
		}
		if userAssetNow == nil {
			return ErrUserNotFound
		}

		userAssetNow.VIPLevel = req.TargetVIPLevel
		userAssetNow.VIPActiveAt = userAsset.VIPActiveAt
		err = s.userAssetRepo.UpdateByID(ctx, tx, userID, userAssetNow)
		if err != nil {
			return err
		}
		upgradedVIP = userAssetNow
		return nil
	})
	if err != nil {
		return nil, err
	}

	return toUpgradeVIPResponse(upgradedVIP, req.OriginalVIPLevel, req.TargetVIPLevel, upgradePrice), nil
}

func (s *buyVIPService) ShopListOfVIP(ctx context.Context, userID string) (*dto.ShopListOfVIPResponse, error) {
	userAsset, err := s.userAssetRepo.GetByID(ctx, s.db, userID)
	if err != nil {
		return nil, err
	}
	if userAsset == nil {
		return nil, ErrUserNotFound
	}

	now := time.Now()
	currentVIP := effectiveVIPLevel(userAsset, now)
	shop := make([]dto.ShopVIPItem, 0)

	switch currentVIP {
	case vipLevel0:
		shop = append(shop, s.shopPlan(vipShopTypeBuy, vipLevel1, vipDurationMonth, now))
		shop = append(shop, s.shopPlan(vipShopTypeBuy, vipLevel1, vipDurationYear, now))
		shop = append(shop, s.shopPlan(vipShopTypeBuy, vipLevel2, vipDurationMonth, now))
		shop = append(shop, s.shopPlan(vipShopTypeBuy, vipLevel2, vipDurationYear, now))
	case vipLevel1:
		shop = append(shop, s.shopPlan(vipShopTypeRenew, vipLevel1, vipDurationMonth, *userAsset.VIPActiveAt))
		shop = append(shop, s.shopPlan(vipShopTypeRenew, vipLevel1, vipDurationYear, *userAsset.VIPActiveAt))
		upgradePrice, err := s.upgradeVIPPrice(*userAsset.VIPActiveAt)
		if err != nil {
			return nil, err
		}
		shop = append(shop, dto.ShopVIPItem{
			Type:     vipShopTypeUpgrade,
			VIPLevel: vipLevel2,
			Price:    upgradePrice,
			ActiveAt: userAsset.VIPActiveAt,
		})
	case vipLevel2:
		shop = append(shop, s.shopPlan(vipShopTypeRenew, vipLevel2, vipDurationMonth, *userAsset.VIPActiveAt))
		shop = append(shop, s.shopPlan(vipShopTypeRenew, vipLevel2, vipDurationYear, *userAsset.VIPActiveAt))
	default:
		return nil, ErrParam
	}

	return &dto.ShopListOfVIPResponse{Shop: shop}, nil
}

func (s *buyVIPService) vipPrice(currentVIP string, targetVIP string, duration string) (int64, error) {
	switch currentVIP {
	case vipLevel0:
	case vipLevel1:
		if targetVIP == vipLevel2 {
			return 0, ErrParam
		}
	case vipLevel2:
		if targetVIP == vipLevel1 {
			return 0, ErrParam
		}
	default:
		return 0, ErrParam
	}

	switch targetVIP {
	case vipLevel1:
		return priceByDuration(s.cfg.Purchased.VIP1, duration)
	case vipLevel2:
		return priceByDuration(s.cfg.Purchased.VIP2, duration)
	default:
		return 0, ErrParam
	}
}

// upgradeVIPPrice 计算从 VIP1 升级到 VIP2 的差价。
// 公式：((过期时间 - 当前时间) / 30 + 1) * 每月差价
func (s *buyVIPService) upgradeVIPPrice(vipActiveAt time.Time) (int64, error) {
	monthlyDiff := s.cfg.Purchased.VIP2.Month - s.cfg.Purchased.VIP1.Month
	if monthlyDiff <= 0 {
		return 0, ErrParam
	}
	now := time.Now()
	remainingDays := int64(vipActiveAt.Sub(now).Hours() / 24)
	if remainingDays < 0 {
		remainingDays = 0
	}
	price := (remainingDays/30 + 1) * monthlyDiff
	return price, nil
}

func (s *buyVIPService) shopPlan(shopType string, vipLevel string, duration string, baseAt time.Time) dto.ShopVIPItem {
	var priceConfig config.VIPPriceConfig
	switch vipLevel {
	case vipLevel1:
		priceConfig = s.cfg.Purchased.VIP1
	case vipLevel2:
		priceConfig = s.cfg.Purchased.VIP2
	}

	price, _ := priceByDuration(priceConfig, duration)
	activeAt, _ := vipActiveAt(duration, baseAt)
	return dto.ShopVIPItem{
		Type:     shopType,
		VIPLevel: vipLevel,
		Price:    price,
		Duration: duration,
		ActiveAt: &activeAt,
	}
}

func priceByDuration(price config.VIPPriceConfig, duration string) (int64, error) {
	switch duration {
	case vipDurationMonth:
		return price.Month, nil
	case vipDurationYear:
		return price.Year, nil
	default:
		return 0, ErrParam
	}
}

func effectiveVIPLevel(userAsset *model.UserAsset, now time.Time) string {
	if userAsset == nil || userAsset.VIPActiveAt == nil || !userAsset.VIPActiveAt.After(now) {
		return vipLevel0
	}

	switch userAsset.VIPLevel {
	case vipLevel1, vipLevel2:
		return userAsset.VIPLevel
	default:
		return vipLevel0
	}
}

func vipActiveAt(duration string, now time.Time) (time.Time, error) {
	switch duration {
	case vipDurationMonth:
		return now.AddDate(0, 1, 0), nil
	case vipDurationYear:
		return now.AddDate(1, 0, 0), nil
	default:
		return time.Time{}, ErrParam
	}
}

func toBuyVIPResponse(userAsset *model.UserAsset, price int64, duration string) *dto.BuyVIPResponse {
	return &dto.BuyVIPResponse{
		UserID:      userAsset.ID,
		Gold:        userAsset.Gold,
		VIPMark:     userAsset.VIPLevel,
		VIPLevel:    VIPStrToLevel(userAsset.VIPLevel),
		VIPActiveAt: userAsset.VIPActiveAt,
		Price:       price,
		Duration:    duration,
	}
}

func toUpgradeVIPResponse(userAsset *model.UserAsset, originalVIPLevel string, targetVIPLevel string, price int64) *dto.UpgradeVIPResponse {
	return &dto.UpgradeVIPResponse{
		UserID:           userAsset.ID,
		Gold:             userAsset.Gold,
		OriginalVIPLevel: originalVIPLevel,
		TargetVIPLevel:   targetVIPLevel,
		VIPMark:          userAsset.VIPLevel,
		VIPLevel:         VIPStrToLevel(userAsset.VIPLevel),
		VIPActiveAt:      userAsset.VIPActiveAt,
		Price:            price,
	}
}
