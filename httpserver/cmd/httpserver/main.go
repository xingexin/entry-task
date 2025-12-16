package main

import (
	"entry-task/httpserver/config"
	"entry-task/httpserver/internal/handler"
	"entry-task/httpserver/internal/router"
	pb "entry-task/proto/user"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	log "entry-task/httpserver/pkg/logger"
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

	log.Info("HTTP Server 启动中...")
	log.Info("配置加载成功", zap.String("config_path", *configPath))

	// 3. 连接 gRPC Server（TCP Server）
	grpcAddr := cfg.GRPC.GetAddr()
	log.Info("正在连接 gRPC Server...", zap.String("addr", grpcAddr))

	conn, err := grpc.Dial(
		grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal("连接 gRPC Server 失败",
			zap.String("addr", grpcAddr),
			zap.Error(err))
	}
	defer conn.Close()

	log.Info("gRPC 连接成功", zap.String("addr", grpcAddr))

	// 4. 创建 gRPC Client
	grpcClient := pb.NewUserServiceClient(conn)

	// 5. 创建 Handler（依赖注入）
	userHandler := handler.NewUserHandler(grpcClient)
	log.Info("Handler 创建成功")

	// 6. 设置路由
	r := router.SetupRouter(userHandler)
	log.Info("路由设置完成")

	// 7. 启动 HTTP Server（在 goroutine 中）
	addr := cfg.Server.GetHTTPAddr()
	go func() {
		log.Info("HTTP Server 启动成功",
			zap.String("addr", addr),
			zap.String("mode", cfg.Server.Mode))

		if err := r.Run(addr); err != nil {
			log.Fatal("启动 HTTP Server 失败", zap.Error(err))
		}
	}()

	// 8. 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("收到退出信号，开始优雅关闭...")
	log.Info("HTTP Server 已关闭")
}
