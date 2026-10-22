package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"fiction-book/internal/api/middleware"
	"fiction-book/internal/dto"
	"fiction-book/internal/service"
	"fiction-book/pkg/logger"
	"fiction-book/pkg/response"
	customValidator "fiction-book/pkg/validator"
)

func init() {
	customValidator.Init()
}

// TestMain 初始化 logger，response.PushError 等会直接使用全局 logger
func TestMain(m *testing.M) {
	if err := logger.Init("debug"); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

// MockUserService 模拟用户服务
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) Create(ctx context.Context, req *dto.CreateUserRequest) (*dto.UserResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	if v, ok := args.Get(0).(*dto.UserResponse); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserService) GetByID(ctx context.Context, id string) (*dto.UserResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	if v, ok := args.Get(0).(*dto.UserResponse); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserService) Update(ctx context.Context, id string, req *dto.UpdateUserRequest) (*dto.UserResponse, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	if v, ok := args.Get(0).(*dto.UserResponse); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserService) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserService) Login(ctx context.Context, req *dto.LoginRequest) (*dto.LoginResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	if v, ok := args.Get(0).(*dto.LoginResponse); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserService) RefreshToken(ctx context.Context, req *dto.RefreshTokenRequest) (*dto.RefreshTokenResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	if v, ok := args.Get(0).(*dto.RefreshTokenResponse); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserService) List(ctx context.Context, page, pageSize int) ([]*dto.UserResponse, error) {
	args := m.Called(ctx, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	if v, ok := args.Get(0).([]*dto.UserResponse); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserService) UpdatePassword(ctx context.Context, id string, req *dto.UpdateUserPasswordRequest) (*dto.UserResponse, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	if v, ok := args.Get(0).(*dto.UserResponse); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserService) UpdateEmail(ctx context.Context, id string, req *dto.UpdateUserEmailRequest) (*dto.UserResponse, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	if v, ok := args.Get(0).(*dto.UserResponse); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserService) SendEmailVerification(ctx context.Context, receiver string) error {
	args := m.Called(ctx, receiver)
	return args.Error(0)
}

func (m *MockUserService) UpdateAvatar(ctx context.Context, userID string, avatar *multipart.FileHeader) (*dto.UserResponse, error) {
	args := m.Called(ctx, userID, avatar)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	if v, ok := args.Get(0).(*dto.UserResponse); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserService) GetStatByID(ctx context.Context, userID string) (*dto.UserStatResponse, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	if v, ok := args.Get(0).(*dto.UserStatResponse); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserService) ForgetPassword(ctx context.Context, req *dto.ForgetPasswordRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

// testUserHandler 用于测试的处理器，仅包含用户服务
type testUserHandler struct {
	userService service.UserService
}

func (h *testUserHandler) GetUser(c *gin.Context) {
	req := middleware.MustGetRequest[dto.GetUserRequest](c)

	user, err := h.userService.GetByID(c.Request.Context(), req.ID)
	if err != nil {
		response.PushError(c, err)
		return
	}
	response.Success(c, user)
}

func (h *testUserHandler) CreateUser(c *gin.Context) {
	req := middleware.MustGetRequest[dto.CreateUserRequest](c)

	user, err := h.userService.Create(c.Request.Context(), req)
	if err != nil {
		response.PushError(c, err)
		return
	}
	response.Success(c, user)
}

func TestGetUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockUserService)
	handler := &testUserHandler{userService: mockService}

	expectedUser := &dto.UserResponse{
		ID:       "1",
		Username: "testuser",
		Email:    "test@example.com",
	}

	mockService.On("GetByID", mock.Anything, "1").Return(expectedUser, nil)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/users/:id", middleware.Validation(&dto.GetUserRequest{}), handler.GetUser)

	req, err := http.NewRequestWithContext(context.Background(), "GET", "/users/1", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestCreateUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockUserService)
	handler := &testUserHandler{userService: mockService}

	createReq := &dto.CreateUserRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123", // pragma: allowlist secret
	}

	expectedUser := &dto.UserResponse{
		ID:       "1",
		Username: "testuser",
		Email:    "test@example.com",
	}

	mockService.On("Create", mock.Anything, mock.MatchedBy(func(req *dto.CreateUserRequest) bool {
		return req.Username == createReq.Username &&
			req.Email == createReq.Email &&
			req.Password == createReq.Password //nolint:gosec // pragma: allowlist secret
	})).Return(expectedUser, nil)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.POST("/users", middleware.Validation(&dto.CreateUserRequest{}), handler.CreateUser)

	jsonData, err := json.Marshal(createReq)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}
	req, err := http.NewRequestWithContext(context.Background(), "POST", "/users", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestGetUserNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockUserService)
	handler := &testUserHandler{userService: mockService}

	mockService.On("GetByID", mock.Anything, "999").Return(nil, response.ErrUserNotFound)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/users/:id", middleware.Validation(&dto.GetUserRequest{}), handler.GetUser)

	req, err := http.NewRequestWithContext(context.Background(), "GET", "/users/999", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, float64(4002), resp["code"])

	mockService.AssertExpectations(t)
}
