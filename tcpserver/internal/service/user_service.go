package service

import (
	"context"
	"entry-task/tcpserver/internal/dto"
	"entry-task/tcpserver/internal/repository"
	"entry-task/tcpserver/pkg/redis"
	"errors"
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	log "entry-task/tcpserver/pkg/logger"
)

// ============================================================================
// 业务错误定义
// ============================================================================

var (
	ErrInvalidCredentials  = errors.New("用户名或密码错误")
	ErrUserNotFound        = errors.New("用户不存在")
	ErrPasswordHashFailed  = errors.New("密码哈希失败")
	ErrSessionCreateFailed = errors.New("创建会话失败")
	ErrInvalidToken        = errors.New("无效的Token")
	ErrLoginLimitExceeded  = errors.New("登录失败次数过多，请稍后再试")
)

const (
	// 登录失败次数限制
	MaxLoginFailures = 5
)

// ============================================================================
// UserService 接口
// ============================================================================

type UserService interface {
	// Login 用户登录
	Login(ctx context.Context, loginDTO *dto.LoginDTO) (*dto.LoginResultDTO, error)

	// Logout 用户登出
	Logout(ctx context.Context, logoutDTO *dto.LogoutDTO) error

	// GetProfile 获取用户信息（通过Token）
	GetProfile(ctx context.Context, validateDTO *dto.ValidateTokenDTO) (*dto.UserProfileDTO, error)

	// UpdateNickname 更新用户昵称
	UpdateNickname(ctx context.Context, updateDTO *dto.UpdateNicknameDTO) (*dto.UserProfileDTO, error)

	// UpdateProfilePicture 更新用户头像URL
	UpdateProfilePicture(ctx context.Context, updateDTO *dto.UpdateProfilePictureDTO) (*dto.UserProfileDTO, error)
}

// ============================================================================
// userService 实现
// ============================================================================

type userService struct {
	userRepo     repository.UserRepository
	redisManager redis.Manager
}

// NewUserService 创建UserService实例
func NewUserService(userRepo repository.UserRepository, redisManager redis.Manager) UserService {
	return &userService{
		userRepo:     userRepo,
		redisManager: redisManager,
	}
}

// ============================================================================
// Login 登录
// ============================================================================

func (s *userService) Login(ctx context.Context, loginDTO *dto.LoginDTO) (*dto.LoginResultDTO, error) {
	// 1. 验证DTO
	if err := loginDTO.Validate(); err != nil {
		log.Warn("登录参数验证失败", zap.Error(err), zap.String("username", loginDTO.Username))
		return nil, err
	}

	// 2. 检查登录失败次数限制
	failCount, err := s.redisManager.GetLoginLimiter().GetLoginFailCount(ctx, loginDTO.Username)
	if err != nil {
		log.Error("获取登录失败次数失败", zap.Error(err), zap.String("username", loginDTO.Username))
		// 降级策略：失败不影响登录流程
	}
	if failCount >= MaxLoginFailures {
		log.Warn("登录失败次数过多",
			zap.String("username", loginDTO.Username),
			zap.Int64("fail_count", failCount))
		return nil, ErrLoginLimitExceeded
	}

	// 3. 查询用户（从Repository获取，包含password_hash）
	user, err := s.userRepo.GetByUsername(ctx, loginDTO.Username)
	if err != nil {
		log.Warn("用户不存在", zap.String("username", loginDTO.Username))
		// 记录登录失败
		if _, recordErr := s.redisManager.GetLoginLimiter().RecordLoginFail(ctx, loginDTO.Username); recordErr != nil {
			log.Error("记录登录失败次数失败", zap.Error(recordErr))
		}
		return nil, ErrInvalidCredentials
	}

	// 4. 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(loginDTO.Password)); err != nil {
		log.Warn("密码错误",
			zap.String("username", loginDTO.Username),
			zap.Error(err))
		// 记录登录失败
		if _, recordErr := s.redisManager.GetLoginLimiter().RecordLoginFail(ctx, loginDTO.Username); recordErr != nil {
			log.Error("记录登录失败次数失败", zap.Error(recordErr))
		}
		return nil, ErrInvalidCredentials
	}

	// 5. 创建Session
	token, err := s.redisManager.GetSession().CreateSession(ctx, user.ID)
	if err != nil {
		log.Error("创建Session失败", zap.Error(err), zap.Uint64("user_id", user.ID))
		return nil, ErrSessionCreateFailed
	}

	// 6. 清空登录失败次数
	if err := s.redisManager.GetLoginLimiter().ResetLoginFail(ctx, loginDTO.Username); err != nil {
		log.Error("重置登录失败次数失败", zap.Error(err))
		// 不影响主流程
	}

	// 7. 转换为DTO并返回
	userDTO := dto.FromModel(user)
	log.Info("用户登录成功",
		zap.String("username", loginDTO.Username),
		zap.Uint64("user_id", user.ID))

	return &dto.LoginResultDTO{
		Token:   token,
		Profile: userDTO.ToProfile(),
	}, nil
}

// ============================================================================
// Logout 登出
// ============================================================================

func (s *userService) Logout(ctx context.Context, logoutDTO *dto.LogoutDTO) error {
	// 1. 验证DTO
	if err := logoutDTO.Validate(); err != nil {
		return err
	}

	// 2. 销毁Session
	if err := s.redisManager.GetSession().DestroySession(ctx, logoutDTO.Token); err != nil {
		log.Error("销毁Session失败", zap.Error(err), zap.String("token", logoutDTO.Token))
		return fmt.Errorf("登出失败: %w", err)
	}

	log.Info("用户登出成功", zap.String("token", logoutDTO.Token))
	return nil
}

// ============================================================================
// GetProfile 获取用户信息
// ============================================================================

func (s *userService) GetProfile(ctx context.Context, validateDTO *dto.ValidateTokenDTO) (*dto.UserProfileDTO, error) {
	// 1. 验证DTO
	if err := validateDTO.Validate(); err != nil {
		return nil, err
	}

	// 2. 验证Token，获取UserID
	userID, err := s.redisManager.GetSession().ValidateSession(ctx, validateDTO.Token)
	if err != nil {
		log.Warn("Token验证失败", zap.Error(err), zap.String("token", validateDTO.Token))
		return nil, ErrInvalidToken
	}

	// 3. 从Repository获取用户信息（优先缓存，返回 CachedUser）
	cachedUser, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		log.Error("获取用户信息失败", zap.Error(err), zap.Uint64("user_id", userID))
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	if cachedUser == nil {
		log.Warn("用户不存在", zap.Uint64("user_id", userID))
		return nil, ErrUserNotFound
	}

	// 4. 转换为DTO
	profileDTO := dto.FromCachedUser(cachedUser)
	log.Debug("获取用户信息成功", zap.Uint64("user_id", userID))

	return profileDTO, nil
}

// ============================================================================
// UpdateNickname 更新昵称
// ============================================================================

func (s *userService) UpdateNickname(ctx context.Context, updateDTO *dto.UpdateNicknameDTO) (*dto.UserProfileDTO, error) {
	// 1. 验证DTO
	if err := updateDTO.Validate(); err != nil {
		log.Warn("更新昵称参数验证失败",
			zap.Error(err),
			zap.Uint64("user_id", updateDTO.UserID),
			zap.String("nickname", updateDTO.Nickname))
		return nil, err
	}

	// 2. 调用Repository更新（自动处理缓存）
	if err := s.userRepo.UpdateNickname(ctx, updateDTO.UserID, updateDTO.Nickname); err != nil {
		log.Error("更新昵称失败",
			zap.Error(err),
			zap.Uint64("user_id", updateDTO.UserID),
			zap.String("nickname", updateDTO.Nickname))
		return nil, fmt.Errorf("更新昵称失败: %w", err)
	}

	// 3. 重新查询用户信息（从缓存或数据库）
	cachedUser, err := s.userRepo.GetByID(ctx, updateDTO.UserID)
	if err != nil {
		log.Error("更新后查询用户信息失败", zap.Error(err), zap.Uint64("user_id", updateDTO.UserID))
		return nil, fmt.Errorf("更新后查询用户信息失败: %w", err)
	}

	if cachedUser == nil {
		log.Warn("更新后用户不存在", zap.Uint64("user_id", updateDTO.UserID))
		return nil, ErrUserNotFound
	}

	// 4. 转换为DTO并返回
	profileDTO := dto.FromCachedUser(cachedUser)
	log.Info("更新昵称成功",
		zap.Uint64("user_id", updateDTO.UserID),
		zap.String("nickname", updateDTO.Nickname))

	return profileDTO, nil
}

// ============================================================================
// UpdateProfilePicture 更新头像URL
// ============================================================================

func (s *userService) UpdateProfilePicture(ctx context.Context, updateDTO *dto.UpdateProfilePictureDTO) (*dto.UserProfileDTO, error) {
	// 1. 验证DTO
	if err := updateDTO.Validate(); err != nil {
		log.Warn("更新头像参数验证失败",
			zap.Error(err),
			zap.Uint64("user_id", updateDTO.UserID),
			zap.String("profile_picture", updateDTO.ProfilePicture))
		return nil, err
	}

	// 2. 调用Repository更新（自动处理缓存）
	if err := s.userRepo.UpdateProfilePicture(ctx, updateDTO.UserID, updateDTO.ProfilePicture); err != nil {
		log.Error("更新头像失败",
			zap.Error(err),
			zap.Uint64("user_id", updateDTO.UserID),
			zap.String("profile_picture", updateDTO.ProfilePicture))
		return nil, fmt.Errorf("更新头像失败: %w", err)
	}

	// 3. 重新查询用户信息（从缓存或数据库）
	cachedUser, err := s.userRepo.GetByID(ctx, updateDTO.UserID)
	if err != nil {
		log.Error("更新后查询用户信息失败", zap.Error(err), zap.Uint64("user_id", updateDTO.UserID))
		return nil, fmt.Errorf("更新后查询用户信息失败: %w", err)
	}

	if cachedUser == nil {
		log.Warn("更新后用户不存在", zap.Uint64("user_id", updateDTO.UserID))
		return nil, ErrUserNotFound
	}

	// 4. 转换为DTO并返回
	profileDTO := dto.FromCachedUser(cachedUser)
	log.Info("更新头像成功",
		zap.Uint64("user_id", updateDTO.UserID),
		zap.String("profile_picture", updateDTO.ProfilePicture))

	return profileDTO, nil
}
