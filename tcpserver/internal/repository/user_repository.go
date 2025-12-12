package repository

import (
	"database/sql"
	"entry-task/tcpserver/internal/model"
	"fmt"

	"github.com/jmoiron/sqlx"
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
	db *sqlx.DB
}

// NewUserRepository 创建用户仓储实例
func NewUserRepository(db *sqlx.DB) UserRepository {
	return &userRepository{db: db}
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

// GetByID 根据ID查询用户
func (r *userRepository) GetByID(id uint64) (*model.User, error) {
	var user model.User
	query := `SELECT id, username, password_hash, nickname, profile_picture, created_at, updated_at 
              FROM users WHERE id = ?`

	err := r.db.Get(&user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %d", id)
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

// UpdateNickname 更新用户昵称
func (r *userRepository) UpdateNickname(id uint64, nickname string) error {
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

	return nil
}

// UpdateProfilePicture 更新用户头像
func (r *userRepository) UpdateProfilePicture(id uint64, profilePicture string) error {
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
	defer func(tx *sqlx.Tx) {
		err := tx.Rollback()
		if err != nil {

		}
	}(tx)

	query := `INSERT INTO users (id, username, password_hash, nickname, profile_picture) 
              VALUES (?, ?, ?, ?, ?)`

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func(stmt *sql.Stmt) {
		err := stmt.Close()
		if err != nil {

		}
	}(stmt)

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
