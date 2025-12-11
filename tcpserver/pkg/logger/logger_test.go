package logger

import (
	"os"
	"strings"
	"testing"

	"go.uber.org/zap"
)

// TestInit 测试日志初始化
func TestInit(t *testing.T) {
	cfg := &Config{
		Level:  "info",
		Output: "stdout",
	}

	err := Init(cfg)
	if err != nil {
		t.Fatalf("初始化日志失败: %v", err)
	}

	if Logger == nil {
		t.Error("Logger 未初始化")
	}

	if Sugar == nil {
		t.Error("Sugar 未初始化")
	}
}

// TestInitWithDifferentLevels 测试不同日志级别初始化
func TestInitWithDifferentLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}

	for _, level := range levels {
		cfg := &Config{
			Level:  level,
			Output: "stdout",
		}

		err := Init(cfg)
		if err != nil {
			t.Errorf("初始化日志级别 %s 失败: %v", level, err)
		}
	}
}

// TestInitWithFile 测试文件输出
func TestInitWithFile(t *testing.T) {
	// 创建临时日志文件
	tmpFile := "./test_logger.log"
	defer os.Remove(tmpFile)

	cfg := &Config{
		Level:    "info",
		Output:   "file",
		FilePath: tmpFile,
	}

	err := Init(cfg)
	if err != nil {
		t.Fatalf("初始化文件日志失败: %v", err)
	}

	// 写入日志
	Info("测试日志", zap.String("key", "value"))
	Sync()

	// 验证文件是否创建
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("日志文件未创建")
	}

	// 读取文件内容
	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("读取日志文件失败: %v", err)
	}

	if !strings.Contains(string(content), "测试日志") {
		t.Error("日志文件中未找到预期内容")
	}
}

// TestLogLevels 测试各个日志级别
func TestLogLevels(t *testing.T) {
	// 重新初始化为 debug 级别，这样所有级别都会输出
	cfg := &Config{
		Level:  "debug",
		Output: "stdout",
	}
	Init(cfg)

	// 测试各个级别（只验证不会panic）
	tests := []struct {
		name string
		fn   func()
	}{
		{"Debug", func() { Debug("debug message", zap.String("key", "value")) }},
		{"Info", func() { Info("info message", zap.String("key", "value")) }},
		{"Warn", func() { Warn("warn message", zap.String("key", "value")) }},
		{"Error", func() { Error("error message", zap.String("key", "value")) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 只要不panic就算通过
			tt.fn()
		})
	}
}

// TestInfoLog 测试Info日志
func TestInfoLog(t *testing.T) {
	cfg := &Config{
		Level:  "info",
		Output: "stdout",
	}
	Init(cfg)

	// 测试不会panic
	Info("测试Info日志")
	Info("测试Info日志带字段", zap.String("username", "zhangsan"), zap.Int("age", 25))
}

// TestWarnLog 测试Warn日志
func TestWarnLog(t *testing.T) {
	cfg := &Config{
		Level:  "warn",
		Output: "stdout",
	}
	Init(cfg)

	Warn("测试Warn日志")
	Warn("测试Warn日志带字段", zap.String("reason", "测试原因"))
}

// TestErrorLog 测试Error日志
func TestErrorLog(t *testing.T) {
	cfg := &Config{
		Level:  "error",
		Output: "stdout",
	}
	Init(cfg)

	Error("测试Error日志")
	Error("测试Error日志带字段", zap.String("error", "something wrong"))
}

// TestDebugLog 测试Debug日志
func TestDebugLog(t *testing.T) {
	cfg := &Config{
		Level:  "debug",
		Output: "stdout",
	}
	Init(cfg)

	Debug("测试Debug日志")
	Debug("测试Debug日志带字段", zap.String("data", "debug data"))
}

// TestSugarLogger 测试SugaredLogger
func TestSugarLogger(t *testing.T) {
	cfg := &Config{
		Level:  "info",
		Output: "stdout",
	}
	Init(cfg)

	// 测试Sugar的各种方法
	Sugar.Info("Sugar Info")
	Sugar.Infof("Sugar Infof: %s", "test")
	Sugar.Infow("Sugar Infow", "key", "value")
	
	Sugar.Warn("Sugar Warn")
	Sugar.Error("Sugar Error")
	Sugar.Debug("Sugar Debug")
}

// TestSync 测试日志同步
func TestSync(t *testing.T) {
	cfg := &Config{
		Level:  "info",
		Output: "stdout",
	}
	Init(cfg)

	Info("测试日志")
	
	// 测试Sync不会panic
	Sync()
}

// TestSyncWithNilLogger 测试Logger为nil时的Sync
func TestSyncWithNilLogger(t *testing.T) {
	Logger = nil
	
	// 应该不会panic
	Sync()
}

// TestLogWithMultipleFields 测试带多个字段的日志
func TestLogWithMultipleFields(t *testing.T) {
	cfg := &Config{
		Level:  "info",
		Output: "stdout",
	}
	Init(cfg)

	Info("用户登录",
		zap.String("username", "zhangsan"),
		zap.String("ip", "192.168.1.100"),
		zap.Int("port", 8080),
		zap.Bool("success", true),
		zap.Duration("duration", 100),
	)
}

// BenchmarkInfo 性能测试：Info日志
func BenchmarkInfo(b *testing.B) {
	cfg := &Config{
		Level:  "info",
		Output: "stdout",
	}
	Init(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("benchmark test", zap.Int("iteration", i))
	}
}

// BenchmarkSugarInfo 性能测试：Sugar Info日志
func BenchmarkSugarInfo(b *testing.B) {
	cfg := &Config{
		Level:  "info",
		Output: "stdout",
	}
	Init(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Sugar.Infow("benchmark test", "iteration", i)
	}
}

// BenchmarkInfoWithFields 性能测试：带多个字段的Info日志
func BenchmarkInfoWithFields(b *testing.B) {
	cfg := &Config{
		Level:  "info",
		Output: "stdout",
	}
	Init(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("benchmark test",
			zap.String("username", "zhangsan"),
			zap.Int("age", 25),
			zap.Bool("active", true),
		)
	}
}

