package main

import (
	"context"
	"entry-task/tcpserver/config"
	"entry-task/tcpserver/internal/model"
	"entry-task/tcpserver/internal/repository"
	"entry-task/tcpserver/pkg/container"
	"entry-task/tcpserver/pkg/db"
	"entry-task/tcpserver/pkg/logger"
	"flag"
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var (
	configPath = flag.String("config", "/Users/chuyao.zhuo/GolandProjects/entry-task/tcpserver/config/config.yaml", "配置文件路径")
	username   = flag.String("username", "testuser", "用户名")
	password   = flag.String("password", "password", "密码")
	nickname   = flag.String("nickname", "测试用户", "昵称")
)

func main() {
	flag.Parse()

	// 1. 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		panic("加载配置失败: " + err.Error())
	}

	// 2. 初始化日志
	logConfig := &logger.Config{
		Level:    cfg.Log.Level,
		Output:   cfg.Log.Output,
		FilePath: cfg.Log.FilePath,
	}
	if err := logger.Init(logConfig); err != nil {
		panic("初始化日志失败: " + err.Error())
	}
	defer logger.Sync()

	logger.Info("开始创建测试用户...")

	// 3. 初始化依赖注入容器
	if err := container.Init(); err != nil {
		logger.Fatal("初始化容器失败", zap.Error(err))
	}

	// 4. 注册配置到容器
	if err := container.Container.Provide(func() *config.Config {
		return cfg
	}); err != nil {
		logger.Fatal("注册配置失败", zap.Error(err))
	}

	// 5. 获取 UserRepository
	var userRepo repository.UserRepository
	if err := container.Invoke(func(repo repository.UserRepository) {
		userRepo = repo
	}); err != nil {
		logger.Fatal("获取 UserRepository 失败", zap.Error(err))
	}

	// 6. 使用雪花算法生成 ID
	userID, err := db.GenerateID()
	if err != nil {
		logger.Fatal("生成雪花ID失败", zap.Error(err))
	}
	logger.Info("生成雪花ID", zap.Int64("id", userID))

	// 7. 使用 bcrypt 加密密码
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		logger.Fatal("加密密码失败", zap.Error(err))
	}
	logger.Info("密码加密成功")

	// 8. 创建用户对象
	user := &model.User{
		ID:             uint64(userID),
		Username:       *username,
		PasswordHash:   string(passwordHash),
		Nickname:       *nickname,
		ProfilePicture: "",
	}

	// 9. 调用 Repository 的 Create 方法
	ctx := context.Background()
	if err := userRepo.Create(ctx, user); err != nil {
		logger.Fatal("创建用户失败", zap.Error(err))

	}

	// 10. 成功提示
	logger.Info("✅ 测试用户创建成功！",
		zap.String("username", user.Username),
		zap.String("password", *password),
		zap.String("nickname", user.Nickname),
		zap.Uint64("user_id", user.ID),
	)

	fmt.Println("\n=========================================")
	fmt.Printf("✅ 测试账号创建成功！\n")
	fmt.Println("=========================================")
	fmt.Printf("用户名:  %s\n", user.Username)
	fmt.Printf("密码:    %s\n", *password)
	fmt.Printf("昵称:    %s\n", user.Nickname)
	fmt.Printf("用户ID:  %d (雪花算法生成)\n", user.ID)
	fmt.Println("=========================================")
	fmt.Println("\n📝 现在可以使用这个账号测试登录了！")
	fmt.Printf("\n测试命令：\n")
	fmt.Printf("curl -X POST http://localhost:8080/api/v1/auth/login \\\n")
	fmt.Printf("  -H \"Content-Type: application/json\" \\\n")
	fmt.Printf("  -d '{\"username\": \"%s\", \"password\": \"%s\"}'\n\n", user.Username, *password)
}
