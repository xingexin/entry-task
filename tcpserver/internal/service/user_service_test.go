package service

import (
	"context"
	"errors"
	"testing"

	"entry-task/tcpserver/internal/dto"
	"entry-task/tcpserver/internal/model"
	"entry-task/tcpserver/pkg/logger"
	"entry-task/tcpserver/pkg/redis"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

// ============================================================================
// 测试初始化
// ============================================================================

// TestMain 在所有测试运行前初始化
func TestMain(m *testing.M) {
	// 初始化日志（测试环境使用 Fatal 级别，只显示严重错误）
	cfg := &logger.Config{
		Level:  "fatal", // 只显示 Fatal 级别日志，测试中的 Error 日志不会显示
		Output: "stdout",
	}
	if err := logger.Init(cfg); err != nil {
		panic("初始化日志失败: " + err.Error())
	}

	// 运行测试
	m.Run()
}

// ============================================================================
// Mock 定义
// ============================================================================

// MockUserRepository 模拟 UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uint64) (*redis.CachedUser, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*redis.CachedUser), args.Error(1)
}

func (m *MockUserRepository) Create(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateNickname(ctx context.Context, id uint64, nickname string) error {
	args := m.Called(ctx, id, nickname)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateProfilePicture(ctx context.Context, id uint64, profilePicture string) error {
	args := m.Called(ctx, id, profilePicture)
	return args.Error(0)
}

func (m *MockUserRepository) BatchCreate(ctx context.Context, users []*model.User) error {
	args := m.Called(ctx, users)
	return args.Error(0)
}

// MockSessionManager 模拟 SessionManager
type MockSessionManager struct {
	mock.Mock
}

func (m *MockSessionManager) CreateSession(ctx context.Context, userID uint64) (string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.Error(1)
}

func (m *MockSessionManager) ValidateSession(ctx context.Context, token string) (uint64, error) {
	args := m.Called(ctx, token)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockSessionManager) DestroySession(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockSessionManager) RefreshSession(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

// MockLoginLimiter 模拟 LoginLimiter
type MockLoginLimiter struct {
	mock.Mock
}

func (m *MockLoginLimiter) RecordLoginFail(ctx context.Context, username string) (int64, error) {
	args := m.Called(ctx, username)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockLoginLimiter) GetLoginFailCount(ctx context.Context, username string) (int64, error) {
	args := m.Called(ctx, username)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockLoginLimiter) IsLoginAllowed(ctx context.Context, username string) (bool, error) {
	args := m.Called(ctx, username)
	return args.Bool(0), args.Error(1)
}

func (m *MockLoginLimiter) ResetLoginFail(ctx context.Context, username string) error {
	args := m.Called(ctx, username)
	return args.Error(0)
}

// MockUserCache 模拟 UserCache
type MockUserCache struct {
	mock.Mock
}

func (m *MockUserCache) GetUser(ctx context.Context, id uint64) (*redis.CachedUser, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*redis.CachedUser), args.Error(1)
}

func (m *MockUserCache) SetUser(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserCache) DeleteUser(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserCache) SetNullCache(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockRedisManager 模拟 RedisManager
type MockRedisManager struct {
	mock.Mock
	session      *MockSessionManager
	loginLimiter *MockLoginLimiter
	userCache    *MockUserCache
}

func NewMockRedisManager() *MockRedisManager {
	return &MockRedisManager{
		session:      &MockSessionManager{},
		loginLimiter: &MockLoginLimiter{},
		userCache:    &MockUserCache{},
	}
}

func (m *MockRedisManager) GetClient() redis.Client {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(redis.Client)
}

func (m *MockRedisManager) GetSession() redis.SessionManager {
	return m.session
}

func (m *MockRedisManager) GetLoginLimiter() redis.LoginLimiter {
	return m.loginLimiter
}

func (m *MockRedisManager) GetUserCache() redis.UserCache {
	return m.userCache
}

// ============================================================================
// 测试辅助函数
// ============================================================================

func setupTestService() (*userService, *MockUserRepository, *MockRedisManager) {
	mockRepo := new(MockUserRepository)
	mockRedis := NewMockRedisManager()

	service := &userService{
		userRepo:     mockRepo,
		redisManager: mockRedis,
	}

	return service, mockRepo, mockRedis
}

// hashPassword 生成密码哈希（测试辅助函数）
func hashPassword(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	return string(hash)
}

// ============================================================================
// Login 测试
// ============================================================================

func TestLogin_Success(t *testing.T) {
	service, mockRepo, mockRedis := setupTestService()
	ctx := context.Background()

	// 准备测试数据
	username := "testuser"
	password := "Test@123"
	userID := uint64(123456)
	passwordHash := hashPassword(password)
	token := "test-token-123"

	loginDTO := &dto.LoginDTO{
		Username: username,
		Password: password,
	}

	mockUser := &model.User{
		ID:           userID,
		Username:     username,
		PasswordHash: passwordHash,
		Nickname:     "测试用户",
	}

	// 设置 Mock 期望
	mockRedis.loginLimiter.On("GetLoginFailCount", ctx, username).Return(int64(0), nil)
	mockRepo.On("GetByUsername", ctx, username).Return(mockUser, nil)
	mockRedis.session.On("CreateSession", ctx, userID).Return(token, nil)
	mockRedis.loginLimiter.On("ResetLoginFail", ctx, username).Return(nil)

	// 执行测试
	result, err := service.Login(ctx, loginDTO)

	// 断言
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, token, result.Token)
	assert.Equal(t, username, result.Profile.Username)
	assert.Equal(t, "测试用户", result.Profile.Nickname)

	// 验证 Mock 调用
	mockRepo.AssertExpectations(t)
	mockRedis.loginLimiter.AssertExpectations(t)
	mockRedis.session.AssertExpectations(t)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	service, mockRepo, mockRedis := setupTestService()
	ctx := context.Background()

	username := "testuser"
	password := "WrongPassword"

	loginDTO := &dto.LoginDTO{
		Username: username,
		Password: password,
	}

	// 设置 Mock 期望 - 用户不存在
	mockRedis.loginLimiter.On("GetLoginFailCount", ctx, username).Return(int64(0), nil)
	mockRepo.On("GetByUsername", ctx, username).Return(nil, errors.New("user not found"))
	mockRedis.loginLimiter.On("RecordLoginFail", ctx, username).Return(int64(1), nil)

	// 执行测试
	result, err := service.Login(ctx, loginDTO)

	// 断言
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrInvalidCredentials, err)

	mockRepo.AssertExpectations(t)
	mockRedis.loginLimiter.AssertExpectations(t)
}

func TestLogin_WrongPassword(t *testing.T) {
	service, mockRepo, mockRedis := setupTestService()
	ctx := context.Background()

	username := "testuser"
	correctPassword := "Test@123"
	wrongPassword := "WrongPass"
	passwordHash := hashPassword(correctPassword)

	loginDTO := &dto.LoginDTO{
		Username: username,
		Password: wrongPassword,
	}

	mockUser := &model.User{
		ID:           123456,
		Username:     username,
		PasswordHash: passwordHash,
	}

	// 设置 Mock 期望
	mockRedis.loginLimiter.On("GetLoginFailCount", ctx, username).Return(int64(0), nil)
	mockRepo.On("GetByUsername", ctx, username).Return(mockUser, nil)
	mockRedis.loginLimiter.On("RecordLoginFail", ctx, username).Return(int64(1), nil)

	// 执行测试
	result, err := service.Login(ctx, loginDTO)

	// 断言
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrInvalidCredentials, err)

	mockRepo.AssertExpectations(t)
	mockRedis.loginLimiter.AssertExpectations(t)
}

func TestLogin_ExceedMaxAttempts(t *testing.T) {
	service, _, mockRedis := setupTestService()
	ctx := context.Background()

	username := "testuser"
	loginDTO := &dto.LoginDTO{
		Username: username,
		Password: "Test@123",
	}

	// 设置 Mock 期望 - 登录失败次数已达上限
	mockRedis.loginLimiter.On("GetLoginFailCount", ctx, username).Return(int64(5), nil)

	// 执行测试
	result, err := service.Login(ctx, loginDTO)

	// 断言
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrLoginLimitExceeded, err)

	mockRedis.loginLimiter.AssertExpectations(t)
}

func TestLogin_SessionCreateFailed(t *testing.T) {
	service, mockRepo, mockRedis := setupTestService()
	ctx := context.Background()

	username := "testuser"
	password := "Test@123"
	userID := uint64(123456)
	passwordHash := hashPassword(password)

	loginDTO := &dto.LoginDTO{
		Username: username,
		Password: password,
	}

	mockUser := &model.User{
		ID:           userID,
		Username:     username,
		PasswordHash: passwordHash,
	}

	// 设置 Mock 期望
	mockRedis.loginLimiter.On("GetLoginFailCount", ctx, username).Return(int64(0), nil)
	mockRepo.On("GetByUsername", ctx, username).Return(mockUser, nil)
	mockRedis.session.On("CreateSession", ctx, userID).Return("", errors.New("redis error"))

	// 执行测试
	result, err := service.Login(ctx, loginDTO)

	// 断言
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrSessionCreateFailed, err)

	mockRepo.AssertExpectations(t)
	mockRedis.session.AssertExpectations(t)
}

// ============================================================================
// Logout 测试
// ============================================================================

func TestLogout_Success(t *testing.T) {
	service, _, mockRedis := setupTestService()
	ctx := context.Background()

	token := "test-token-123"
	logoutDTO := &dto.LogoutDTO{
		Token: token,
	}

	// 设置 Mock 期望
	mockRedis.session.On("DestroySession", ctx, token).Return(nil)

	// 执行测试
	err := service.Logout(ctx, logoutDTO)

	// 断言
	assert.NoError(t, err)
	mockRedis.session.AssertExpectations(t)
}

func TestLogout_InvalidToken(t *testing.T) {
	service, _, mockRedis := setupTestService()
	ctx := context.Background()

	token := "invalid-token"
	logoutDTO := &dto.LogoutDTO{
		Token: token,
	}

	// 设置 Mock 期望
	mockRedis.session.On("DestroySession", ctx, token).Return(errors.New("token not found"))

	// 执行测试
	err := service.Logout(ctx, logoutDTO)

	// 断言
	assert.Error(t, err)
	mockRedis.session.AssertExpectations(t)
}

// ============================================================================
// GetProfile 测试
// ============================================================================

func TestGetProfile_Success(t *testing.T) {
	service, mockRepo, mockRedis := setupTestService()
	ctx := context.Background()

	token := "test-token-123"
	userID := uint64(123456)

	validateDTO := &dto.ValidateTokenDTO{
		Token: token,
	}

	cachedUser := &redis.CachedUser{
		ID:             userID,
		Username:       "testuser",
		Nickname:       "测试用户",
		ProfilePicture: "/avatar.png",
	}

	// 设置 Mock 期望
	mockRedis.session.On("ValidateSession", ctx, token).Return(userID, nil)
	mockRepo.On("GetByID", ctx, userID).Return(cachedUser, nil)

	// 执行测试
	profile, err := service.GetProfile(ctx, validateDTO)

	// 断言
	assert.NoError(t, err)
	assert.NotNil(t, profile)
	assert.Equal(t, userID, profile.ID)
	assert.Equal(t, "testuser", profile.Username)
	assert.Equal(t, "测试用户", profile.Nickname)

	mockRedis.session.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestGetProfile_InvalidToken(t *testing.T) {
	service, _, mockRedis := setupTestService()
	ctx := context.Background()

	token := "invalid-token"
	validateDTO := &dto.ValidateTokenDTO{
		Token: token,
	}

	// 设置 Mock 期望
	mockRedis.session.On("ValidateSession", ctx, token).Return(uint64(0), errors.New("invalid token"))

	// 执行测试
	profile, err := service.GetProfile(ctx, validateDTO)

	// 断言
	assert.Error(t, err)
	assert.Nil(t, profile)
	assert.Equal(t, ErrInvalidToken, err)

	mockRedis.session.AssertExpectations(t)
}

func TestGetProfile_UserNotFound(t *testing.T) {
	service, mockRepo, mockRedis := setupTestService()
	ctx := context.Background()

	token := "test-token-123"
	userID := uint64(123456)

	validateDTO := &dto.ValidateTokenDTO{
		Token: token,
	}

	// 设置 Mock 期望
	mockRedis.session.On("ValidateSession", ctx, token).Return(userID, nil)
	mockRepo.On("GetByID", ctx, userID).Return(nil, nil) // 用户不存在

	// 执行测试
	profile, err := service.GetProfile(ctx, validateDTO)

	// 断言
	assert.Error(t, err)
	assert.Nil(t, profile)
	assert.Equal(t, ErrUserNotFound, err)

	mockRedis.session.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// UpdateNickname 测试
// ============================================================================

func TestUpdateNickname_Success(t *testing.T) {
	service, mockRepo, _ := setupTestService()
	ctx := context.Background()

	userID := uint64(123456)
	newNickname := "新昵称"

	updateDTO := &dto.UpdateNicknameDTO{
		UserID:   userID,
		Nickname: newNickname,
	}

	updatedUser := &redis.CachedUser{
		ID:       userID,
		Username: "testuser",
		Nickname: newNickname,
	}

	// 设置 Mock 期望
	mockRepo.On("UpdateNickname", ctx, userID, newNickname).Return(nil)
	mockRepo.On("GetByID", ctx, userID).Return(updatedUser, nil)

	// 执行测试
	profile, err := service.UpdateNickname(ctx, updateDTO)

	// 断言
	assert.NoError(t, err)
	assert.NotNil(t, profile)
	assert.Equal(t, newNickname, profile.Nickname)

	mockRepo.AssertExpectations(t)
}

func TestUpdateNickname_UpdateFailed(t *testing.T) {
	service, mockRepo, _ := setupTestService()
	ctx := context.Background()

	userID := uint64(123456)
	newNickname := "新昵称"

	updateDTO := &dto.UpdateNicknameDTO{
		UserID:   userID,
		Nickname: newNickname,
	}

	// 设置 Mock 期望 - 更新失败
	mockRepo.On("UpdateNickname", ctx, userID, newNickname).Return(errors.New("update failed"))

	// 执行测试
	profile, err := service.UpdateNickname(ctx, updateDTO)

	// 断言
	assert.Error(t, err)
	assert.Nil(t, profile)

	mockRepo.AssertExpectations(t)
}

func TestUpdateNickname_UserNotFoundAfterUpdate(t *testing.T) {
	service, mockRepo, _ := setupTestService()
	ctx := context.Background()

	userID := uint64(123456)
	newNickname := "新昵称"

	updateDTO := &dto.UpdateNicknameDTO{
		UserID:   userID,
		Nickname: newNickname,
	}

	// 设置 Mock 期望
	mockRepo.On("UpdateNickname", ctx, userID, newNickname).Return(nil)
	mockRepo.On("GetByID", ctx, userID).Return(nil, nil) // 更新后查询不到用户

	// 执行测试
	profile, err := service.UpdateNickname(ctx, updateDTO)

	// 断言
	assert.Error(t, err)
	assert.Nil(t, profile)
	assert.Equal(t, ErrUserNotFound, err)

	mockRepo.AssertExpectations(t)
}

// ============================================================================
// UpdateProfilePicture 测试
// ============================================================================

func TestUpdateProfilePicture_Success(t *testing.T) {
	service, mockRepo, _ := setupTestService()
	ctx := context.Background()

	userID := uint64(123456)
	newPicture := "/uploads/avatars/123456.png"

	updateDTO := &dto.UpdateProfilePictureDTO{
		UserID:         userID,
		ProfilePicture: newPicture,
	}

	updatedUser := &redis.CachedUser{
		ID:             userID,
		Username:       "testuser",
		ProfilePicture: newPicture,
	}

	// 设置 Mock 期望
	mockRepo.On("UpdateProfilePicture", ctx, userID, newPicture).Return(nil)
	mockRepo.On("GetByID", ctx, userID).Return(updatedUser, nil)

	// 执行测试
	profile, err := service.UpdateProfilePicture(ctx, updateDTO)

	// 断言
	assert.NoError(t, err)
	assert.NotNil(t, profile)
	assert.Equal(t, newPicture, profile.ProfilePicture)

	mockRepo.AssertExpectations(t)
}

func TestUpdateProfilePicture_UpdateFailed(t *testing.T) {
	service, mockRepo, _ := setupTestService()
	ctx := context.Background()

	userID := uint64(123456)
	newPicture := "/uploads/avatars/123456.png"

	updateDTO := &dto.UpdateProfilePictureDTO{
		UserID:         userID,
		ProfilePicture: newPicture,
	}

	// 设置 Mock 期望 - 更新失败
	mockRepo.On("UpdateProfilePicture", ctx, userID, newPicture).Return(errors.New("update failed"))

	// 执行测试
	profile, err := service.UpdateProfilePicture(ctx, updateDTO)

	// 断言
	assert.Error(t, err)
	assert.Nil(t, profile)

	mockRepo.AssertExpectations(t)
}

func TestUpdateProfilePicture_UserNotFoundAfterUpdate(t *testing.T) {
	service, mockRepo, _ := setupTestService()
	ctx := context.Background()

	userID := uint64(123456)
	newPicture := "/uploads/avatars/123456.png"

	updateDTO := &dto.UpdateProfilePictureDTO{
		UserID:         userID,
		ProfilePicture: newPicture,
	}

	// 设置 Mock 期望
	mockRepo.On("UpdateProfilePicture", ctx, userID, newPicture).Return(nil)
	mockRepo.On("GetByID", ctx, userID).Return(nil, nil) // 更新后查询不到用户

	// 执行测试
	profile, err := service.UpdateProfilePicture(ctx, updateDTO)

	// 断言
	assert.Error(t, err)
	assert.Nil(t, profile)
	assert.Equal(t, ErrUserNotFound, err)

	mockRepo.AssertExpectations(t)
}

// ============================================================================
// DTO 验证测试
// ============================================================================

func TestLogin_InvalidDTO_EmptyUsername(t *testing.T) {
	service, _, _ := setupTestService()
	ctx := context.Background()

	loginDTO := &dto.LoginDTO{
		Username: "",
		Password: "Test@123",
	}

	// 执行测试
	result, err := service.Login(ctx, loginDTO)

	// 断言
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestLogin_InvalidDTO_EmptyPassword(t *testing.T) {
	service, _, _ := setupTestService()
	ctx := context.Background()

	loginDTO := &dto.LoginDTO{
		Username: "testuser",
		Password: "",
	}

	// 执行测试
	result, err := service.Login(ctx, loginDTO)

	// 断言
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestLogout_InvalidDTO_EmptyToken(t *testing.T) {
	service, _, _ := setupTestService()
	ctx := context.Background()

	logoutDTO := &dto.LogoutDTO{
		Token: "",
	}

	// 执行测试
	err := service.Logout(ctx, logoutDTO)

	// 断言
	assert.Error(t, err)
}

func TestUpdateNickname_InvalidDTO_EmptyNickname(t *testing.T) {
	service, _, _ := setupTestService()
	ctx := context.Background()

	updateDTO := &dto.UpdateNicknameDTO{
		UserID:   123456,
		Nickname: "",
	}

	// 执行测试
	profile, err := service.UpdateNickname(ctx, updateDTO)

	// 断言
	assert.Error(t, err)
	assert.Nil(t, profile)
}
