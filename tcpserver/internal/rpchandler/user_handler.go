package rpchandler

import (
	"context"
	pb "entry-task/proto/user"
	"entry-task/tcpserver/internal/dto"
	"entry-task/tcpserver/internal/service"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	log "entry-task/tcpserver/pkg/logger"
)

// ============================================================================
// RPC 错误码映射
// ============================================================================

const (
	CodeSuccess           = 0     // 成功
	CodeInvalidParams     = 40001 // 参数错误
	CodeInvalidCredential = 40002 // 用户名或密码错误
	CodeUnauthorized      = 40003 // Token无效或已过期
	CodeUserNotFound      = 40004 // 用户不存在
	CodeTooManyRequests   = 42901 // 请求过于频繁
	CodeInternalError     = 50001 // 内部错误
)

// ============================================================================
// UserServiceHandler gRPC Handler
// ============================================================================

type UserServiceHandler struct {
	pb.UnimplementedUserServiceServer // 嵌入未实现的服务器，保证向前兼容
	userService                       service.UserService
}

// NewUserServiceHandler 创建 gRPC Handler
func NewUserServiceHandler(userService service.UserService) *UserServiceHandler {
	return &UserServiceHandler{
		userService: userService,
	}
}

// ============================================================================
// Login 登录
// ============================================================================

func (h *UserServiceHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	// 1. Proto → DTO
	loginDTO := dto.FromProtoLoginRequest(req)

	// 2. 调用 Service 层
	result, err := h.userService.Login(ctx, loginDTO)

	// 3. 错误处理
	if err != nil {
		code, message := mapServiceError(err)
		log.Warn("登录失败",
			zap.String("username", req.Username),
			zap.Int32("code", code),
			zap.Error(err))

		return &pb.LoginResponse{
			Code:    code,
			Message: message,
			Token:   "",
			User:    nil,
		}, nil // 返回业务错误，不返回 gRPC 错误
	}

	// 4. DTO → Proto（成功）
	log.Info("登录成功", zap.String("username", req.Username))
	return result.ToProtoResponse(CodeSuccess, "登录成功"), nil
}

// ============================================================================
// Logout 登出
// ============================================================================

func (h *UserServiceHandler) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	// 1. Proto → DTO
	logoutDTO := dto.FromProtoLogoutRequest(req)

	// 2. 调用 Service 层
	err := h.userService.Logout(ctx, logoutDTO)

	// 3. 错误处理
	if err != nil {
		code, message := mapServiceError(err)
		log.Warn("登出失败",
			zap.String("token", req.Token),
			zap.Int32("code", code),
			zap.Error(err))

		return &pb.LogoutResponse{
			Code:    code,
			Message: message,
		}, nil
	}

	// 4. 成功响应
	log.Info("登出成功", zap.String("token", req.Token))
	return dto.ToProtoLogoutResponse(CodeSuccess, "登出成功"), nil
}

// ============================================================================
// GetProfile 获取用户信息
// ============================================================================

func (h *UserServiceHandler) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	// 1. Proto → DTO
	validateDTO := dto.FromProtoGetProfileRequest(req)

	// 2. 调用 Service 层
	profileDTO, err := h.userService.GetProfile(ctx, validateDTO)

	// 3. 错误处理
	if err != nil {
		code, message := mapServiceError(err)
		log.Warn("获取用户信息失败",
			zap.String("token", req.Token),
			zap.Int32("code", code),
			zap.Error(err))

		return &pb.GetProfileResponse{
			Code:    code,
			Message: message,
			User:    nil,
		}, nil
	}

	// 4. DTO → Proto（成功）
	log.Debug("获取用户信息成功", zap.Uint64("user_id", profileDTO.ID))
	return profileDTO.ToProtoGetProfileResponse(CodeSuccess, "获取成功"), nil
}

// ============================================================================
// UpdateNickname 更新昵称
// ============================================================================

func (h *UserServiceHandler) UpdateNickname(ctx context.Context, req *pb.UpdateNicknameRequest) (*pb.UpdateNicknameResponse, error) {
	// 1. 先验证 Token，获取 UserID
	validateDTO := &dto.ValidateTokenDTO{Token: req.Token}
	profileDTO, err := h.userService.GetProfile(ctx, validateDTO)
	if err != nil {
		code, message := mapServiceError(err)
		log.Warn("Token验证失败",
			zap.String("token", req.Token),
			zap.Int32("code", code),
			zap.Error(err))

		return &pb.UpdateNicknameResponse{
			Code:    code,
			Message: message,
			User:    nil,
		}, nil
	}

	// 2. Proto → DTO
	updateDTO := dto.FromProtoUpdateNicknameRequest(req, profileDTO.ID)

	// 3. 调用 Service 层
	updatedProfile, err := h.userService.UpdateNickname(ctx, updateDTO)

	// 4. 错误处理
	if err != nil {
		code, message := mapServiceError(err)
		log.Warn("更新昵称失败",
			zap.Uint64("user_id", profileDTO.ID),
			zap.String("nickname", req.Nickname),
			zap.Int32("code", code),
			zap.Error(err))

		return &pb.UpdateNicknameResponse{
			Code:    code,
			Message: message,
			User:    nil,
		}, nil
	}

	// 5. DTO → Proto（成功）
	log.Info("更新昵称成功",
		zap.Uint64("user_id", updatedProfile.ID),
		zap.String("nickname", req.Nickname))
	return updatedProfile.ToProtoUpdateNicknameResponse(CodeSuccess, "更新成功"), nil
}

// ============================================================================
// UpdateProfilePicture 更新头像
// ============================================================================

func (h *UserServiceHandler) UpdateProfilePicture(ctx context.Context, req *pb.UpdateProfilePictureRequest) (*pb.UpdateProfilePictureResponse, error) {
	// 1. 先验证 Token，获取 UserID
	validateDTO := &dto.ValidateTokenDTO{Token: req.Token}
	profileDTO, err := h.userService.GetProfile(ctx, validateDTO)
	if err != nil {
		code, message := mapServiceError(err)
		log.Warn("Token验证失败",
			zap.String("token", req.Token),
			zap.Int32("code", code),
			zap.Error(err))

		return &pb.UpdateProfilePictureResponse{
			Code:    code,
			Message: message,
			User:    nil,
		}, nil
	}

	// 2. Proto → DTO
	updateDTO := dto.FromProtoUpdateProfilePictureRequest(req, profileDTO.ID)

	// 3. 调用 Service 层
	updatedProfile, err := h.userService.UpdateProfilePicture(ctx, updateDTO)

	// 4. 错误处理
	if err != nil {
		code, message := mapServiceError(err)
		log.Warn("更新头像失败",
			zap.Uint64("user_id", profileDTO.ID),
			zap.String("profile_picture", req.ProfilePicture),
			zap.Int32("code", code),
			zap.Error(err))

		return &pb.UpdateProfilePictureResponse{
			Code:    code,
			Message: message,
			User:    nil,
		}, nil
	}

	// 5. DTO → Proto（成功）
	log.Info("更新头像成功",
		zap.Uint64("user_id", updatedProfile.ID),
		zap.String("profile_picture", req.ProfilePicture))
	return updatedProfile.ToProtoUpdateProfilePictureResponse(CodeSuccess, "更新成功"), nil
}

// ============================================================================
// 错误映射函数
// ============================================================================

// mapServiceError 将 Service 层错误映射为 RPC 错误码和消息
func mapServiceError(err error) (int32, string) {
	switch err {
	// 验证错误
	case dto.ErrUsernameEmpty, dto.ErrUsernameInvalid,
		dto.ErrPasswordEmpty, dto.ErrPasswordTooShort, dto.ErrPasswordTooLong,
		dto.ErrNicknameEmpty, dto.ErrNicknameTooLong,
		dto.ErrTokenEmpty, dto.ErrPictureURLEmpty, dto.ErrUserIDInvalid:
		return CodeInvalidParams, err.Error()

	// 登录错误
	case service.ErrInvalidCredentials:
		return CodeInvalidCredential, "用户名或密码错误"

	case service.ErrLoginLimitExceeded:
		return CodeTooManyRequests, "登录失败次数过多，请稍后再试"

	// Token错误
	case service.ErrInvalidToken:
		return CodeUnauthorized, "Token无效或已过期"

	// 用户不存在
	case service.ErrUserNotFound:
		return CodeUserNotFound, "用户不存在"

	// 其他内部错误
	default:
		return CodeInternalError, "内部错误"
	}
}

// ============================================================================
// gRPC 错误转换（可选，用于严重错误场景）
// ============================================================================

// toGRPCError 将业务错误转换为 gRPC 错误（严重错误时使用）
func toGRPCError(err error) error {
	switch err {
	case service.ErrInvalidToken:
		return status.Error(codes.Unauthenticated, err.Error())
	case service.ErrUserNotFound:
		return status.Error(codes.NotFound, err.Error())
	case service.ErrLoginLimitExceeded:
		return status.Error(codes.ResourceExhausted, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
