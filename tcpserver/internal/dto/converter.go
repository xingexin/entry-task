package dto

import (
	pb "entry-task/proto/user"
	"entry-task/tcpserver/internal/model"
	"entry-task/tcpserver/pkg/redis"
)

// ============================================================================
// Proto → DTO (gRPC 请求 → Service 层)
// ============================================================================

// FromProtoLoginRequest Proto登录请求 → DTO
func FromProtoLoginRequest(req *pb.LoginRequest) *LoginDTO {
	return &LoginDTO{
		Username: req.Username,
		Password: req.Password,
	}
}

// FromProtoLogoutRequest Proto登出请求 → DTO
func FromProtoLogoutRequest(req *pb.LogoutRequest) *LogoutDTO {
	return &LogoutDTO{
		Token: req.Token,
	}
}

// FromProtoGetProfileRequest Proto获取Profile请求 → DTO
func FromProtoGetProfileRequest(req *pb.GetProfileRequest) *ValidateTokenDTO {
	return &ValidateTokenDTO{
		Token: req.Token,
	}
}

// FromProtoUpdateNicknameRequest Proto更新昵称请求 → DTO
func FromProtoUpdateNicknameRequest(req *pb.UpdateNicknameRequest, userID uint64) *UpdateNicknameDTO {
	return &UpdateNicknameDTO{
		UserID:   userID,
		Nickname: req.Nickname,
	}
}

// FromProtoUpdateProfilePictureRequest Proto更新头像请求 → DTO
func FromProtoUpdateProfilePictureRequest(req *pb.UpdateProfilePictureRequest, userID uint64) *UpdateProfilePictureDTO {
	return &UpdateProfilePictureDTO{
		UserID:         userID,
		ProfilePicture: req.ProfilePicture,
	}
}

// ============================================================================
// DTO → Proto (Service 层 → gRPC 响应)
// ============================================================================

// ToProto UserProfileDTO → Proto UserProfile
func (p *UserProfileDTO) ToProto() *pb.UserProfile {
	if p == nil {
		return nil
	}
	return &pb.UserProfile{
		Id:        p.ID,
		Username:  p.Username,
		Nickname:  p.Nickname,
		AvatarUrl: p.ProfilePicture,
	}
}

// ToProtoResponse LoginResultDTO → Proto LoginResponse
func (r *LoginResultDTO) ToProtoResponse(code int32, message string) *pb.LoginResponse {
	return &pb.LoginResponse{
		Code:    code,
		Message: message,
		Token:   r.Token,
		User:    r.Profile.ToProto(),
	}
}

// ToProtoLogoutResponse LogoutDTO → Proto LogoutResponse
func ToProtoLogoutResponse(code int32, message string) *pb.LogoutResponse {
	return &pb.LogoutResponse{
		Code:    code,
		Message: message,
	}
}

// ToProtoGetProfileResponse UserProfileDTO → Proto GetProfileResponse
func (p *UserProfileDTO) ToProtoGetProfileResponse(code int32, message string) *pb.GetProfileResponse {
	return &pb.GetProfileResponse{
		Code:    code,
		Message: message,
		User:    p.ToProto(),
	}
}

// ToProtoUpdateNicknameResponse UserProfileDTO → Proto UpdateNicknameResponse
func (p *UserProfileDTO) ToProtoUpdateNicknameResponse(code int32, message string) *pb.UpdateNicknameResponse {
	return &pb.UpdateNicknameResponse{
		Code:    code,
		Message: message,
		User:    p.ToProto(),
	}
}

// ToProtoUpdateProfilePictureResponse UserProfileDTO → Proto UpdateProfilePictureResponse
func (p *UserProfileDTO) ToProtoUpdateProfilePictureResponse(code int32, message string) *pb.UpdateProfilePictureResponse {
	return &pb.UpdateProfilePictureResponse{
		Code:    code,
		Message: message,
		User:    p.ToProto(),
	}
}

// ============================================================================
// Model → DTO (Repository 层 → Service 层)
// ============================================================================

// FromModel model.User → UserDTO
func FromModel(user *model.User) *UserDTO {
	if user == nil {
		return nil
	}
	return &UserDTO{
		ID:             user.ID,
		Username:       user.Username,
		PasswordHash:   user.PasswordHash,
		Nickname:       user.Nickname,
		ProfilePicture: user.ProfilePicture,
		CreatedAt:      user.CreatedAt,
		UpdatedAt:      user.UpdatedAt,
	}
}

// FromCachedUser redis.CachedUser → UserProfileDTO
func FromCachedUser(cached *redis.CachedUser) *UserProfileDTO {
	if cached == nil {
		return nil
	}
	return &UserProfileDTO{
		ID:             cached.ID,
		Username:       cached.Username,
		Nickname:       cached.Nickname,
		ProfilePicture: cached.ProfilePicture,
	}
}

// ============================================================================
// DTO → Model (Service 层 → Repository 层)
// ============================================================================

// ToModel UserDTO → model.User
func (u *UserDTO) ToModel() *model.User {
	return &model.User{
		ID:             u.ID,
		Username:       u.Username,
		PasswordHash:   u.PasswordHash,
		Nickname:       u.Nickname,
		ProfilePicture: u.ProfilePicture,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
	}
}
