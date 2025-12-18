package main

import (
	"flag"
	"fmt"
	"sync"
	"time"

	"entry-task/tcpserver/config"
	"entry-task/tcpserver/pkg/db"
	logger "entry-task/tcpserver/pkg/logger"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const (
	NumWorkers      = 10         // 10个并发worker
	DefaultPassword = "Test@123" // 统一测试密码
)

func main() {
	const BatchSize = 5000      // 每批插入5000条
	const TotalUsers = 10000000 // 1000万用户

	configPath := flag.String("config", "../config/config.yaml", "配置文件路径")
	flag.Parse()

	fmt.Println("========================================")
	fmt.Println("批量生成1000万用户数据")
	fmt.Println("========================================")

	// 1. 加载配置
	fmt.Println("正在加载配置文件...")
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Printf("❌ 加载配置失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 配置文件加载成功: %s\n", *configPath)

	// 2. 初始化日志
	logConfig := &logger.Config{
		Level:    cfg.Log.Level,
		Output:   cfg.Log.Output,
		FilePath: cfg.Log.FilePath,
	}
	if err := logger.Init(logConfig); err != nil {
		fmt.Printf("❌ 初始化日志失败: %v\n", err)
		return
	}
	defer logger.Sync()

	// 3. 初始化数据库连接（使用项目封装的方法）
	logger.Info("正在初始化数据库连接...")
	database, err := db.InitDB(cfg)
	if err != nil {
		logger.Fatal("❌ 初始化数据库失败", zap.Error(err))
		return
	}
	defer database.Close()

	// 4. 调整连接池配置（优化批量插入）
	database.SetMaxOpenConns(20)
	database.SetMaxIdleConns(10)
	logger.Info("数据库连接池配置完成", zap.Int("max_open_conns", 20), zap.Int("max_idle_conns", 10))

	// 5. 预先计算统一密码的hash（只计算一次！）
	logger.Info("正在生成密码哈希...")
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(DefaultPassword), bcrypt.DefaultCost)
	if err != nil {
		logger.Fatal("生成密码哈希失败", zap.Error(err))
		return
	}
	passwordHashStr := string(passwordHash)

	logger.Info("========================================")
	logger.Info("开始生成用户数据",
		zap.Int("total_users", TotalUsers),
		zap.String("password", DefaultPassword),
		zap.Int("batch_size", BatchSize),
		zap.Int("num_workers", NumWorkers))
	logger.Info("========================================")

	startTime := time.Now()

	// 计算每个worker负责的范围
	usersPerWorker := TotalUsers / NumWorkers
	var wg sync.WaitGroup

	for i := 0; i < NumWorkers; i++ {
		wg.Add(1)
		startID := i*usersPerWorker + 1
		endID := (i + 1) * usersPerWorker
		if i == NumWorkers-1 {
			endID = TotalUsers // 最后一个worker处理剩余的
		}

		go func(workerID, start, end int) {
			defer wg.Done()
			insertBatch(database, workerID, start, end, passwordHashStr)
		}(i, startID, endID)
	}

	wg.Wait()

	duration := time.Since(startTime)
	logger.Info("========================================")
	logger.Info("✅ 数据生成完成！",
		zap.Duration("total_time", duration),
		zap.Float64("avg_speed", float64(TotalUsers)/duration.Seconds()))
	logger.Info("========================================")
	logger.Info("用户信息",
		zap.String("username_format", "user00000001 到 user10000000"),
		zap.String("password", DefaultPassword))
}

func insertBatch(database *sqlx.DB, workerID, start, end int, passwordHash string) {
	total := end - start + 1
	processed := 0
	startTime := time.Now()

	for i := start; i <= end; i += BatchSize {
		batchEnd := i + BatchSize - 1
		if batchEnd > end {
			batchEnd = end
		}

		// 构建批量插入SQL
		query := "INSERT INTO users (id, username, password_hash, nickname, profile_picture, created_at, updated_at) VALUES "
		values := []interface{}{}
		now := time.Now()

		for j := i; j <= batchEnd; j++ {
			if j > i {
				query += ","
			}
			query += "(?, ?, ?, ?, ?, ?, ?)"

			username := fmt.Sprintf("user%08d", j)
			nickname := fmt.Sprintf("测试用户%08d", j)

			values = append(values,
				j,            // id
				username,     // username
				passwordHash, // password_hash（统一密码）
				nickname,     // nickname
				"",           // profile_picture
				now,          // created_at
				now,          // updated_at
			)
		}

		// 执行批量插入
		batchStart := time.Now()
		_, err := database.Exec(query, values...)
		batchDuration := time.Since(batchStart)

		if err != nil {
			logger.Error("批量插入失败",
				zap.Int("worker_id", workerID),
				zap.Int("start_id", i),
				zap.Int("end_id", batchEnd),
				zap.Error(err))
			continue
		}

		processed += (batchEnd - i + 1)

		// 每5万条输出一次进度
		if processed%50000 == 0 {
			progress := float64(processed) / float64(total) * 100
			elapsed := time.Since(startTime)
			speed := float64(processed) / elapsed.Seconds()
			logger.Info("生成进度",
				zap.Int("worker_id", workerID),
				zap.Float64("progress", progress),
				zap.Int("processed", processed),
				zap.Int("total", total),
				zap.Float64("speed", speed),
				zap.Duration("batch_duration", batchDuration))
		}
	}

	totalDuration := time.Since(startTime)
	avgSpeed := float64(processed) / totalDuration.Seconds()
	logger.Info("Worker完成",
		zap.Int("worker_id", workerID),
		zap.Int("processed", processed),
		zap.Duration("total_time", totalDuration),
		zap.Float64("avg_speed", avgSpeed))
}
