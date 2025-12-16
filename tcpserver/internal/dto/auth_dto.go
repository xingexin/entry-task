package dto

// ============================================================================
// 登录相关 DTO
// ============================================================================

// LoginDTO 登录请求
type LoginDTO struct {
	Username string
	Password string // 明文密码
}

// LoginResultDTO 登录结果
type LoginResultDTO struct {
	Token   string
	Profile *UserProfileDTO
}

// LogoutDTO 登出请求
type LogoutDTO struct {
	Token string
}

// ============================================================================
// Token 验证 DTO
// ============================================================================

// ValidateTokenDTO Token验证请求
type ValidateTokenDTO struct {
	Token string
}

// TokenResultDTO Token验证结果
type TokenResultDTO struct {
	UserID uint64
	Valid  bool
}
