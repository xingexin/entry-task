package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 全局配置
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	Redis     RedisConfig     `yaml:"redis"`
	Snowflake SnowflakeConfig `yaml:"snowflake"`
	Log       LogConfig       `yaml:"log"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver          string `yaml:"driver"`            // 数据库驱动: mysql, postgres
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	Username        string `yaml:"username"`
	Password        string `yaml:"password"`
	Database        string `yaml:"database"`
	Charset         string `yaml:"charset"`
	ParseTime       bool   `yaml:"parse_time"`
	Loc             string `yaml:"loc"`
	MaxOpenConns    int    `yaml:"max_open_conns"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	ConnMaxLifetime int    `yaml:"conn_max_lifetime"` // 秒
}

// GetDSN 获取数据库连接字符串
func (d *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
		d.Username,
		d.Password,
		d.Host,
		d.Port,
		d.Database,
		d.Charset,
		d.ParseTime,
		d.Loc,
	)
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	Password     string `yaml:"password"`
	DB           int    `yaml:"db"`
	PoolSize     int    `yaml:"pool_size"`
	MinIdleConns int    `yaml:"min_idle_conns"`
	MaxRetries   int    `yaml:"max_retries"`
	DialTimeout  int    `yaml:"dial_timeout"`  // 秒
	ReadTimeout  int    `yaml:"read_timeout"`  // 秒
	WriteTimeout int    `yaml:"write_timeout"` // 秒
}

// GetAddr 获取Redis地址
func (r *RedisConfig) GetAddr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// GetDialTimeout 获取连接超时时间
func (r *RedisConfig) GetDialTimeout() time.Duration {
	return time.Duration(r.DialTimeout) * time.Second
}

// GetReadTimeout 获取读超时时间
func (r *RedisConfig) GetReadTimeout() time.Duration {
	return time.Duration(r.ReadTimeout) * time.Second
}

// GetWriteTimeout 获取写超时时间
func (r *RedisConfig) GetWriteTimeout() time.Duration {
	return time.Duration(r.WriteTimeout) * time.Second
}

// SnowflakeConfig 雪花ID配置
type SnowflakeConfig struct {
	MachineID int64 `yaml:"machine_id"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level    string `yaml:"level"`
	Output   string `yaml:"output"`
	FilePath string `yaml:"file_path"`
}

var globalConfig *Config

// Load 加载配置文件
func Load(configPath string) (*Config, error) {
	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 保存到全局变量
	globalConfig = &config

	return &config, nil
}

// Get 获取全局配置
func Get() *Config {
	if globalConfig == nil {
		panic("配置未初始化，请先调用 Load()")
	}
	return globalConfig
}

// GetDatabase 获取数据库配置
func GetDatabase() *DatabaseConfig {
	return &Get().Database
}

// GetRedis 获取Redis配置
func GetRedis() *RedisConfig {
	return &Get().Redis
}

// GetSnowflake 获取雪花ID配置
func GetSnowflake() *SnowflakeConfig {
	return &Get().Snowflake
}
