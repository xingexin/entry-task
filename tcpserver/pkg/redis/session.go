package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	log "entry-task/tcpserver/pkg/logger"
)

const (
	// SessionTTL Session默认过期时间（2小时）
	SessionTTL = 2 * time.Hour

	// SessionKeyPrefix Session键前缀
	SessionKeyPrefix = "sess:"
)

// SessionManager Session管理器接口
type SessionManager interface {
	// CreateSession 创建Session（生成token并存储到Redis）
	CreateSession(ctx context.Context, userID uint64) (string, error)

	// ValidateSession 验证Session（根据token获取userID）
	ValidateSession(ctx context.Context, token string) (uint64, error)

	// DestroySession 销毁Session（登出时删除token）
	DestroySession(ctx context.Context, token string) error

	// RefreshSession 刷新Session（延长有效期）
	RefreshSession(ctx context.Context, token string) error
}

// sessionManager Session管理器实现
type sessionManager struct {
	client Client
}

// NewSessionManager 创建Session管理器
func NewSessionManager(client Client) SessionManager {
	return &sessionManager{client: client}
}

// CreateSession 创建Session
func (sm *sessionManager) CreateSession(ctx context.Context, userID uint64) (string, error) {
	token := uuid.New().String()
	key := SessionKeyPrefix + token

	err := sm.client.Set(ctx, key, userID, SessionTTL)
	if err != nil {
		log.Error("创建Session失败", zap.Error(err), zap.Uint64("user_id", userID))
		return "", fmt.Errorf("创建Session失败: %w", err)
	}

	log.Info("创建Session成功", zap.String("token", token), zap.Uint64("user_id", userID))
	return token, nil
}

// ValidateSession 验证Session
func (sm *sessionManager) ValidateSession(ctx context.Context, token string) (uint64, error) {
	key := SessionKeyPrefix + token
	userID, err := sm.client.GetUint64(ctx, key)
	if err != nil {
		return 0, fmt.Errorf("Session无效或已过期: %w", err)
	}
	return userID, nil
}

// DestroySession 销毁Session
func (sm *sessionManager) DestroySession(ctx context.Context, token string) error {
	key := SessionKeyPrefix + token
	err := sm.client.Del(ctx, key)
	if err != nil {
		log.Error("销毁Session失败", zap.Error(err), zap.String("token", token))
		return err
	}
	log.Info("销毁Session成功", zap.String("token", token))
	return nil
}

// RefreshSession 刷新Session
func (sm *sessionManager) RefreshSession(ctx context.Context, token string) error {
	key := SessionKeyPrefix + token
	return sm.client.Expire(ctx, key, SessionTTL)
}
