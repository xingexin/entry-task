package dto

import (
	"errors"
	"regexp"
	"unicode/utf8"
)

var (
	// 用户名规则：3-50个字符，字母、数字、下划线
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,50}$`)
)

// ============================================================================
// 验证错误
// ============================================================================

var (
	ErrUsernameEmpty    = errors.New("用户名不能为空")
	ErrUsernameInvalid  = errors.New("用户名格式不正确（3-50个字符，仅限字母、数字、下划线）")
	ErrPasswordEmpty    = errors.New("密码不能为空")
	ErrPasswordTooShort = errors.New("密码长度不能少于6位")
	ErrPasswordTooLong  = errors.New("密码长度不能超过100位")
	ErrNicknameEmpty    = errors.New("昵称不能为空")
	ErrNicknameTooLong  = errors.New("昵称长度不能超过50个字符")
	ErrTokenEmpty       = errors.New("Token不能为空")
	ErrPictureURLEmpty  = errors.New("头像URL不能为空")
	ErrUserIDInvalid    = errors.New("用户ID无效")
)

// ============================================================================
// LoginDTO 验证
// ============================================================================

// Validate 验证登录DTO
func (d *LoginDTO) Validate() error {
	if d.Username == "" {
		return ErrUsernameEmpty
	}
	if !usernameRegex.MatchString(d.Username) {
		return ErrUsernameInvalid
	}
	if d.Password == "" {
		return ErrPasswordEmpty
	}
	if len(d.Password) < 6 {
		return ErrPasswordTooShort
	}
	if len(d.Password) > 100 {
		return ErrPasswordTooLong
	}
	return nil
}

// ============================================================================
// UpdateNicknameDTO 验证
// ============================================================================

// Validate 验证更新昵称DTO
func (d *UpdateNicknameDTO) Validate() error {
	if d.UserID == 0 {
		return ErrUserIDInvalid
	}
	if d.Nickname == "" {
		return ErrNicknameEmpty
	}
	// 支持Unicode字符（中文、emoji等）
	runeCount := utf8.RuneCountInString(d.Nickname)
	if runeCount > 50 {
		return ErrNicknameTooLong
	}
	return nil
}

// ============================================================================
// UpdateProfilePictureDTO 验证
// ============================================================================

// Validate 验证更新头像DTO
func (d *UpdateProfilePictureDTO) Validate() error {
	if d.UserID == 0 {
		return ErrUserIDInvalid
	}
	if d.ProfilePicture == "" {
		return ErrPictureURLEmpty
	}
	return nil
}

// ============================================================================
// ValidateTokenDTO 验证
// ============================================================================

// Validate 验证TokenDTO
func (d *ValidateTokenDTO) Validate() error {
	if d.Token == "" {
		return ErrTokenEmpty
	}
	return nil
}

// ============================================================================
// LogoutDTO 验证
// ============================================================================

// Validate 验证登出DTO
func (d *LogoutDTO) Validate() error {
	if d.Token == "" {
		return ErrTokenEmpty
	}
	return nil
}
