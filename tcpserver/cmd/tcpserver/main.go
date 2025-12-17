package main

import (
	pb "entry-task/proto/user"
	"entry-task/tcpserver/config"
	"entry-task/tcpserver/internal/middleware"
	"entry-task/tcpserver/internal/rpchandler"
	"entry-task/tcpserver/pkg/container"
	"entry-task/tcpserver/pkg/redis"
	"flag"
	"net"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	log "entry-task/tcpserver/pkg/logger"
)

var (
	configPath = flag.String("config", "config/config.yaml", "配置文件路径")
)

func main() {
	// 解析命令行参数
	flag.Parse()

	// 1. 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		panic("加载配置失败: " + err.Error())
	}

	// 2. 初始化日志
	logConfig := &log.Config{
		Level:    cfg.Log.Level,
		Output:   cfg.Log.Output,
		FilePath: cfg.Log.FilePath,
	}
	if err := log.Init(logConfig); err != nil {
		panic("初始化日志失败: " + err.Error())
	}
	defer log.Sync()

	log.Info("TCP Server 启动中...")
	log.Info("配置加载成功", zap.String("config_path", *configPath))

	// 3. 初始化依赖注入容器
	if err := container.Init(); err != nil {
		log.Fatal("初始化容器失败", zap.Error(err))
	}
	log.Info("依赖注入容器初始化成功")

	// 4. 注册配置到容器（供依赖注入使用）
	if err := container.Container.Provide(func() *config.Config {
		return cfg
	}); err != nil {
		log.Fatal("注册配置失败", zap.Error(err))
	}

	// 5. 从容器获取 RedisManager（用于鉴权拦截器）
	var redisManager redis.Manager
	if err := container.Invoke(func(rm redis.Manager) {
		redisManager = rm
	}); err != nil {
		log.Fatal("获取 RedisManager 失败", zap.Error(err))
	}
	log.Info("RedisManager 初始化成功")

	// 6. 创建 gRPC Server，注册拦截器链
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.RecoveryInterceptor(),         // 第1层：Panic 恢复（最外层）
			middleware.LoggingInterceptor(),          // 第2层：日志记录
			middleware.AuthInterceptor(redisManager), // 第3层：鉴权验证
			middleware.MetricsInterceptor(),          // 第4层：性能监控（最内层）
		),
	)
	log.Info("gRPC Server 创建成功，拦截器链已注册")

	// 7. 从容器获取 Handler
	var handler *rpchandler.UserServiceHandler
	if err := container.Invoke(func(h *rpchandler.UserServiceHandler) {
		handler = h
	}); err != nil {
		log.Fatal("获取 Handler 失败", zap.Error(err))
	}

	// 8. 注册 gRPC 服务
	pb.RegisterUserServiceServer(grpcServer, handler)
	log.Info("gRPC 服务注册成功",
		zap.String("service", "UserService"),
		zap.Int("methods", 5),
	)

	// 9. 监听端口
	addr := cfg.Server.GetTCPAddr()
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal("监听失败", zap.String("addr", addr), zap.Error(err))
	}

	// 10. 启动 gRPC Server（在 goroutine 中）
	go func() {
		log.Info("TCP Server 启动成功",
			zap.String("addr", addr),
			zap.String("mode", cfg.Server.Mode),
		)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal("启动 gRPC Server 失败", zap.Error(err))
		}
	}()

	// 11. 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("收到退出信号，开始优雅关闭...")

	// 12. 优雅关闭 gRPC Server
	grpcServer.GracefulStop()
	log.Info("TCP Server 已关闭")
}
