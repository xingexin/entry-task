package handler

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"entry-task/httpserver/pkg/response"
	pb "entry-task/proto/user"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"

	log "entry-task/httpserver/pkg/logger"
)

const (
	// MaxFileSize 文件上传配置
	MaxFileSize       = 5 * 1024 * 1024                        // 5MB
	AllowedExtensions = ".jpg,.jpeg,.png,.webp"                // 允许的文件类型
	UploadDir         = "./uploads/avatars"                    // 上传目录
	DefaultAvatar     = "httpserver/static/default_avatar.png" // 默认头像
)

// ============================================================================
// Handler 结构体
// ============================================================================

type UserHandler struct {
	grpcClient pb.UserServiceClient
}

// NewUserHandler 创建 UserHandler 实例
func NewUserHandler(grpcClient pb.UserServiceClient) *UserHandler {
	return &UserHandler{
		grpcClient: grpcClient,
	}
}

// ============================================================================
// 请求结构体
// ============================================================================

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UpdateNicknameRequest struct {
	Nickname string `json:"nickname" binding:"required"`
}

// ============================================================================
// Handler 方法
// ============================================================================

// Login 登录
func (h *UserHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.CodeBadRequest, "请求参数错误")
		return
	}

	//设置超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	//调用gRPC的API
	loginResp, err := h.grpcClient.Login(ctx, &pb.LoginRequest{
		Username: req.Username,
		Password: req.Password,
	})

	if err != nil {
		log.Error("登录RPC调用失败", zap.Error(err))
		response.Error(c, response.CodeRPCError, "登录失败")
		return
	}

	if loginResp.Code != 0 {
		httpCode := mapRPCCode(loginResp.Code)
		response.Error(c, httpCode, loginResp.Message)
		return
	}

	// 设置Cookie（Web浏览器自动使用）
	c.SetCookie(
		"auth_token",    // Cookie名称
		loginResp.Token, // Token值
		7200,            // MaxAge: 2小时（秒）
		"/",             // Path: 全站有效
		"",              // Domain: 当前域
		false,           // Secure: 生产环境建议改为true
		true,            // HttpOnly: 防止XSS攻击
	)

	// 发送响应（必须在设置Cookie和Header之后）
	response.Success(c, gin.H{
		"username":   loginResp.User.Username,
		"nickname":   loginResp.User.Nickname,
		"avatar_url": "/api/v1/profile/picture",
	})
}

// GetProfile 获取用户信息
func (h *UserHandler) GetProfile(c *gin.Context) {
	token := extractToken(c)
	if token == "" {
		response.Error(c, response.CodeUnauthorized, "未认证")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx,
		metadata.Pairs("authorization", token))

	resp, err := h.grpcClient.GetProfile(ctx, &pb.GetProfileRequest{
		Token: token,
	})

	if err != nil {
		log.Error("RPC调用失败", zap.Error(err))
		response.Error(c, response.CodeRPCError, "获取用户信息失败")
		return
	}

	if resp.Code != 0 {
		httpCode := mapRPCCode(resp.Code)
		response.Error(c, httpCode, resp.Message)
		return
	}

	response.Success(c, gin.H{
		"username":   resp.User.Username,
		"nickname":   resp.User.Nickname,
		"avatar_url": "/api/v1/profile/picture",
	})
}

// UpdateNickname 更新昵称
func (h *UserHandler) UpdateNickname(c *gin.Context) {
	token := extractToken(c)
	if token == "" {
		response.Error(c, response.CodeUnauthorized, "未认证")
		return
	}

	var req UpdateNicknameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.CodeBadRequest, "请求参数错误")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx,
		metadata.Pairs("authorization", token))

	resp, err := h.grpcClient.UpdateNickname(ctx, &pb.UpdateNicknameRequest{
		Token:    token,
		Nickname: req.Nickname,
	})

	if err != nil {
		log.Error("RPC调用失败", zap.Error(err))
		response.Error(c, response.CodeRPCError, "更新昵称失败")
		return
	}

	if resp.Code != 0 {
		httpCode := mapRPCCode(resp.Code)
		response.Error(c, httpCode, resp.Message)
		return
	}

	response.Success(c, gin.H{
		"username":   resp.User.Username,
		"nickname":   resp.User.Nickname,
		"avatar_url": "/api/v1/profile/picture",
	})
}

// UploadProfilePicture 上传头像
func (h *UserHandler) UploadProfilePicture(c *gin.Context) {
	token := extractToken(c)
	if token == "" {
		response.Error(c, response.CodeUnauthorized, "未认证")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, response.CodeBadRequest, "请上传文件")
		return
	}

	if file.Size > MaxFileSize {
		response.Error(c, response.CodeFileTooLarge, "文件过大")
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !strings.Contains(AllowedExtensions, ext) {
		response.Error(c, response.CodeUnsupportedFileType, "不支持的文件类型")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx,
		metadata.Pairs("authorization", token))

	profileResp, err := h.grpcClient.GetProfile(ctx, &pb.GetProfileRequest{
		Token: token,
	})

	if err != nil || profileResp.Code != 0 {
		response.Error(c, response.CodeUnauthorized, "未认证")
		return
	}

	userID := profileResp.User.Id
	filename := fmt.Sprintf("%d%s", userID, ext)
	savePath := filepath.Join(UploadDir, filename)

	if err := os.MkdirAll(UploadDir, 0755); err != nil {
		log.Error("创建上传目录失败", zap.Error(err))
		response.Error(c, response.CodeInternalServerError, "服务器错误")
		return
	}

	if err := c.SaveUploadedFile(file, savePath); err != nil {
		log.Error("保存文件失败", zap.Error(err))
		response.Error(c, response.CodeInternalServerError, "保存文件失败")
		return
	}

	avatarURL := fmt.Sprintf("/uploads/avatars/%s", filename)

	ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel2()

	ctx2 = metadata.NewOutgoingContext(ctx2,
		metadata.Pairs("authorization", token))

	updateResp, err := h.grpcClient.UpdateProfilePicture(ctx2, &pb.UpdateProfilePictureRequest{
		Token:          token,
		ProfilePicture: avatarURL,
	})

	if err != nil {
		log.Error("RPC调用失败", zap.Error(err))
		// 尝试删除已上传的文件，失败只记录日志
		if removeErr := os.Remove(savePath); removeErr != nil {
			log.Warn("删除文件失败", zap.Error(removeErr), zap.String("path", savePath))
		}
		response.Error(c, response.CodeRPCError, "更新头像失败")
		return
	}

	if updateResp.Code != 0 {
		httpCode := mapRPCCode(updateResp.Code)
		// 尝试删除已上传的文件，失败只记录日志
		if removeErr := os.Remove(savePath); removeErr != nil {
			log.Warn("删除文件失败", zap.Error(removeErr), zap.String("path", savePath))
		}
		response.Error(c, httpCode, updateResp.Message)
		return
	}

	response.Success(c, gin.H{
		"avatar_url": "/api/v1/profile/picture",
	})
}

// GetProfilePicture 获取头像
func (h *UserHandler) GetProfilePicture(c *gin.Context) {
	token := extractToken(c)
	if token == "" {
		serveDefaultAvatar(c)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx,
		metadata.Pairs("authorization", token))

	resp, err := h.grpcClient.GetProfile(ctx, &pb.GetProfileRequest{
		Token: token,
	})

	if err != nil || resp.Code != 0 {
		serveDefaultAvatar(c)
		return
	}

	avatarURL := resp.User.AvatarUrl

	if avatarURL == "" || !strings.HasPrefix(avatarURL, "/uploads/avatars/") {
		serveDefaultAvatar(c)
		return
	}

	localPath := "." + avatarURL

	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		log.Warn("头像文件不存在",
			zap.String("path", localPath),
			zap.Uint64("user_id", resp.User.Id))
		serveDefaultAvatar(c)
		return
	}

	c.File(localPath)
}

// serveDefaultAvatar 返回默认头像
func serveDefaultAvatar(c *gin.Context) {
	// 检查默认头像文件是否存在
	if _, err := os.Stat(DefaultAvatar); os.IsNotExist(err) {
		log.Error("默认头像文件不存在", zap.String("path", DefaultAvatar))
		c.Status(http.StatusNotFound)
		return
	}
	c.File(DefaultAvatar)
}

// Logout 登出
func (h *UserHandler) Logout(c *gin.Context) {
	token := extractToken(c)
	if token == "" {
		response.Error(c, response.CodeUnauthorized, "未认证")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx,
		metadata.Pairs("authorization", token))

	resp, err := h.grpcClient.Logout(ctx, &pb.LogoutRequest{
		Token: token,
	})

	if err != nil {
		log.Error("RPC调用失败", zap.Error(err))
		response.Error(c, response.CodeRPCError, "登出失败")
		return
	}

	if resp.Code != 0 {
		httpCode := mapRPCCode(resp.Code)
		response.Error(c, httpCode, resp.Message)
		return
	}

	// 清除Cookie
	c.SetCookie(
		"auth_token",
		"", // 空值
		-1, // MaxAge=-1 表示立即删除
		"/",
		"",
		false,
		true,
	)

	response.Success(c, gin.H{})
}

// ============================================================================
// 工具函数
// ============================================================================

// extractToken 从请求头或Cookie中提取认证 Token
// 支持以下格式：
//   - Authorization: Bearer <token>
//   - Authorization: <token>
//   - Cookie: auth_token=<token>
//
// 返回去除 "Bearer " 前缀和首尾空格后的 token 字符串
func extractToken(c *gin.Context) string {
	token, _ := c.Cookie("auth_token")
	return token
}

// mapRPCCode 将 RPC 错误码映射为 HTTP 响应错误码
// 参数:
//   - rpcCode: TCP Server 返回的 gRPC 错误码
//
// 返回:
//   - 对应的 HTTP 错误码（定义在 response.Code 中）
func mapRPCCode(rpcCode int32) int {
	switch rpcCode {
	case 40001:
		return response.CodeInvalidParams
	case 40002:
		return response.CodeInvalidAccountOrPassword
	case 40003:
		return response.CodeUnauthorized
	case 40004:
		return response.CodeUserNotFound
	case 40104:
		return response.CodeInvalidNickname
	case 42901:
		return response.CodeBadRequest
	default:
		return response.CodeInternalServerError
	}
}
