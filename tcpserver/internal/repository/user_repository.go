package repository

import (
	"context"
	"database/sql"
	"entry-task/tcpserver/internal/model"
	"entry-task/tcpserver/pkg/redis"
	"fmt"
	"math/rand"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	log "entry-task/tcpserver/pkg/logger"
)

const (
	doubleDeleteDelayTime = time.Millisecond * 500
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// UserRepository 用户仓储接口
type UserRepository interface {
	// GetByUsername 根据用户名查询用户（用于登录）
	GetByUsername(ctx context.Context, username string) (*model.User, error)

	// GetByID 根据ID查询用户
	GetByID(ctx context.Context, id uint64) (*redis.CachedUser, error)

	// Create 创建用户
	Create(ctx context.Context, user *model.User) error

	// UpdateNickname 更新用户昵称
	UpdateNickname(ctx context.Context, id uint64, nickname string) error

	// UpdateProfilePicture 更新用户头像
	UpdateProfilePicture(ctx context.Context, id uint64, profilePicture string) error

	// BatchCreate 批量创建用户（用于生成测试数据）
	BatchCreate(ctx context.Context, users []*model.User) error
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
func (r *userRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
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
func (r *userRepository) GetByID(ctx context.Context, id uint64) (*redis.CachedUser, error) {
	// 1. 先查缓存
	cachedUser, err := r.redisManager.GetUserCache().GetUser(ctx, id)
	if err != nil {
		// Redis 错误（不是 redis.Nil），记录日志但继续查数据库（降级策略）
		log.Error("查询Redis缓存失败", zap.Error(err), zap.Uint64("user_id", id))
		// 继续执行，尝试从数据库查询
	}

	// 2. 缓存命中
	if cachedUser != nil {
		log.Debug("用户缓存命中", zap.Uint64("user_id", id))
		return cachedUser, nil
	}

	// 3. 缓存未命中，查数据库（使用 model.User，带 db tag）
	var dbUser model.User
	query := `SELECT id, username, nickname, profile_picture FROM users WHERE id = ?`
	err = r.db.Get(&dbUser, query, id)

	if err != nil {
		if err == sql.ErrNoRows {
			// 4. 用户不存在，设置负缓存
			log.Debug("用户不存在，设置负缓存", zap.Uint64("user_id", id))
			if cacheErr := r.redisManager.GetUserCache().SetNullCache(ctx, id); cacheErr != nil {
				log.Error("设置负缓存失败", zap.Error(cacheErr), zap.Uint64("user_id", id))
				// 不返回 cacheErr，继续返回用户不存在的信息
			}
			return nil, nil // 用户不存在
		}
		// 数据库查询错误
		log.Error("数据库查询失败", zap.Error(err), zap.Uint64("user_id", id))
		return nil, err
	}

	// 5. 用户存在，转换为 CachedUser
	cachedUser = &redis.CachedUser{
		ID:             dbUser.ID,
		Username:       dbUser.Username,
		Nickname:       dbUser.Nickname,
		ProfilePicture: dbUser.ProfilePicture,
	}

	// 6. 异步设置缓存（不阻塞返回）
	go func() {
		setCtx := context.Background()
		if err := r.redisManager.GetUserCache().SetUser(setCtx, &dbUser); err != nil {
			log.Error("设置用户缓存失败", zap.Error(err), zap.Uint64("user_id", id))
		} else {
			log.Debug("设置用户缓存成功", zap.Uint64("user_id", id))
		}
	}()

	log.Debug("从数据库加载用户成功", zap.Uint64("user_id", id))
	return cachedUser, nil
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
func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	query := `INSERT INTO users (id, username, password_hash, nickname, profile_picture) 
              VALUES (?, ?, ?, ?, ?)`

	_, err := r.db.Exec(query, user.ID, user.Username, user.PasswordHash, user.Nickname, user.ProfilePicture)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// UpdateNickname 更新用户昵称
func (r *userRepository) UpdateNickname(ctx context.Context, id uint64, nickname string) error {
	// 1. 删除缓存（降级策略：失败不影响主流程）
	if err := r.redisManager.GetUserCache().DeleteUser(ctx, id); err != nil {
		log.Error("删除用户缓存失败（第一次）",
			zap.Error(err),
			zap.Uint64("user_id", id),
			zap.String("nickname", nickname))
		// 不返回错误，继续执行数据库更新
	}

	// 2. 更新数据库
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

	//延迟 doubleDeleteDelayTime 再次删除缓存
	uid := id                                                                       // 防止闭包捕获循环变量
	delay := doubleDeleteDelayTime + time.Duration(rand.Intn(200))*time.Millisecond //延迟抖动

	time.AfterFunc(delay, func() {
		ctx2, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := r.redisManager.GetUserCache().DeleteUser(ctx2, uid); err != nil {
			log.Error("delay delete failed", zap.Error(err), zap.Uint64("user_id", uid))
		}
	})

	log.Info("更新用户昵称成功",
		zap.Uint64("user_id", id),
		zap.String("nickname", nickname),
	)

	return nil
}

// UpdateProfilePicture 更新用户头像
func (r *userRepository) UpdateProfilePicture(ctx context.Context, id uint64, profilePicture string) error {
	// 1. 删除缓存（降级策略：失败不影响主流程）
	if err := r.redisManager.GetUserCache().DeleteUser(ctx, id); err != nil {
		log.Error("删除用户缓存失败（第一次）",
			zap.Error(err),
			zap.Uint64("user_id", id),
			zap.String("profile_picture", profilePicture))
		// 不返回错误，继续执行数据库更新
	}

	// 2. 更新数据库
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

	//延迟 doubleDeleteDelayTime 再次删除缓存
	uid := id                                                                       // 防止闭包捕获循环变量
	delay := doubleDeleteDelayTime + time.Duration(rand.Intn(200))*time.Millisecond //延迟抖动

	time.AfterFunc(delay, func() {
		ctx2, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := r.redisManager.GetUserCache().DeleteUser(ctx2, uid); err != nil {
			log.Error("delay delete failed", zap.Error(err), zap.Uint64("user_id", uid))
		}
	})

	log.Info("更新用户头像成功",
		zap.Uint64("user_id", id),
		zap.String("profile_picture", profilePicture),
	)

	return nil
}

// BatchCreate 批量创建用户
func (r *userRepository) BatchCreate(ctx context.Context, users []*model.User) error {
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
