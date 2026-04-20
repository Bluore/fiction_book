package service

import (
	"context"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
	"fiction-book/pkg/constant"
	"fiction-book/pkg/logger"
	"fiction-book/pkg/redisclient"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type UserBookshelfService interface {
	GetUserBookshelfList(ctx context.Context, req *dto.ListUserBookshelfRequest) (*dto.ListUserBookshelfResponse, error)
	CreateUserBookshelf(ctx context.Context, userID string, req *dto.CreateUserBookshelfRequest) error
}

type userBookshelfService struct {
	userRepo          repository.UserRepository
	bookRepo          repository.BookRepository
	userBookshelfRepo repository.UserBookshelfRepository
	db                *gorm.DB
	cfg               *config.Config
}

func NewUserBookshelfService(
	userRepo repository.UserRepository,
	bookRepo repository.BookRepository,
	userBookshelfRepo repository.UserBookshelfRepository,
	db *gorm.DB,
	cfg *config.Config,
) UserBookshelfService {
	return &userBookshelfService{
		userRepo:          userRepo,
		bookRepo:          bookRepo,
		userBookshelfRepo: userBookshelfRepo,
		db:                db,
		cfg:               cfg,
	}
}

func (s userBookshelfService) GetUserBookshelfList(ctx context.Context, req *dto.ListUserBookshelfRequest) (*dto.ListUserBookshelfResponse, error) {
	books, err := s.userBookshelfRepo.ListByUserID(ctx, s.db, req.UserID, req.Page, req.Size)
	if err != nil {
		return nil, err
	}

	bookRes := make([]*dto.UserBookshelfResponse, len(books))
	for i, book := range books {
		bookRes[i] = toUserBookshelfResponse(s.cfg, book)
	}

	count, err := s.userBookshelfRepo.CountByUserID(ctx, s.db, req.UserID)
	if err != nil {
		return nil, err
	}

	page := dto.PageListResponse{
		Page:  req.Page,
		Size:  req.Size,
		Count: count,
	}

	res := &dto.ListUserBookshelfResponse{
		Books:            bookRes,
		PageListResponse: page,
	}
	return res, nil
}

func (s userBookshelfService) CreateUserBookshelf(ctx context.Context, userID string, req *dto.CreateUserBookshelfRequest) error {
	err := s.db.Transaction(func(db *gorm.DB) error {
		_, err := s.bookRepo.IncreaseOfBookShelf(ctx, db, req.BookID)
		if err != nil {
			return err
		}
		err = s.userBookshelfRepo.CreateByUserIDAndBookID(ctx, s.db, userID, req.BookID)
		if err != nil {
			return err
		}

		// redis中增加热力值
		_, err = redisclient.Get(ctx, constant.RedisIsUserBookshelf5Min(userID))
		if err != nil && err != redis.Nil {
			return err
		}
		if err == redis.Nil {
			client, err := redisclient.GetClient()
			if err != nil {
				logger.Error(fmt.Sprint("redis client is nil:%v", err))
				return nil
			}
			// 热力值限流
			client.Set(ctx, constant.RedisIsUserBookshelf5Min(userID), true, time.Duration(5*time.Minute))

			// 如果排行榜没有字段则创建书籍
			bookRankErr := client.Get(ctx, constant.RedisBookThermalDaily(time.Now())).Err()
			if bookRankErr == redis.Nil {
				bookshelfArgs := redis.Z{Score: 0, Member: userID}
				client.ZAdd(ctx, constant.RedisBookThermalDaily(time.Now()), bookshelfArgs)
			}

			// 排行榜增加权值
			client.ZIncrBy(ctx, constant.RedisBookThermalDaily(time.Now()), 10, userID)

			// 如果没有过期时间则设置过期时间
			expRes := client.TTL(ctx, constant.RedisBookThermalDaily(time.Now()))
			if expRes.Err() != nil {
				logger.Error(fmt.Sprintf("redis:set book thermal daily ttl fail:%v", expRes.Err()))
				return nil
			}
			if expRes.Val() == -1 {
				client.Expire(ctx, constant.RedisBookThermalDaily(time.Now()), time.Duration(3*24*time.Hour))
			}
		}

		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func toUserBookshelfResponse(_ *config.Config, in *model.UserBookshelf) *dto.UserBookshelfResponse {
	return &dto.UserBookshelfResponse{
		ID:        in.ID,
		UserID:    in.UserID,
		BookID:    in.BookID,
		CreatedAt: in.CreatedAt,
	}
}
