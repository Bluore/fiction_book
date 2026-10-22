package service

import (
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"log"
	"mime/multipart"
	"os"
	"strings"
	"time"

	"fiction-book/internal/cache"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
	"fiction-book/pkg/email"
	"fiction-book/pkg/jwt"
	"fiction-book/pkg/logger"
	"fiction-book/pkg/parameter"
	"fiction-book/pkg/response"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var imageFilePath = "/upload/avatar/"

// UserService 用户服务接口
type UserService interface {
	Create(ctx context.Context, req *dto.CreateUserRequest) (*dto.UserResponse, error)
	GetByID(ctx context.Context, id string) (*dto.UserResponse, error)
	Update(ctx context.Context, id string, req *dto.UpdateUserRequest) (*dto.UserResponse, error)
	UpdatePassword(ctx context.Context, id string, req *dto.UpdateUserPasswordRequest) (*dto.UserResponse, error)
	UpdateEmail(ctx context.Context, id string, req *dto.UpdateUserEmailRequest) (*dto.UserResponse, error)
	Delete(ctx context.Context, id string) error
	Login(ctx context.Context, req *dto.LoginRequest) (*dto.LoginResponse, error)
	RefreshToken(ctx context.Context, req *dto.RefreshTokenRequest) (*dto.RefreshTokenResponse, error)
	List(ctx context.Context, page, pageSize int) ([]*dto.UserResponse, error)
	SendEmailVerification(ctx context.Context, receiver string) error
	UpdateAvatar(ctx context.Context, userID string, avatar *multipart.FileHeader) (*dto.UserResponse, error)
	GetStatByID(ctx context.Context, userID string) (*dto.UserStatResponse, error)
	ForgetPassword(ctx context.Context, req *dto.ForgetPasswordRequest) error
}

type userService struct {
	userRepo       repository.UserRepository
	userAssetRepo  repository.UserAssetRepository
	userInfoRepo   repository.UserInfoRepository
	userCache      cache.UserCache
	userStatCache  cache.UserStatCache
	userAssetCache cache.UserAssetCache
	userInfoCache  cache.UserInfoCache
	authCache      cache.AuthCache
	db             *gorm.DB
	cfg            *config.Config
}

// NewUserService 创建用户服务实例
func NewUserService(
	userRepo repository.UserRepository,
	userAssetRepo repository.UserAssetRepository,
	userInfoRepo repository.UserInfoRepository,
	userCache cache.UserCache,
	userStatCache cache.UserStatCache,
	userAssetCache cache.UserAssetCache,
	userInfoCache cache.UserInfoCache,
	authCache cache.AuthCache,
	db *gorm.DB,
	cfg *config.Config,
) UserService {
	return &userService{
		userRepo:       userRepo,
		userAssetRepo:  userAssetRepo,
		userInfoRepo:   userInfoRepo,
		userCache:      userCache,
		userStatCache:  userStatCache,
		userAssetCache: userAssetCache,
		userInfoCache:  userInfoCache,
		authCache:      authCache,
		db:             db,
		cfg:            cfg,
	}
}

func (s *userService) UpdateEmail(ctx context.Context, id string, req *dto.UpdateUserEmailRequest) (*dto.UserResponse, error) {
	// 获取用户信息（密码哈希不进入缓存，校验密码需读取数据库）
	user, err := s.userRepo.GetByID(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, response.ErrUserNotFound
	}

	if !checkPasswordHash(req.Password, user.Password) {
		return nil, response.ErrInvalidPassword
	}

	err = s.emailVerify(ctx, req.Email, req.Verify)
	if err != nil {
		return nil, err
	}

	exist, err := s.userRepo.GetByEmail(ctx, s.db, req.Email)
	if err != nil {
		return nil, err
	}
	if exist != nil {
		return nil, response.ErrEmailExists
	}

	user.Email = req.Email
	if err := s.userRepo.Update(ctx, s.db, user); err != nil {
		return nil, err
	}
	if _, err := s.userCache.DeleteUser(ctx, user.ID); err != nil {
		logger.Warn("delete user cache fail", zap.Error(err))
	}
	return s.toUserResponse(user), nil
}

func (s *userService) UpdatePassword(ctx context.Context, id string, req *dto.UpdateUserPasswordRequest) (*dto.UserResponse, error) {
	// 获取用户信息（密码哈希不进入缓存，更新需要完整用户记录）
	user, err := s.userRepo.GetByID(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, response.ErrUserNotFound
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

	if err := s.userRepo.Update(ctx, s.db, user); err != nil {
		return nil, err
	}
	if _, err := s.userCache.DeleteUser(ctx, user.ID); err != nil {
		logger.Warn("delete user cache fail", zap.Error(err))
	}
	return s.toUserResponse(user), nil
}

func (s *userService) Create(ctx context.Context, req *dto.CreateUserRequest) (*dto.UserResponse, error) {
	err := s.emailVerify(ctx, req.Email, req.Verify)
	if err != nil {
		return nil, err
	}

	var (
		user     *model.User
		userInfo *model.UserInfo
	)

	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 检查用户名是否已存在
		existingUser, err := s.userRepo.GetByUsername(ctx, tx, req.Username)
		if err != nil {
			return err
		}
		if existingUser != nil {
			return response.ErrUserExists
		}

		// 检查邮箱是否已存在
		existingUser, err = s.userRepo.GetByEmail(ctx, tx, req.Email)
		if err != nil {
			return err
		}
		if existingUser != nil {
			return response.ErrEmailExists
		}

		// 加密密码
		hashedPassword, err := hashPassword(req.Password)
		if err != nil {
			return err
		}

		// 创建用户
		role := "user"
		// 在 debug 模式下，如果用户名为 admin，则创建管理员账号
		if s.cfg.Server.Mode == "debug" && req.Username == "admin" {
			role = "admin"
		}
		user = &model.User{
			ID:       uuid.New().String(),
			Username: req.Username,
			Email:    req.Email,
			Password: hashedPassword, // pragma: allowlist secret
			Age:      req.Age,
			Role:     role,
		}

		if err := s.userRepo.Create(ctx, tx, user); err != nil {
			return err
		}

		// 创建用户详情
		userInfo = &model.UserInfo{
			UserID:         user.ID,
			Username:       user.Username,
			UserProfileUrl: user.ProfileImage,
		}
		if err := s.userInfoRepo.Create(ctx, tx, userInfo); err != nil {
			return err
		}

		// 创建用户资产
		if err := s.userAssetRepo.CreateUserAsset(ctx, tx, user.ID); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// 事务提交成功后回写缓存，避免事务回滚时写入脏缓存
	if err := s.userCache.UpdateUser(ctx, user.ID, user); err != nil {
		logger.Warn("update user cache fail", zap.Error(err))
	}
	if err := s.userInfoCache.UpdateUserInfo(ctx, user.ID, userInfo); err != nil {
		logger.Warn("update user_info cache fail", zap.Error(err))
	}

	return s.toUserResponse(user), nil
}

func (s *userService) emailVerify(ctx context.Context, emailVer string, pwdVer string) error {
	if s.cfg.Server.Mode == "debug" {
		return nil
	}

	// 验证码是否正确
	rVerify, err := s.authCache.GetEmailVerification(ctx, emailVer)
	if err != nil {
		return response.ErrEmailVerifySend
	}
	if pwdVer != rVerify {
		return response.ErrVerification
	}
	err = s.authCache.DeleteEmailVerification(ctx, emailVer)

	if err != nil {
		return nil
	}
	return nil
}

func (s *userService) GetByID(ctx context.Context, id string) (*dto.UserResponse, error) {
	user, err := queryUserDetailWithCache(ctx, s.db, s.userCache, s.userRepo, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, response.ErrUserNotFound
	}
	return s.toUserResponse(user), nil
}

func (s *userService) Update(ctx context.Context, id string, req *dto.UpdateUserRequest) (*dto.UserResponse, error) {
	// 密码哈希不进入缓存，更新需要完整用户记录，直接从数据库读取
	user, err := s.userRepo.GetByID(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, response.ErrUserNotFound
	}

	// 更新字段
	usernameChanged := req.Username != nil && *req.Username != user.Username
	if req.Username != nil {
		user.Username = *req.Username
	}
	if req.Age != nil {
		user.Age = *req.Age
	}
	if req.Description != nil {
		user.Description = *req.Description
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.userRepo.Update(ctx, tx, user); err != nil {
			return err
		}
		if !usernameChanged {
			return nil
		}
		return s.userInfoRepo.UpdateUsername(ctx, tx, user.ID, user.Username)
	})
	if err != nil {
		return nil, err
	}
	if _, err := s.userCache.DeleteUser(ctx, user.ID); err != nil {
		logger.Warn("delete user cache fail", zap.Error(err))
	}
	if usernameChanged {
		if _, err := s.userInfoCache.DeleteUserInfo(ctx, user.ID); err != nil {
			logger.Warn("delete user_info cache fail", zap.Error(err))
		}
	}

	return s.toUserResponse(user), nil
}

func (s *userService) Delete(ctx context.Context, id string) error {
	user, err := queryUserDetailWithCache(ctx, s.db, s.userCache, s.userRepo, id)
	if err != nil {
		return err
	}
	if user == nil {
		return response.ErrUserNotFound
	}

	if err := s.userRepo.Delete(ctx, s.db, id); err != nil {
		return err
	}
	if _, err := s.userCache.DeleteUser(ctx, id); err != nil {
		logger.Warn("delete user cache fail", zap.Error(err))
	}
	return nil
}

func (s *userService) Login(ctx context.Context, req *dto.LoginRequest) (*dto.LoginResponse, error) {
	user, err := s.userRepo.GetByUsername(ctx, s.db, req.Username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, response.ErrUserNotFound
	}

	// 验证密码
	if !checkPasswordHash(req.Password, user.Password) {
		return nil, response.ErrInvalidPassword
	}

	token, refreshToken, err := s.issueTokens(ctx, user)
	if err != nil {
		return nil, err
	}

	return &dto.LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresIn:    s.cfg.JWT.Expire,
		UserID:       user.ID,
		Username:     user.Username,
	}, nil
}

// defaultRefreshExpire RefreshToken 默认有效期：7 天
const defaultRefreshExpire = 7 * 24 * 60 * 60

// refreshExpire 获取 RefreshToken 有效期（秒）
func (s *userService) refreshExpire() int {
	if s.cfg.JWT.RefreshExpire > 0 {
		return s.cfg.JWT.RefreshExpire
	}
	return defaultRefreshExpire
}

// issueTokens 生成 AccessToken 与 RefreshToken，并把 RefreshToken 写入 Redis 白名单
// 每个用户只保留最新的 RefreshToken，新登录会顶掉旧登录，实现单点登录
func (s *userService) issueTokens(ctx context.Context, user *model.User) (string, string, error) {
	accessToken, err := jwt.GenerateTokenWithType(user.ID, user.Username, user.Role, jwt.TokenTypeAccess, s.cfg.JWT.Secret, s.cfg.JWT.Expire) // pragma: allowlist secret
	if err != nil {
		return "", "", err
	}

	refreshExpire := s.refreshExpire()
	refreshToken, err := jwt.GenerateTokenWithType(user.ID, user.Username, user.Role, jwt.TokenTypeRefresh, s.cfg.JWT.Secret, refreshExpire) // pragma: allowlist secret
	if err != nil {
		return "", "", err
	}

	if err := s.authCache.UpdateRefreshToken(ctx, user.ID, refreshToken, time.Duration(refreshExpire)*time.Second); err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// RefreshToken 使用 RefreshToken 换取新的 AccessToken
// RefreshToken 必须存在于 Redis 白名单中，且只允许当前登录端使用
func (s *userService) RefreshToken(ctx context.Context, req *dto.RefreshTokenRequest) (*dto.RefreshTokenResponse, error) {
	claims, err := jwt.ParseToken(req.RefreshToken, s.cfg.JWT.Secret)
	if err != nil {
		// RefreshToken 过期或非法，只能重新登录
		return nil, response.ErrUserNotLogin
	}
	if claims.TokenType != jwt.TokenTypeRefresh {
		return nil, response.ErrUserNotLogin
	}

	whitelistToken, hit, err := s.authCache.GetRefreshToken(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	if !hit || whitelistToken != req.RefreshToken {
		return nil, response.ErrUserNotLogin
	}

	user, err := s.userRepo.GetByID(ctx, s.db, claims.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, response.ErrUserNotFound
	}

	token, refreshToken, err := s.issueTokens(ctx, user)
	if err != nil {
		return nil, err
	}

	return &dto.RefreshTokenResponse{
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresIn:    s.cfg.JWT.Expire,
	}, nil
}

func (s *userService) List(ctx context.Context, page, pageSize int) ([]*dto.UserResponse, error) {
	offset := (page - 1) * pageSize
	users, err := s.userRepo.List(ctx, s.db, offset, pageSize)
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
	err = s.authCache.UpdateEmailVerification(
		ctx,
		receiver,
		num,
		5*time.Minute,
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
	// 获取信息（密码哈希不进入缓存，更新需要完整用户记录，直接从数据库读取）
	user, err := s.userRepo.GetByID(ctx, s.db, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, response.ErrUserNotFound
	}

	// 获取ID
	imgId := getFileID(user.ProfileImage, imageFilePath)
	if imgId == "" {
		return nil, errors.New("imageID decode error")
	}

	// 上传图片
	uploadOut, err := uploadImage(&parameter.UploadImageIn{
		Avatar:     avatar,
		ImageId:    imgId,
		ImagePath:  imageFilePath,
		SizeWidth:  180,
		SizeHeight: 180,
		SizeFile:   2 * 1024 * 1024,
		AcceptType: map[string]bool{
			"png": true,
			"jpg": true,
		},
	})
	if err != nil {
		return nil, err
	}

	// 更新数据库
	user.ProfileImage = uploadOut.FilePath
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.userRepo.Update(ctx, tx, user); err != nil {
			return err
		}
		return s.userInfoRepo.UpdateProfileURL(ctx, tx, user.ID, user.ProfileImage)
	})
	if err != nil {
		return nil, err
	}
	if _, err := s.userCache.DeleteUser(ctx, user.ID); err != nil {
		logger.Warn("delete user cache fail", zap.Error(err))
	}
	if _, err := s.userInfoCache.DeleteUserInfo(ctx, user.ID); err != nil {
		logger.Warn("delete user_info cache fail", zap.Error(err))
	}

	return s.toUserResponse(user), nil
}

func uploadImage(in *parameter.UploadImageIn) (*parameter.UploadImageOut, error) {
	// 判断图像大小
	if in.Avatar.Size > in.SizeFile {
		return nil, response.ErrSizeExceed
	}
	// 判断图像类型
	if len(strings.Split(in.Avatar.Filename, ".")) > 2 {
		return nil, response.ErrFileType
	}
	suf := strings.Split(in.Avatar.Filename, ".")[1]
	if _, ok := in.AcceptType[suf]; !ok {
		return nil, response.ErrFileType
	}
	path := fmt.Sprintf("%s%s.%s", in.ImagePath, in.ImageId, suf)

	// 打开图像
	src, err := in.Avatar.Open()
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
	if img.Bounds().Dx() != in.SizeWidth || img.Bounds().Dy() != in.SizeHeight {
		return nil, response.ErrImage
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

	return &parameter.UploadImageOut{
		FilePath: path,
	}, nil
}

func getFileID(path string, prefix string) string {
	var imgId string
	if path == "" {
		imgId = uuid.New().String()
	} else {
		imgId = strings.TrimPrefix(path, prefix)
		imgId = strings.Split(imgId, ".")[0]

		_, err := uuid.Parse(imgId)
		if err != nil {
			return ""
		}
	}
	return imgId
}

func (s *userService) GetStatByID(ctx context.Context, userID string) (*dto.UserStatResponse, error) {
	user, err := queryUserDetailWithCache(ctx, s.db, s.userCache, s.userRepo, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, response.ErrUserNotFound
	}

	userAsset, err := queryUserAssetDetailWithCache(ctx, s.db, s.userAssetCache, s.userAssetRepo, userID)
	if err != nil {
		logger.Error("query user asset detail fail", zap.Error(err))
		return nil, err
	}

	// 数据只存储在redis中
	userStatReadBooksNum, userStatReadBooksNumHit, err := s.userStatCache.GetReadBookNum(ctx, userID)
	if err != nil {
		return nil, err
	}
	if userStatReadBooksNumHit {
		logger.Info(fmt.Sprintf("userStat read book num not hit by userID: %s", userID))
	}

	userStatReadWordsNum, userStatReadWordNumHit, err := s.userStatCache.GetReadWordsNum(ctx, userID)
	if err != nil {
		return nil, err
	}
	if userStatReadWordNumHit {
		logger.Info(fmt.Sprintf("userStat read words num not hit by userID: %s", userID))
	}

	resp := &dto.UserStatResponse{
		UserResponse: *s.toUserResponse(user),

		Gold:        userAsset.Gold,
		VIPLevel:    VIPStrToLevel(userAsset.VIPLevel),
		VIPActiveAt: userAsset.VIPActiveAt,
		VIPMark:     userAsset.VIPLevel,

		ReadBooksNum: userStatReadBooksNum,
		ReadWordsNum: userStatReadWordsNum,
	}

	return resp, nil
}

func (s *userService) ForgetPassword(ctx context.Context, req *dto.ForgetPasswordRequest) error {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		userModel, err := s.userRepo.GetByEmail(ctx, tx, req.Email)
		if err != nil {
			return err
		}
		if userModel == nil {
			return response.ErrUserNotFound
		}

		err = s.emailVerify(ctx, req.Email, req.EmailVerify)
		if err != nil {
			return err
		}

		userModel.Password, err = hashPassword(req.Password)
		if err != nil {
			return err
		}

		err = s.userRepo.Update(ctx, tx, userModel)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}
	return nil
}
