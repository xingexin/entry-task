package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"entry-task/tcpserver/config"
	log "entry-task/tcpserver/pkg/logger"
)

// Client Redis客户端接口
type Client interface {
	// Set 设置键值对
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error

	// Get 获取字符串值
	Get(ctx context.Context, key string) (string, error)

	// GetUint64 获取uint64类型的值
	GetUint64(ctx context.Context, key string) (uint64, error)

	// Del 删除一个或多个键
	Del(ctx context.Context, keys ...string) error

	// Exists 检查键是否存在
	Exists(ctx context.Context, keys ...string) (int64, error)

	// Expire 设置键的过期时间
	Expire(ctx context.Context, key string, expiration time.Duration) error

	// TTL 获取键的剩余生存时间
	TTL(ctx context.Context, key string) (time.Duration, error)

	// Incr 将键的值加1
	Incr(ctx context.Context, key string) (int64, error)

	// IncrBy 将键的值增加指定数值
	IncrBy(ctx context.Context, key string, value int64) (int64, error)

	// SetJSON 设置JSON格式的值
	SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error

	// GetJSON 获取JSON格式的值并反序列化
	GetJSON(ctx context.Context, key string, dest interface{}) error

	// Ping 测试Redis连接
	Ping(ctx context.Context) error

	// Close 关闭Redis连接
	Close() error
}

// redisClient Redis客户端实现
type redisClient struct {
	client *redis.Client
}

// InitRedis 初始化Redis连接
func InitRedis(cfg *config.Config) (Client, error) {
	log.Info("开始初始化Redis连接",
		zap.String("host", cfg.Redis.Host),
		zap.Int("port", cfg.Redis.Port),
		zap.Int("db", cfg.Redis.DB),
	)

	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.GetAddr(),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		MaxRetries:   cfg.Redis.MaxRetries,
		DialTimeout:  cfg.Redis.GetDialTimeout(),
		ReadTimeout:  cfg.Redis.GetReadTimeout(),
		WriteTimeout: cfg.Redis.GetWriteTimeout(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Error("Redis连接测试失败", zap.Error(err))
		return nil, fmt.Errorf("Redis连接失败: %w", err)
	}

	log.Info("Redis连接成功",
		zap.String("addr", cfg.Redis.GetAddr()),
		zap.Int("pool_size", cfg.Redis.PoolSize),
	)

	return &redisClient{client: client}, nil
}

// Set 设置键值对
func (r *redisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

// Get 获取字符串值
func (r *redisClient) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

// GetUint64 获取uint64类型的值
func (r *redisClient) GetUint64(ctx context.Context, key string) (uint64, error) {
	return r.client.Get(ctx, key).Uint64()
}

// Del 删除一个或多个键
func (r *redisClient) Del(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

// Exists 检查键是否存在
func (r *redisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Exists(ctx, keys...).Result()
}

// Expire 设置键的过期时间
func (r *redisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return r.client.Expire(ctx, key, expiration).Err()
}

// TTL 获取键的剩余生存时间
func (r *redisClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.client.TTL(ctx, key).Result()
}

// Incr 将键的值加1
func (r *redisClient) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

// IncrBy 将键的值增加指定数值
func (r *redisClient) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return r.client.IncrBy(ctx, key, value).Result()
}

// SetJSON 设置JSON格式的值
func (r *redisClient) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("JSON序列化失败: %w", err)
	}
	return r.Set(ctx, key, data, expiration)
}

// GetJSON 获取JSON格式的值并反序列化
func (r *redisClient) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := r.Get(ctx, key)
	if err != nil {
		// 直接返回redis.Nil，让调用者处理
		if err == redis.Nil {
			return redis.Nil
		}
		return err
	}
	if err := json.Unmarshal([]byte(data), dest); err != nil {
		return fmt.Errorf("JSON反序列化失败: %w", err)
	}
	return nil
}

// Ping 测试Redis连接
func (r *redisClient) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Close 关闭Redis连接
func (r *redisClient) Close() error {
	return r.client.Close()
}
