package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config 全局配置
type Config struct {
	Server ServerConfig `yaml:"server"`
	GRPC   GRPCConfig   `yaml:"grpc"`
	Log    LogConfig    `yaml:"log"`
}

// ServerConfig HTTP Server 配置
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"`
}

// GetHTTPAddr 获取 HTTP Server 地址
func (s *ServerConfig) GetHTTPAddr() string {
	return s.Host + ":" + strconv.Itoa(s.Port)
}

// GRPCConfig gRPC Client 配置
type GRPCConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// GetAddr 获取 gRPC Server 地址
func (g *GRPCConfig) GetAddr() string {
	return g.Host + ":" + strconv.Itoa(g.Port)
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
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

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
