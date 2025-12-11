package config

import (
	"os"
	"testing"
	"time"
)

// TestLoad 测试加载配置文件
func TestLoad(t *testing.T) {
	// 测试加载正常配置
	cfg, err := Load("config.yaml")
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证服务器配置
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host 期望 '0.0.0.0', 实际 '%s'", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port 期望 8080, 实际 %d", cfg.Server.Port)
	}
	if cfg.Server.Mode != "development" {
		t.Errorf("Server.Mode 期望 'development', 实际 '%s'", cfg.Server.Mode)
	}

	// 验证数据库配置
	if cfg.Database.Host != "192.168.215.4" {
		t.Errorf("Database.Host 期望 '192.168.215.4', 实际 '%s'", cfg.Database.Host)
	}
	if cfg.Database.Port != 3306 {
		t.Errorf("Database.Port 期望 3306, 实际 %d", cfg.Database.Port)
	}
	if cfg.Database.Username != "root" {
		t.Errorf("Database.Username 期望 'root', 实际 '%s'", cfg.Database.Username)
	}
	if cfg.Database.Password != "root" {
		t.Errorf("Database.Password 期望 'root', 实际 '%s'", cfg.Database.Password)
	}
	if cfg.Database.Database != "entrytask" {
		t.Errorf("Database.Database 期望 'entrytask', 实际 '%s'", cfg.Database.Database)
	}
	if cfg.Database.MaxOpenConns != 100 {
		t.Errorf("Database.MaxOpenConns 期望 100, 实际 %d", cfg.Database.MaxOpenConns)
	}

	// 验证Redis配置
	if cfg.Redis.Host != "192.168.215.2" {
		t.Errorf("Redis.Host 期望 '192.168.215.2', 实际 '%s'", cfg.Redis.Host)
	}
	if cfg.Redis.Port != 6379 {
		t.Errorf("Redis.Port 期望 6379, 实际 %d", cfg.Redis.Port)
	}
	if cfg.Redis.PoolSize != 10 {
		t.Errorf("Redis.PoolSize 期望 10, 实际 %d", cfg.Redis.PoolSize)
	}

	// 验证雪花ID配置
	if cfg.Snowflake.MachineID != 1 {
		t.Errorf("Snowflake.MachineID 期望 1, 实际 %d", cfg.Snowflake.MachineID)
	}

	// 验证日志配置
	if cfg.Log.Level != "info" {
		t.Errorf("Log.Level 期望 'info', 实际 '%s'", cfg.Log.Level)
	}
}

// TestLoadFileNotExist 测试加载不存在的配置文件
func TestLoadFileNotExist(t *testing.T) {
	_, err := Load("not_exist.yaml")
	if err == nil {
		t.Error("期望返回错误，但没有返回")
	}
}

// TestLoadInvalidYAML 测试加载无效的YAML文件
func TestLoadInvalidYAML(t *testing.T) {
	// 创建临时的无效YAML文件
	invalidYAML := `
server:
  host: "localhost"
  port: invalid_port  # 这是无效的
`
	tmpFile, err := os.CreateTemp("", "invalid_*.yaml")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(invalidYAML); err != nil {
		t.Fatalf("写入临时文件失败: %v", err)
	}
	tmpFile.Close()

	_, err = Load(tmpFile.Name())
	if err == nil {
		t.Error("期望返回错误，但没有返回")
	}
}

// TestDatabaseGetDSN 测试获取数据库DSN
func TestDatabaseGetDSN(t *testing.T) {
	dbConfig := DatabaseConfig{
		Host:      "192.168.215.4",
		Port:      3306,
		Username:  "root",
		Password:  "root",
		Database:  "entrytask",
		Charset:   "utf8mb4",
		ParseTime: true,
		Loc:       "Local",
	}

	expectedDSN := "root:root@tcp(192.168.215.4:3306)/entrytask?charset=utf8mb4&parseTime=true&loc=Local"
	actualDSN := dbConfig.GetDSN()

	if actualDSN != expectedDSN {
		t.Errorf("DSN不匹配\n期望: %s\n实际: %s", expectedDSN, actualDSN)
	}
}

// TestRedisGetAddr 测试获取Redis地址
func TestRedisGetAddr(t *testing.T) {
	redisConfig := RedisConfig{
		Host: "192.168.215.2",
		Port: 6379,
	}

	expectedAddr := "192.168.215.2:6379"
	actualAddr := redisConfig.GetAddr()

	if actualAddr != expectedAddr {
		t.Errorf("Redis地址不匹配\n期望: %s\n实际: %s", expectedAddr, actualAddr)
	}
}

// TestRedisGetTimeouts 测试获取Redis超时配置
func TestRedisGetTimeouts(t *testing.T) {
	redisConfig := RedisConfig{
		DialTimeout:  5,
		ReadTimeout:  3,
		WriteTimeout: 3,
	}

	// 测试连接超时
	dialTimeout := redisConfig.GetDialTimeout()
	if dialTimeout != 5*time.Second {
		t.Errorf("DialTimeout 期望 5s, 实际 %v", dialTimeout)
	}

	// 测试读超时
	readTimeout := redisConfig.GetReadTimeout()
	if readTimeout != 3*time.Second {
		t.Errorf("ReadTimeout 期望 3s, 实际 %v", readTimeout)
	}

	// 测试写超时
	writeTimeout := redisConfig.GetWriteTimeout()
	if writeTimeout != 3*time.Second {
		t.Errorf("WriteTimeout 期望 3s, 实际 %v", writeTimeout)
	}
}

// TestGetGlobalConfig 测试全局配置
func TestGetGlobalConfig(t *testing.T) {
	// 重置全局配置
	globalConfig = nil

	// 测试未初始化时获取配置应该panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("期望panic，但没有发生")
		}
	}()

	Get()
}

// TestGetHelpers 测试配置辅助函数
func TestGetHelpers(t *testing.T) {
	// 加载配置
	_, err := Load("config.yaml")
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 测试 GetDatabase
	dbConfig := GetDatabase()
	if dbConfig == nil {
		t.Error("GetDatabase 返回 nil")
	}
	if dbConfig.Database != "entrytask" {
		t.Errorf("GetDatabase 返回的数据库名不正确: %s", dbConfig.Database)
	}

	// 测试 GetRedis
	redisConfig := GetRedis()
	if redisConfig == nil {
		t.Error("GetRedis 返回 nil")
	}
	if redisConfig.Host != "192.168.215.2" {
		t.Errorf("GetRedis 返回的主机不正确: %s", redisConfig.Host)
	}

	// 测试 GetSnowflake
	snowflakeConfig := GetSnowflake()
	if snowflakeConfig == nil {
		t.Error("GetSnowflake 返回 nil")
	}
	if snowflakeConfig.MachineID != 1 {
		t.Errorf("GetSnowflake 返回的机器ID不正确: %d", snowflakeConfig.MachineID)
	}
}

// BenchmarkLoad 性能测试：加载配置
func BenchmarkLoad(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = Load("config.yaml")
	}
}

// BenchmarkGetDSN 性能测试：获取DSN
func BenchmarkGetDSN(b *testing.B) {
	dbConfig := DatabaseConfig{
		Host:      "192.168.215.4",
		Port:      3306,
		Username:  "root",
		Password:  "root",
		Database:  "entrytask",
		Charset:   "utf8mb4",
		ParseTime: true,
		Loc:       "Local",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dbConfig.GetDSN()
	}
}

