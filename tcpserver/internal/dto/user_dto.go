package dto

import "time"

// ============================================================================
// 用户信息 DTO
// ============================================================================

// UserDTO 完整用户信息（内部业务使用）
type UserDTO struct {
	ID             uint64
	Username       string
	PasswordHash   string // 仅内部使用，不对外暴露
	Nickname       string
	ProfilePicture string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// UserProfileDTO 用户公开信息（用于响应）
type UserProfileDTO struct {
	ID             uint64
	Username       string
	Nickname       string
	ProfilePicture string
}

// ============================================================================
// 操作 DTO
// ============================================================================

// UpdateNicknameDTO 更新昵称
type UpdateNicknameDTO struct {
	UserID   uint64
	Nickname string
}

// UpdateProfilePictureDTO 更新头像URL
type UpdateProfilePictureDTO struct {
	UserID         uint64
	ProfilePicture string
}

// ============================================================================
// 方法
// ============================================================================

// ToProfile 转换为公开Profile
func (u *UserDTO) ToProfile() *UserProfileDTO {
	return &UserProfileDTO{
		ID:             u.ID,
		Username:       u.Username,
		Nickname:       u.Nickname,
		ProfilePicture: u.ProfilePicture,
	}
}

// IsEmpty 检查UserProfile是否为空
func (p *UserProfileDTO) IsEmpty() bool {
	return p.ID == 0
}
