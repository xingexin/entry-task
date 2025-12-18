package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	RedisAddr   = "192.168.215.6:6379" // Redis地址
	RedisDB     = 0                    // Redis数据库
	SessionTTL  = 2 * time.Hour        // Session过期时间（2小时）
	MaxParallel = 500                  // 并发数
	BatchSize   = 10000                // 批量写入文件的大小
)

var redisClient *redis.Client

func main() {

	userCount := flag.Int("count", 200, "需要生成的Session数量")
	flag.Parse()

	fmt.Println("========================================")
	fmt.Printf("直接创建Redis Session（跳过bcrypt验证）\n")
	fmt.Printf("生成 %d 个Session (并发: %d)\n", *userCount, MaxParallel)
	fmt.Println("========================================")

	// 连接Redis
	redisClient = redis.NewClient(&redis.Options{
		Addr:     RedisAddr,
		Password: "", // 如果有密码请填写
		DB:       RedisDB,
		PoolSize: MaxParallel, // 连接池大小匹配并发数
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		fmt.Printf("❌ Redis连接失败: %v\n", err)
		fmt.Println("请检查Redis地址和配置")
		return
	}
	fmt.Println("✅ Redis连接成功")

	startTime := time.Now()

	// 使用channel收集token
	tokenChan := make(chan TokenResult, MaxParallel*2)
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, MaxParallel)

	var successCount int64
	var failCount int64

	// 启动进度显示
	stopProgress := make(chan struct{})
	go showProgress(&successCount, &failCount, *userCount, stopProgress)

	// 启动文件写入
	writerDone := make(chan struct{})
	filename := fmt.Sprintf("tokens_%d.txt", *userCount)
	luaFilename := fmt.Sprintf("tokens_%d.lua", *userCount)
	go tokenWriter(tokenChan, filename, luaFilename, writerDone)

	// 批量创建Session
	for i := 1; i <= *userCount; i++ {
		wg.Add(1)
		semaphore <- struct{}{} // 获取信号量

		go func(idx int) {
			defer wg.Done()
			defer func() { <-semaphore }() // 释放信号量

			username := fmt.Sprintf("user%08d", idx)
			userID := uint64(idx)
			token := createSession(ctx, userID)

			if token != "" {
				tokenChan <- TokenResult{Username: username, Token: token, Index: idx}
				atomic.AddInt64(&successCount, 1)
			} else {
				atomic.AddInt64(&failCount, 1)
			}
		}(i)
	}

	wg.Wait()
	close(tokenChan) // 通知writer结束
	<-writerDone     // 等待writer完成

	close(stopProgress) // 停止进度显示
	elapsed := time.Since(startTime)

	fmt.Println("\n========================================")
	fmt.Printf("✅ Session创建完成！\n")
	fmt.Printf("   成功: %d/%d (%.1f%%)\n", successCount, *userCount, float64(successCount)/float64(*userCount)*100)
	fmt.Printf("   失败: %d (%.1f%%)\n", failCount, float64(failCount)/float64(*userCount)*100)
	fmt.Printf("   耗时: %.2fs\n", elapsed.Seconds())
	fmt.Printf("   平均速度: %.0f Session/秒\n", float64(successCount)/elapsed.Seconds())
	fmt.Println("========================================")
	fmt.Printf("✅ Token已保存到: %s\n", filename)
	fmt.Printf("✅ Lua格式Token已保存到: %s\n", luaFilename)
	fmt.Println("\n💡 提示：这些token有效期为2小时")
}

type TokenResult struct {
	Username string
	Token    string
	Index    int
}

// 直接在Redis中创建Session（绕过HTTP登录和bcrypt）
func createSession(ctx context.Context, userID uint64) string {
	// 生成UUID作为token
	token := uuid.New().String()
	key := fmt.Sprintf("sess:%s", token)

	// 在Redis中设置 sess:token → userID，过期时间2小时
	err := redisClient.Set(ctx, key, userID, SessionTTL).Err()
	if err != nil {
		return ""
	}

	return token
}

// 进度显示
func showProgress(successCount, failCount *int64, total int, stop chan struct{}) {
	ticker := time.NewTicker(1 * time.Second) // 每秒显示一次
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			success := atomic.LoadInt64(successCount)
			fail := atomic.LoadInt64(failCount)
			progress := float64(success+fail) / float64(total) * 100
			fmt.Printf("\r进度: %.1f%% (%d/%d) 成功: %d 失败: %d",
				progress, success+fail, total, success, fail)
		}
	}
}

// Token批量写入文件
func tokenWriter(tokenChan <-chan TokenResult, filename, luaFilename string, done chan struct{}) {
	defer close(done)

	scriptDir, _ := os.Getwd()
	txtPath := filepath.Join(scriptDir, filename)
	luaPath := filepath.Join(scriptDir, luaFilename)

	// 打开txt文件
	txtFile, err := os.Create(txtPath)
	if err != nil {
		fmt.Printf("\n❌ 创建txt文件失败: %v\n", err)
		return
	}
	defer txtFile.Close()

	// 打开lua文件
	luaFile, err := os.Create(luaPath)
	if err != nil {
		fmt.Printf("\n❌ 创建lua文件失败: %v\n", err)
		return
	}
	defer luaFile.Close()

	// 写入文件头
	fmt.Fprintf(txtFile, "# 批量生成的Token列表（直接Redis Session）\n")
	fmt.Fprintf(txtFile, "# 生成时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(txtFile, "# 有效期: 2小时\n")
	fmt.Fprintf(txtFile, "#\n")

	fmt.Fprintf(luaFile, "-- 批量生成的Token列表（Lua数组格式）\n")
	fmt.Fprintf(luaFile, "-- 生成时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(luaFile, "-- 有效期: 2小时\n\n")
	fmt.Fprintf(luaFile, "local tokens = {\n")

	// 批量写入缓冲
	txtBuf := make([]byte, 0, 1024*1024) // 1MB缓冲
	luaBuf := make([]byte, 0, 1024*1024)
	count := 0

	for result := range tokenChan {
		// 写入txt格式: username token
		txtBuf = append(txtBuf, fmt.Sprintf("%s %s\n", result.Username, result.Token)...)

		// 写入lua格式
		luaBuf = append(luaBuf, fmt.Sprintf("    \"%s\",\n", result.Token)...)

		count++

		// 每10000条或缓冲区满时写入文件
		if count%BatchSize == 0 || len(txtBuf) > 512*1024 {
			txtFile.Write(txtBuf)
			luaFile.Write(luaBuf)
			txtBuf = txtBuf[:0]
			luaBuf = luaBuf[:0]
		}
	}

	// 写入剩余数据
	if len(txtBuf) > 0 {
		txtFile.Write(txtBuf)
	}
	if len(luaBuf) > 0 {
		luaFile.Write(luaBuf)
	}

	// Lua文件结尾
	fmt.Fprintf(luaFile, "}\n\nreturn tokens\n")
}
