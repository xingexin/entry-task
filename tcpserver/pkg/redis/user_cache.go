package redis

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"entry-task/tcpserver/internal/model"
	log "entry-task/tcpserver/pkg/logger"
)

const (
	// UserCacheKeyPrefix 用户缓存键前缀
	UserCacheKeyPrefix = "user:"

	// UserCacheTTL 用户缓存过期时间（30分钟）
	UserCacheTTL = 30 * time.Minute

	// NullCacheValue 负缓存标记值
	NullCacheValue = "NULL"

	// NullCacheTTL 负缓存过期时间（5分钟）
	NullCacheTTL = 5 * time.Minute

	// DelayDeleteTime 延迟双删的延迟时间（500ms）
	DelayDeleteTime = 500 * time.Millisecond
)

// CachedUser 缓存的用户信息
type CachedUser struct {
	ID             uint64 `json:"id"`
	Username       string `json:"username"`
	Nickname       string `json:"nickname"`
	ProfilePicture string `json:"profile_picture"`
}

// UserCache 用户缓存管理器接口
type UserCache interface {
	// GetUser 获取用户缓存
	GetUser(ctx context.Context, userID uint64) (*CachedUser, error)

	// SetUser 设置用户缓存（TTL: 30分钟）
	SetUser(ctx context.Context, user *model.User) error

	// SetNullCache 设置负缓存（用户不存在时，TTL: 5分钟）
	SetNullCache(ctx context.Context, userID uint64) error

	// DeleteUser 删除用户缓存
	DeleteUser(ctx context.Context, userID uint64) error

	// DeleteUserWithDelay 延迟双删（先删除，500ms后再删除一次）
	DeleteUserWithDelay(ctx context.Context, userID uint64) error

	// GetOrLoad 获取缓存或从数据库加载（缓存穿透保护）
	GetOrLoad(
		ctx context.Context,
		userID uint64,
		loadFunc func(uint64) (*model.User, error),
	) (*model.User, error)
}

// userCache 用户缓存管理器实现
type userCache struct {
	client Client
}

// NewUserCache 创建用户缓存管理器
func NewUserCache(client Client) UserCache {
	return &userCache{client: client}
}

// GetUser 获取用户缓存
func (uc *userCache) GetUser(ctx context.Context, userID uint64) (*CachedUser, error) {
	key := fmt.Sprintf("%s%d", UserCacheKeyPrefix, userID)

	var user CachedUser
	err := uc.client.GetJSON(ctx, key, &user)
	if err != nil {
		// 使用redis.Nil判断键不存在
		if err.Error() == "redis: nil" {
			return nil, nil
		}
		return nil, err
	}

	// 检查是否是负缓存
	if user.Username == NullCacheValue {
		log.Debug("命中负缓存", zap.Uint64("user_id", userID))
		return nil, nil
	}

	log.Debug("命中用户缓存", zap.Uint64("user_id", userID))
	return &user, nil
}

// SetUser 设置用户缓存
func (uc *userCache) SetUser(ctx context.Context, user *model.User) error {
	key := fmt.Sprintf("%s%d", UserCacheKeyPrefix, user.ID)

	cachedUser := &CachedUser{
		ID:             user.ID,
		Username:       user.Username,
		Nickname:       user.Nickname,
		ProfilePicture: user.ProfilePicture,
	}

	err := uc.client.SetJSON(ctx, key, cachedUser, UserCacheTTL)
	if err != nil {
		log.Error("设置用户缓存失败", zap.Error(err), zap.Uint64("user_id", user.ID))
		return err
	}

	log.Debug("设置用户缓存成功", zap.Uint64("user_id", user.ID))
	return nil
}

// SetNullCache 设置负缓存
func (uc *userCache) SetNullCache(ctx context.Context, userID uint64) error {
	key := fmt.Sprintf("%s%d", UserCacheKeyPrefix, userID)
	nullUser := &CachedUser{Username: NullCacheValue}

	err := uc.client.SetJSON(ctx, key, nullUser, NullCacheTTL)
	if err != nil {
		log.Error("设置负缓存失败", zap.Error(err), zap.Uint64("user_id", userID))
		return err
	}

	log.Debug("设置负缓存成功", zap.Uint64("user_id", userID))
	return nil
}

// DeleteUser 删除用户缓存
func (uc *userCache) DeleteUser(ctx context.Context, userID uint64) error {
	key := fmt.Sprintf("%s%d", UserCacheKeyPrefix, userID)
	err := uc.client.Del(ctx, key)
	if err != nil {
		log.Error("删除用户缓存失败", zap.Error(err), zap.Uint64("user_id", userID))
		return err
	}
	log.Debug("删除用户缓存成功", zap.Uint64("user_id", userID))
	return nil
}

// DeleteUserWithDelay 延迟双删
func (uc *userCache) DeleteUserWithDelay(ctx context.Context, userID uint64) error {
	// 第一次删除
	if err := uc.DeleteUser(ctx, userID); err != nil {
		return err
	}

	// 异步延迟删除
	go func() {
		time.Sleep(DelayDeleteTime)
		delayCtx := context.Background()
		if err := uc.DeleteUser(delayCtx, userID); err != nil {
			log.Error("延迟删除缓存失败", zap.Error(err), zap.Uint64("user_id", userID))
		} else {
			log.Debug("延迟删除缓存成功", zap.Uint64("user_id", userID))
		}
	}()

	return nil
}

// GetOrLoad 获取缓存或从数据库加载（缓存穿透保护）
func (uc *userCache) GetOrLoad(
	ctx context.Context,
	userID uint64,
	loadFunc func(uint64) (*model.User, error),
) (*model.User, error) {
	// 1. 先查缓存
	cachedUser, err := uc.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 2. 缓存命中
	if cachedUser != nil {
		return &model.User{
			ID:             cachedUser.ID,
			Username:       cachedUser.Username,
			Nickname:       cachedUser.Nickname,
			ProfilePicture: cachedUser.ProfilePicture,
		}, nil
	}

	// 3. 从数据库加载
	user, err := loadFunc(userID)
	if err != nil {
		return nil, err
	}

	// 4. 用户不存在，设置负缓存（防止缓存穿透）
	if user == nil {
		if err := uc.SetNullCache(ctx, userID); err != nil {
			log.Error("设置负缓存失败（GetOrLoad）",
				zap.Error(err),
				zap.Uint64("user_id", userID))
			// 不返回错误，允许继续返回nil
		}
		return nil, nil
	}

	// 5. 用户存在，设置缓存
	if err := uc.SetUser(ctx, user); err != nil {
		log.Error("设置用户缓存失败（GetOrLoad）",
			zap.Error(err),
			zap.Uint64("user_id", userID))
		// 不返回错误，允许继续返回user数据（缓存失败不影响数据返回）
	}
	return user, nil
}
