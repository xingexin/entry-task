package repository

import (
	"context"
	"database/sql"
	"entry-task/tcpserver/internal/model"
	"entry-task/tcpserver/pkg/redis"
	"fmt"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	log "entry-task/tcpserver/pkg/logger"
)

// UserRepository 用户仓储接口
type UserRepository interface {
	// GetByUsername 根据用户名查询用户（用于登录）
	GetByUsername(username string) (*model.User, error)

	// GetByID 根据ID查询用户
	GetByID(id uint64) (*model.User, error)

	// Create 创建用户
	Create(user *model.User) error

	// UpdateNickname 更新用户昵称
	UpdateNickname(id uint64, nickname string) error

	// UpdateProfilePicture 更新用户头像
	UpdateProfilePicture(id uint64, profilePicture string) error

	// BatchCreate 批量创建用户（用于生成测试数据）
	BatchCreate(users []*model.User) error
}

// userRepository 用户仓储实现
type userRepository struct {
	db           *sqlx.DB
	redisManager redis.Manager
}

// NewUserRepository 创建用户仓储实例
func NewUserRepository(db *sqlx.DB, redisManager redis.Manager) UserRepository {
	return &userRepository{
		db:           db,
		redisManager: redisManager,
	}
}

// GetByUsername 根据用户名查询用户
func (r *userRepository) GetByUsername(username string) (*model.User, error) {
	var user model.User
	query := `SELECT id, username, password_hash, nickname, profile_picture, created_at, updated_at 
              FROM users WHERE username = ?`

	err := r.db.Get(&user, query, username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", username)
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return &user, nil
}

// GetByID 根据ID查询用户（优先从缓存获取，自动处理负缓存）
func (r *userRepository) GetByID(id uint64) (*model.User, error) {
	ctx := context.Background()

	// 使用GetOrLoad自动处理缓存逻辑（含负缓存策略）
	user, err := r.redisManager.GetUserCache().GetOrLoad(ctx, id, func(userID uint64) (*model.User, error) {
		return r.getByIDFromDB(userID)
	})

	if err != nil {
		log.Error("查询用户失败",
			zap.Error(err),
			zap.Uint64("user_id", id),
		)
		return nil, err
	}

	return user, nil
}

// getByIDFromDB 从数据库查询用户（内部方法）
func (r *userRepository) getByIDFromDB(id uint64) (*model.User, error) {
	var user model.User
	query := `SELECT id, username, password_hash, nickname, profile_picture, created_at, updated_at 
              FROM users WHERE id = ?`

	err := r.db.Get(&user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // 用户不存在，返回nil（会触发负缓存）
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return &user, nil
}

// Create 创建用户
func (r *userRepository) Create(user *model.User) error {
	query := `INSERT INTO users (id, username, password_hash, nickname, profile_picture) 
              VALUES (?, ?, ?, ?, ?)`

	_, err := r.db.Exec(query, user.ID, user.Username, user.PasswordHash, user.Nickname, user.ProfilePicture)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// UpdateNickname 更新用户昵称（自动延迟双删缓存）
func (r *userRepository) UpdateNickname(id uint64, nickname string) error {
	ctx := context.Background()

	// 1. 更新数据库
	query := `UPDATE users SET nickname = ? WHERE id = ?`
	result, err := r.db.Exec(query, nickname, id)
	if err != nil {
		return fmt.Errorf("failed to update nickname: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found: %d", id)
	}

	// 2. 延迟双删缓存（立即删除 + 500ms后再删一次）
	if err := r.redisManager.GetUserCache().DeleteUserWithDelay(ctx, id); err != nil {
		log.Error("删除用户缓存失败（昵称更新后）",
			zap.Error(err),
			zap.Uint64("user_id", id),
			zap.String("nickname", nickname))
		// 不返回错误，因为数据库已更新成功，缓存删除失败不影响数据一致性
		// 缓存会在TTL后自动过期
	}

	log.Info("更新用户昵称成功",
		zap.Uint64("user_id", id),
		zap.String("nickname", nickname),
	)

	return nil
}

// UpdateProfilePicture 更新用户头像（自动延迟双删缓存）
func (r *userRepository) UpdateProfilePicture(id uint64, profilePicture string) error {
	ctx := context.Background()

	// 1. 更新数据库
	query := `UPDATE users SET profile_picture = ? WHERE id = ?`
	result, err := r.db.Exec(query, profilePicture, id)
	if err != nil {
		return fmt.Errorf("failed to update profile picture: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found: %d", id)
	}

	// 2. 延迟双删缓存（立即删除 + 500ms后再删一次）
	if err := r.redisManager.GetUserCache().DeleteUserWithDelay(ctx, id); err != nil {
		log.Error("删除用户缓存失败（头像更新后）",
			zap.Error(err),
			zap.Uint64("user_id", id),
			zap.String("profile_picture", profilePicture))
		// 不返回错误，因为数据库已更新成功，缓存删除失败不影响数据一致性
	}

	log.Info("更新用户头像成功",
		zap.Uint64("user_id", id),
		zap.String("profile_picture", profilePicture),
	)

	return nil
}

// BatchCreate 批量创建用户
func (r *userRepository) BatchCreate(users []*model.User) error {
	if len(users) == 0 {
		return nil
	}

	// 使用事务批量插入
	tx, err := r.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Error("事务回滚失败", zap.Error(err))
		}
	}()

	query := `INSERT INTO users (id, username, password_hash, nickname, profile_picture) 
              VALUES (?, ?, ?, ?, ?)`

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error("关闭statement失败", zap.Error(err))
		}
	}()

	for _, user := range users {
		_, err := stmt.Exec(user.ID, user.Username, user.PasswordHash, user.Nickname, user.ProfilePicture)
		if err != nil {
			return fmt.Errorf("failed to insert user %s: %w", user.Username, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
