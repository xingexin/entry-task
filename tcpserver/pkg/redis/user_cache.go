package redis

import (
	"context"
	"strconv"
	"time"

	"go.uber.org/zap"

	"entry-task/tcpserver/internal/model"
	log "entry-task/tcpserver/pkg/logger"
)

const (
	// UserCacheKeyPrefix 用户缓存键前缀
	// 缓存键设计示例：user:123
	UserCacheKeyPrefix = "user:"

	// UserCacheTTL 用户缓存过期时间（30分钟）
	UserCacheTTL = 30 * time.Minute

	// NullCacheValue 负缓存标记值
	NullCacheValue = "NULL"

	// NullCacheTTL 负缓存过期时间（5分钟）
	NullCacheTTL = 5 * time.Minute
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
// 正缓存键设计示例：user:123
// 负缓存键设计示例：user:null:123
func (uc *userCache) GetUser(ctx context.Context, userID uint64) (*CachedUser, error) {
	key := UserCacheKeyPrefix + strconv.FormatUint(userID, 10)

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
// 正缓存键设计示例：user:123
// 负缓存键设计示例：user:null:123
func (uc *userCache) SetUser(ctx context.Context, user *model.User) error {
	key := UserCacheKeyPrefix + strconv.FormatUint(user.ID, 10)

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
	key := UserCacheKeyPrefix + strconv.FormatUint(userID, 10)
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
	key := UserCacheKeyPrefix + strconv.FormatUint(userID, 10)
	err := uc.client.Del(ctx, key)
	if err != nil {
		log.Error("删除用户缓存失败", zap.Error(err), zap.Uint64("user_id", userID))
		return err
	}
	log.Debug("删除用户缓存成功", zap.Uint64("user_id", userID))
	return nil
}
