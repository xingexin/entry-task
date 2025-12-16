package middleware

import (
	"context"
	"entry-task/tcpserver/pkg/redis"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	log "entry-task/tcpserver/pkg/logger"
)

// ============================================================================
// 1. 日志拦截器
// ============================================================================

// LoggingInterceptor 记录所有 RPC 请求的日志
func LoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// 记录请求开始
		log.Info("gRPC 请求开始",
			zap.String("method", info.FullMethod),
		)

		// 调用实际的 Handler
		resp, err := handler(ctx, req)

		// 记录请求结束
		duration := time.Since(start)
		if err != nil {
			log.Error("gRPC 请求失败",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
				zap.Error(err),
			)
		} else {
			log.Info("gRPC 请求成功",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
			)
		}

		return resp, err
	}
}

// ============================================================================
// 2. Panic 恢复拦截器
// ============================================================================

// RecoveryInterceptor 捕获 Panic 并返回错误
func RecoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		// 使用 defer + recover 捕获 panic
		defer func() {
			if r := recover(); r != nil {
				log.Error("gRPC Panic 恢复",
					zap.String("method", info.FullMethod),
					zap.Any("panic", r),
				)
				// 返回内部错误
				err = status.Error(codes.Internal, "服务内部错误")
			}
		}()

		// 调用 Handler
		return handler(ctx, req)
	}
}

// ============================================================================
// 3. 鉴权拦截器（核心！）
// ============================================================================

// AuthInterceptor Token 验证拦截器
func AuthInterceptor(redisManager redis.Manager) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {

		// ===== 第1步：检查白名单（不需要鉴权的方法）=====
		publicMethods := map[string]bool{
			"/user.UserService/Login": true, // 登录接口公开
		}

		if publicMethods[info.FullMethod] {
			// 白名单方法，直接放行
			log.Debug("公开方法，跳过鉴权", zap.String("method", info.FullMethod))
			return handler(ctx, req)
		}

		// ===== 第2步：提取 metadata =====
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			log.Warn("缺少 metadata", zap.String("method", info.FullMethod))
			return nil, status.Error(codes.Unauthenticated, "缺少认证信息")
		}

		// ===== 第3步：提取 Token =====
		tokens := md.Get("authorization")
		if len(tokens) == 0 {
			log.Warn("缺少 Token", zap.String("method", info.FullMethod))
			return nil, status.Error(codes.Unauthenticated, "缺少 Token")
		}

		token := tokens[0]
		if token == "" {
			log.Warn("Token 为空", zap.String("method", info.FullMethod))
			return nil, status.Error(codes.Unauthenticated, "Token 为空")
		}

		// ===== 第4步：验证 Token（调用 Redis Session）=====
		userID, err := redisManager.GetSession().ValidateSession(ctx, token)
		if err != nil {
			log.Warn("Token 验证失败",
				zap.String("method", info.FullMethod),
				zap.String("token", token),
				zap.Error(err),
			)
			return nil, status.Error(codes.Unauthenticated, "Token 无效或已过期")
		}

		// ===== 第5步：Token 有效，放入 context =====
		ctx = context.WithValue(ctx, "user_id", userID)
		log.Debug("Token 验证通过",
			zap.String("method", info.FullMethod),
			zap.Uint64("user_id", userID),
		)

		// ===== 第6步：放行，调用 Handler =====
		return handler(ctx, req)
	}
}

// ============================================================================
// 4. 性能监控拦截器
// ============================================================================

// MetricsInterceptor 性能指标收集
func MetricsInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// 调用 Handler
		resp, err := handler(ctx, req)

		// 记录性能指标
		duration := time.Since(start)
		log.Debug("RPC 性能指标",
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
			zap.Bool("success", err == nil),
		)

		// 这里可以集成 Prometheus 等监控系统
		// metrics.RecordRPCDuration(info.FullMethod, duration)

		return resp, err
	}
}
