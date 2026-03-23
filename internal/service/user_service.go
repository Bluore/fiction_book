package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
	"fiction-book/pkg/email"
	"fiction-book/pkg/jwt"
	"fiction-book/pkg/redis"
)

var (
	ErrUserExists      = errors.New("user already exists")
	ErrUserNotFound    = errors.New("user not found")
	ErrInvalidPassword = errors.New("invalid password")
	ErrVerification    = errors.New("verification failed")
	ErrEmailVerifySend = errors.New("email verification not send")
)

// UserService 用户服务接口
type UserService interface {
	Create(ctx context.Context, req *dto.CreateUserRequest) (*dto.UserResponse, error)
	GetByID(ctx context.Context, id string) (*dto.UserResponse, error)
	Update(ctx context.Context, id string, req *dto.UpdateUserRequest) (*dto.UserResponse, error)
	Delete(ctx context.Context, id string) error
	Login(ctx context.Context, req *dto.LoginRequest) (*dto.LoginResponse, error)
	List(ctx context.Context, page, pageSize int) ([]*dto.UserResponse, error)
	EmailVerify(cfg *config.Config, ctx context.Context, receiver string) error
}

type userService struct {
	userRepo repository.UserRepository
	cfg      *config.Config
}

// NewUserService 创建用户服务实例
func NewUserService(userRepo repository.UserRepository, cfg *config.Config) UserService {
	return &userService{
		userRepo: userRepo,
		cfg:      cfg,
	}
}

func (s *userService) Create(ctx context.Context, req *dto.CreateUserRequest) (*dto.UserResponse, error) {
	// 验证码是否正确
	rVerify, err := redis.Get(
		ctx,
		fmt.Sprintf("email:verify:%s", req.Email),
	)
	if err != nil {
		return nil, ErrEmailVerifySend
	}
	if req.Verify != rVerify {
		return nil, ErrVerification
	}

	// 检查用户名是否已存在
	existingUser, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, ErrUserExists
	}

	// 检查邮箱是否已存在
	existingUser, err = s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, ErrUserExists
	}

	// 加密密码
	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// 创建用户
	user := &model.User{
		ID:       uuid.New().String(),
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPassword, // pragma: allowlist secret
		Age:      req.Age,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return s.toUserResponse(user), nil
}

func (s *userService) GetByID(ctx context.Context, id string) (*dto.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return s.toUserResponse(user), nil
}

func (s *userService) Update(ctx context.Context, id string, req *dto.UpdateUserRequest) (*dto.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// 更新字段
	if req.Username != nil {
		user.Username = *req.Username
	}
	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.Age != nil {
		user.Age = *req.Age
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return s.toUserResponse(user), nil
}

func (s *userService) Delete(ctx context.Context, id string) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	return s.userRepo.Delete(ctx, id)
}

func (s *userService) Login(ctx context.Context, req *dto.LoginRequest) (*dto.LoginResponse, error) {
	user, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// 验证密码
	if !checkPasswordHash(req.Password, user.Password) {
		return nil, ErrInvalidPassword
	}

	// 生成 JWT token
	token, err := jwt.GenerateToken(user.ID, user.Username, s.cfg.JWT.Secret, s.cfg.JWT.Expire) // pragma: allowlist secret
	if err != nil {
		return nil, err
	}

	return &dto.LoginResponse{
		Token:    token,
		UserID:   user.ID,
		Username: user.Username,
	}, nil
}

func (s *userService) List(ctx context.Context, page, pageSize int) ([]*dto.UserResponse, error) {
	offset := (page - 1) * pageSize
	users, err := s.userRepo.List(ctx, offset, pageSize)
	if err != nil {
		return nil, err
	}

	responses := make([]*dto.UserResponse, len(users))
	for i, user := range users {
		responses[i] = s.toUserResponse(user)
	}

	return responses, nil
}

func (s *userService) toUserResponse(user *model.User) *dto.UserResponse {
	return &dto.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Age:       user.Age,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

// hashPassword 加密密码
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// checkPasswordHash 验证密码
func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (s *userService) EmailVerify(cfg *config.Config, ctx context.Context, receiver string) error {
	// 生成验证码
	num, err := email.CreateVerifyNumber()
	if err != nil {
		return err
	}

	// redis添加记录
	err = redis.SetWithExpiration(
		ctx,
		fmt.Sprintf("email:verify:%s",
			receiver),
		num,
		10*time.Minute,
	)
	if err != nil {
		log.Printf("redis add value error: %s\n", err.Error())
		return err
	}

	// 发送邮件
	if err := email.CreateEmailOne(
		cfg,
		receiver,
		"GSF小说 你的一次性邮箱验证码",
		fmt.Sprintf("邮箱验证密码: %s </br> 10分钟内有效", num),
	); err != nil {
		return err
	}
	return nil
}
