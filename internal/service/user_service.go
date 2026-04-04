package service

import (
	"context"
	"errors"
	"fiction-book/pkg/logger"
	"fmt"
	"image"
	"io"
	"log"
	"mime/multipart"
	"os"
	"strings"
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
	ErrEmailExists     = errors.New("email already exists")
	ErrUserNotFound    = errors.New("user not found")
	ErrInvalidPassword = errors.New("invalid password")
	ErrVerification    = errors.New("verification failed")
	ErrEmailVerifySend = errors.New("email verification not send")
	ErrSizeExceed      = errors.New("size exceed")
	ErrImage           = errors.New("image error")
	ErrFileType        = errors.New("file type error")
	ErrUserNotLogin    = errors.New("user need login")

	imageFilePath = "/upload/avatar/"
)

// UserService 用户服务接口
type UserService interface {
	Create(ctx context.Context, req *dto.CreateUserRequest) (*dto.UserResponse, error)
	GetByID(ctx context.Context, id string) (*dto.UserResponse, error)
	Update(ctx context.Context, id string, req *dto.UpdateUserRequest) (*dto.UserResponse, error)
	UpdatePassword(ctx context.Context, id string, req *dto.UpdateUserPasswordRequest) (*dto.UserResponse, error)
	UpdateEmail(ctx context.Context, id string, req *dto.UpdateUserEmailRequest) (*dto.UserResponse, error)
	Delete(ctx context.Context, id string) error
	Login(ctx context.Context, req *dto.LoginRequest) (*dto.LoginResponse, error)
	List(ctx context.Context, page, pageSize int) ([]*dto.UserResponse, error)
	SendEmailVerification(ctx context.Context, receiver string) error
	UpdateAvatar(ctx context.Context, userID string, avatar *multipart.FileHeader) (*dto.UserResponse, error)
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

func (s *userService) UpdateEmail(ctx context.Context, id string, req *dto.UpdateUserEmailRequest) (*dto.UserResponse, error) {
	// 获取用户信息
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	if !checkPasswordHash(req.Password, user.Password) {
		return nil, ErrInvalidPassword
	}

	err = s.emailVerify(ctx, req.Email, req.Verify)
	if err != nil {
		return nil, err
	}

	exist, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if exist != nil {
		return nil, ErrEmailExists
	}

	user.Email = req.Email
	err = s.userRepo.Update(ctx, user)
	if err != nil {
		return nil, err
	}
	return s.toUserResponse(user), nil
}

func (s *userService) UpdatePassword(ctx context.Context, id string, req *dto.UpdateUserPasswordRequest) (*dto.UserResponse, error) {
	// 获取用户信息
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	err = s.emailVerify(ctx, user.Email, req.Verify)
	if err != nil {
		return nil, err
	}

	// 密码加密
	hashPwd, err := hashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	user.Password = hashPwd

	err = s.userRepo.Update(ctx, user)
	if err != nil {
		return nil, err
	}
	return s.toUserResponse(user), nil
}

func (s *userService) Create(ctx context.Context, req *dto.CreateUserRequest) (*dto.UserResponse, error) {
	err := s.emailVerify(ctx, req.Email, req.Verify)
	if err != nil {
		return nil, err
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
		return nil, ErrEmailExists
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

func (s *userService) emailVerify(ctx context.Context, emailVer string, pwdVer string) error {
	if s.cfg.Server.Mode == "debug" {
		return nil
	}

	// 验证码是否正确
	rVerify, err := redis.Get(
		ctx,
		fmt.Sprintf("email:verify:%s", emailVer),
	)
	if err != nil {
		return ErrEmailVerifySend
	}
	if pwdVer != rVerify {
		return ErrVerification
	}
	err = redis.Delete(
		ctx,
		fmt.Sprintf("email:verify:%s", emailVer),
	)
	if err != nil {
		return nil
	}
	return nil
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
	if req.Age != nil {
		user.Age = *req.Age
	}
	if req.Description != nil {
		user.Description = *req.Description
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
		ID:           user.ID,
		Username:     user.Username,
		Email:        user.Email,
		Age:          user.Age,
		Description:  user.Description,
		ProfileImage: s.getProfileImageUri(user.ProfileImage),
		CoverImage:   s.getCoverImageUri(user.CoverImage),
		CreatedAt:    user.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:    user.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

// 获取用户的头像绝对链接
func (s *userService) getProfileImageUri(path string) string {
	if path == "" {
		return fmt.Sprintf("%s%s", s.cfg.Server.Website, "/upload/avatar/default.jpg")
	} else {
		return fmt.Sprintf("%s%s", s.cfg.Server.Website, path)
	}
}

// 获取用户的头图绝对链接
func (s *userService) getCoverImageUri(path string) string {
	if path == "" {
		return fmt.Sprintf("%s%s", s.cfg.Server.Website, "/upload/cover_image/default.jpg")
	} else {
		return fmt.Sprintf("%s%s", s.cfg.Server.Website, path)
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

func (s *userService) SendEmailVerification(ctx context.Context, receiver string) error {
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
		s.cfg,
		receiver,
		"GSF小说 你的一次性邮箱验证码",
		fmt.Sprintf("邮箱验证密码: %s </br> 10分钟内有效", num),
	); err != nil {
		return err
	}
	return nil
}

func (s *userService) UpdateAvatar(ctx context.Context, userID string, avatar *multipart.FileHeader) (*dto.UserResponse, error) {
	// 判断图像大小
	if avatar.Size > 2*1024*1024 {
		return nil, ErrSizeExceed
	}
	// 判断图像类型
	if len(strings.Split(avatar.Filename, ".")) > 2 {
		return nil, ErrFileType
	}
	suf := strings.Split(avatar.Filename, ".")[1]
	if suf != "png" && suf != "jpg" {
		return nil, ErrFileType
	}

	// 获取信息
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	var imgId string
	if user.ProfileImage == "" {
		imgId = uuid.New().String()
	} else {
		imgId = strings.TrimPrefix(user.ProfileImage, imageFilePath)
		imgId = strings.Split(imgId, ".")[0]

		_, err := uuid.Parse(imgId)
		if err != nil {
			return nil, err
		}
	}
	path := fmt.Sprintf("%s%s.%s", imageFilePath, imgId, suf)

	// 打开图像
	src, err := avatar.Open()
	if err != nil {
		return nil, err
	}
	defer func(src multipart.File) {
		err := src.Close()
		if err != nil {
			logger.Error("image file close error: " + err.Error())
		}
	}(src)

	// 解码图像
	img, _, err := image.Decode(src)
	if err != nil {
		return nil, err
	}

	// 判断图像像素大小
	if img.Bounds().Dx() != 180 || img.Bounds().Dy() != 180 {
		return nil, ErrImage
	}

	// 创建文件
	dstFile, err := os.Create(fmt.Sprintf(".%s", path))
	if err != nil {
		return nil, err
	}
	defer func(dstFile *os.File) {
		err := dstFile.Close()
		if err != nil {
			logger.Error("image file close error: " + err.Error())
		}
	}(dstFile)

	// 重置指针流
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	// 复制文件
	_, err = io.Copy(dstFile, src)
	if err != nil {
		return nil, err
	}

	// 更新数据库
	user.ProfileImage = path
	err = s.userRepo.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	return s.toUserResponse(user), nil
}
