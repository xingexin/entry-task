package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"go.uber.org/zap"

	log "entry-task/tcpserver/pkg/logger"
)

const (
	// LoginFailKeyPrefix 登录失败计数键前缀
	LoginFailKeyPrefix = "login_fail:"

	// LoginFailTTL 登录失败计数过期时间（15分钟）
	LoginFailTTL = 15 * time.Minute

	// MaxLoginAttempts 最大登录尝试次数
	MaxLoginAttempts = 5
)

// LoginLimiter 登录限制器接口
type LoginLimiter interface {
	// RecordLoginFail 记录登录失败（计数器+1）
	RecordLoginFail(ctx context.Context, username string) (int64, error)

	// GetLoginFailCount 获取登录失败次数
	GetLoginFailCount(ctx context.Context, username string) (int64, error)

	// IsLoginAllowed 检查是否允许登录（失败次数<5）
	IsLoginAllowed(ctx context.Context, username string) (bool, error)

	// ResetLoginFail 重置登录失败计数（登录成功后调用）
	ResetLoginFail(ctx context.Context, username string) error
}

// loginLimiter 登录限制器实现
type loginLimiter struct {
	client Client
}

// NewLoginLimiter 创建登录限制器
func NewLoginLimiter(client Client) LoginLimiter {
	return &loginLimiter{client: client}
}

// RecordLoginFail 记录登录失败
// 登录失败key设计: login_fail:123123
func (ll *loginLimiter) RecordLoginFail(ctx context.Context, username string) (int64, error) {
	key := LoginFailKeyPrefix + username

	count, err := ll.client.Incr(ctx, key)
	if err != nil {
		log.Error("记录登录失败次数失败", zap.Error(err), zap.String("username", username))
		return 0, err
	}

	if count == 1 {
		if err := ll.client.Expire(ctx, key, LoginFailTTL); err != nil {
			log.Error("设置登录失败计数过期时间失败",
				zap.Error(err),
				zap.String("username", username),
				zap.String("key", key))
			// 不返回错误，因为计数已经成功，过期时间失败不影响主流程
		}
	}

	log.Warn("记录登录失败", zap.String("username", username), zap.Int64("fail_count", count))
	return count, nil
}

// GetLoginFailCount 获取登录失败次数
func (ll *loginLimiter) GetLoginFailCount(ctx context.Context, username string) (int64, error) {
	key := LoginFailKeyPrefix + username
	countStr, err := ll.client.Get(ctx, key)
	if err != nil {
		// 使用字符串比较判断redis.Nil（因为没有直接导入redis包）
		if err.Error() == "redis: nil" {
			return 0, nil
		}
		return 0, err
	}

	count, err := strconv.ParseInt(countStr, 10, 64)
	if err != nil {
		log.Error("解析登录失败计数失败",
			zap.Error(err),
			zap.String("username", username),
			zap.String("count_str", countStr))
		return 0, fmt.Errorf("解析登录失败计数失败: %w", err)
	}
	return count, nil
}

// IsLoginAllowed 检查是否允许登录
func (ll *loginLimiter) IsLoginAllowed(ctx context.Context, username string) (bool, error) {
	count, err := ll.GetLoginFailCount(ctx, username)
	if err != nil {
		return false, err
	}

	allowed := count < MaxLoginAttempts
	if !allowed {
		log.Warn("登录尝试次数过多", zap.String("username", username), zap.Int64("fail_count", count))
	}
	return allowed, nil
}

// ResetLoginFail 重置登录失败计数
func (ll *loginLimiter) ResetLoginFail(ctx context.Context, username string) error {
	key := LoginFailKeyPrefix + username
	err := ll.client.Del(ctx, key)
	if err != nil {
		log.Error("重置登录失败计数失败", zap.Error(err), zap.String("username", username))
		return err
	}
	log.Info("重置登录失败计数", zap.String("username", username))
	return nil
}
