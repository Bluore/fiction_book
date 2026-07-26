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
)

type BuyVIP interface {
	BuyVIP(ctx context.Context, userID string, req *dto.BuyVIPRequest) (*dto.BuyVIPResponse, error)
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
