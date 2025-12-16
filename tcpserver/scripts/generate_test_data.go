package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

// ============================================================================
// 配置常量
// ============================================================================

const (
	// 数据生成配置
	TotalUsers      = 10000000    // 1000 万用户
	BatchSize       = 5000        // 每批插入 5000 条（平衡性能和内存）
	WorkerCount     = 10          // 并发 worker 数量
	DefaultPassword = "P@ssw0rd!" // 默认密码

	// 数据库配置（默认值，可通过命令行参数覆盖）
	DefaultDSN = "root:root@tcp(192.168.215.4:3306)/entrytask?charset=utf8mb4&parseTime=True&loc=Local"
)

// ============================================================================
// 雪花ID生成器（简化版）
// ============================================================================

type SnowflakeGenerator struct {
	mu        sync.Mutex
	machineID int64
	sequence  int64
	lastTime  int64
}

func NewSnowflakeGenerator(machineID int64) *SnowflakeGenerator {
	return &SnowflakeGenerator{
		machineID: machineID,
		sequence:  0,
		lastTime:  0,
	}
}

func (s *SnowflakeGenerator) NextID() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixMilli()
	if now == s.lastTime {
		s.sequence++
		if s.sequence > 4095 {
			// 等待下一毫秒
			for now <= s.lastTime {
				now = time.Now().UnixMilli()
			}
			s.sequence = 0
		}
	} else {
		s.sequence = 0
	}

	s.lastTime = now

	// 雪花ID位分配：
	// 41 位时间戳 | 10 位机器ID | 12 位序列号
	id := (now << 22) | (s.machineID << 12) | s.sequence
	return uint64(id)
}

// ============================================================================
// 用户数据结构
// ============================================================================

type User struct {
	ID             uint64
	Username       string
	PasswordHash   string
	Nickname       string
	ProfilePicture string
}

// ============================================================================
// 数据生成器
// ============================================================================

type DataGenerator struct {
	db           *sql.DB
	snowflake    *SnowflakeGenerator
	passwordHash string
}

func NewDataGenerator(db *sql.DB) (*DataGenerator, error) {
	// 预先生成密码哈希（所有用户使用相同密码，提高性能）
	hash, err := bcrypt.GenerateFromPassword([]byte(DefaultPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("生成密码哈希失败: %w", err)
	}

	return &DataGenerator{
		db:           db,
		snowflake:    NewSnowflakeGenerator(1), // 机器ID=1
		passwordHash: string(hash),
	}, nil
}

// GenerateUser 生成单个用户数据
func (g *DataGenerator) GenerateUser(index int) *User {
	return &User{
		ID:             g.snowflake.NextID(),
		Username:       fmt.Sprintf("user%08d", index+1),
		PasswordHash:   g.passwordHash,
		Nickname:       fmt.Sprintf("User%d", index+1),
		ProfilePicture: "",
	}
}

// BatchInsert 批量插入用户数据
func (g *DataGenerator) BatchInsert(users []*User) error {
	if len(users) == 0 {
		return nil
	}

	// 开始事务
	tx, err := g.db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	// 准备 SQL 语句
	query := `INSERT INTO users (id, username, password_hash, nickname, profile_picture) 
	          VALUES (?, ?, ?, ?, ?)`
	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("准备语句失败: %w", err)
	}
	defer stmt.Close()

	// 批量插入
	for _, user := range users {
		if _, err := stmt.Exec(
			user.ID,
			user.Username,
			user.PasswordHash,
			user.Nickname,
			user.ProfilePicture,
		); err != nil {
			return fmt.Errorf("插入数据失败: %w", err)
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}

// ============================================================================
// 进度显示器
// ============================================================================

type ProgressTracker struct {
	total     int
	current   int
	startTime time.Time
	mu        sync.Mutex
}

func NewProgressTracker(total int) *ProgressTracker {
	return &ProgressTracker{
		total:     total,
		current:   0,
		startTime: time.Now(),
	}
}

func (p *ProgressTracker) Add(n int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current += n
	elapsed := time.Since(p.startTime).Seconds()
	percent := float64(p.current) / float64(p.total) * 100
	speed := float64(p.current) / elapsed

	fmt.Printf("\r进度: %d/%d (%.2f%%) | 速度: %.0f 条/秒 | 已用时: %.0f 秒",
		p.current, p.total, percent, speed, elapsed)
}

func (p *ProgressTracker) Finish() {
	p.mu.Lock()
	defer p.mu.Unlock()

	elapsed := time.Since(p.startTime).Seconds()
	fmt.Printf("\n完成！总计: %d 条 | 总用时: %.2f 秒 | 平均速度: %.0f 条/秒\n",
		p.total, elapsed, float64(p.total)/elapsed)
}

// ============================================================================
// 主函数
// ============================================================================

func main() {
	// 解析命令行参数
	dsn := flag.String("dsn", DefaultDSN, "MySQL 数据源名称")
	workers := flag.Int("workers", WorkerCount, "并发 worker 数量")
	batchSize := flag.Int("batch", BatchSize, "每批插入数量")
	flag.Parse()

	fmt.Println("=============================================================================")
	fmt.Println("测试数据生成工具")
	fmt.Println("=============================================================================")
	fmt.Printf("目标数据量: %d 条\n", TotalUsers)
	fmt.Printf("批次大小: %d 条/批\n", *batchSize)
	fmt.Printf("并发数: %d 个 worker\n", *workers)
	fmt.Printf("数据库: %s\n", *dsn)
	fmt.Println("=============================================================================")

	// 连接数据库
	fmt.Println("正在连接数据库...")
	db, err := sql.Open("mysql", *dsn)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	// 配置连接池
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Fatalf("数据库连接测试失败: %v", err)
	}
	fmt.Println("数据库连接成功！")

	// 检查表是否存在
	var tableName string
	err = db.QueryRow("SHOW TABLES LIKE 'users'").Scan(&tableName)
	if err == sql.ErrNoRows {
		log.Fatal("错误：users 表不存在，请先执行 schema.sql 创建表")
	} else if err != nil {
		log.Fatalf("检查表失败: %v", err)
	}
	fmt.Println("users 表已存在")

	// 检查现有数据量
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		log.Fatalf("查询现有数据失败: %v", err)
	}
	if count > 0 {
		fmt.Printf("警告：表中已有 %d 条数据\n", count)
		fmt.Print("是否继续生成数据？（输入 yes 继续）: ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "yes" {
			fmt.Println("已取消")
			return
		}
	}

	// 创建数据生成器
	generator, err := NewDataGenerator(db)
	if err != nil {
		log.Fatalf("创建数据生成器失败: %v", err)
	}

	// 创建进度跟踪器
	progress := NewProgressTracker(TotalUsers)

	// 创建任务通道
	taskChan := make(chan int, *workers*2)
	var wg sync.WaitGroup

	// 启动 worker
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for batchStart := range taskChan {
				// 确定批次大小
				batchEnd := batchStart + *batchSize
				if batchEnd > TotalUsers {
					batchEnd = TotalUsers
				}
				currentBatchSize := batchEnd - batchStart

				// 生成批次数据
				users := make([]*User, 0, currentBatchSize)
				for j := batchStart; j < batchEnd; j++ {
					users = append(users, generator.GenerateUser(j))
				}

				// 插入数据库（带重试）
				for retry := 0; retry < 3; retry++ {
					if err := generator.BatchInsert(users); err != nil {
						if retry < 2 {
							log.Printf("Worker %d: 批次 %d-%d 插入失败，重试中... (%v)",
								workerID, batchStart, batchEnd, err)
							time.Sleep(time.Second * time.Duration(retry+1))
							continue
						}
						log.Fatalf("Worker %d: 批次 %d-%d 插入失败: %v",
							workerID, batchStart, batchEnd, err)
					}
					break
				}

				// 更新进度
				progress.Add(currentBatchSize)
			}
		}(i)
	}

	// 分配任务
	fmt.Println("\n开始生成数据...")
	for i := 0; i < TotalUsers; i += *batchSize {
		taskChan <- i
	}
	close(taskChan)

	// 等待所有 worker 完成
	wg.Wait()
	progress.Finish()

	// 验证数据
	fmt.Println("\n正在验证数据...")
	if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		log.Fatalf("验证数据失败: %v", err)
	}
	fmt.Printf("当前表中共有 %d 条数据\n", count)

	fmt.Println("\n=============================================================================")
	fmt.Println("数据生成完成！")
	fmt.Println("=============================================================================")
	fmt.Println("测试账号示例:")
	fmt.Println("  用户名: user00000001, user00000002, ..., user10000000")
	fmt.Println("  密码: P@ssw0rd!")
	fmt.Println("=============================================================================")
}
