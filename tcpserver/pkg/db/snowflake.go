package db

import (
	"fmt"
	"sync"
	"time"
)

// Snowflake 雪花ID生成器
// 64位ID结构：
// 1位符号位(0) + 41位时间戳 + 10位机器ID + 12位序列号
// 0 - 0000000000 0000000000 0000000000 0000000000 0 - 0000000000 - 000000000000
type Snowflake struct {
	mu        sync.Mutex
	epoch     int64 // 起始时间戳（毫秒）2024-01-01 00:00:00
	timestamp int64 // 上次生成ID的时间戳
	machineID int64 // 机器ID (0-1023)
	sequence  int64 // 序列号 (0-4095)
}

const (
	// 时间戳位数
	timestampBits = 41
	// 机器ID位数
	machineIDBits = 10
	// 序列号位数
	sequenceBits = 12

	// 最大机器ID
	maxMachineID = -1 ^ (-1 << machineIDBits) // 1023
	// 最大序列号
	maxSequence = -1 ^ (-1 << sequenceBits) // 4095

	// 位移量
	machineIDShift = sequenceBits                 // 12
	timestampShift = sequenceBits + machineIDBits // 22
)

var (
	// 起始时间：2024-01-01 00:00:00 UTC
	epoch = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	// 全局单例
	defaultSnowflake *Snowflake
	once             sync.Once
)

// NewSnowflake 创建雪花ID生成器
// machineID: 机器ID (0-1023)，用于区分不同的服务实例
func NewSnowflake(machineID int64) (*Snowflake, error) {
	if machineID < 0 || machineID > maxMachineID {
		return nil, fmt.Errorf("machineID必须在0-%d之间，当前值：%d", maxMachineID, machineID)
	}

	return &Snowflake{
		epoch:     epoch,
		machineID: machineID,
		timestamp: 0,
		sequence:  0,
	}, nil
}

// NextID 生成下一个ID
func (s *Snowflake) NextID() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixMilli()

	// 时钟回拨检测
	if now < s.timestamp {
		return 0, fmt.Errorf("时钟回拨检测：当前时间 %d < 上次时间 %d", now, s.timestamp)
	}

	if now == s.timestamp {
		// 同一毫秒内，序列号递增
		s.sequence = (s.sequence + 1) & maxSequence
		if s.sequence == 0 {
			// 序列号溢出，等待下一毫秒
			now = s.waitNextMillis(s.timestamp)
		}
	} else {
		// 新的毫秒，序列号重置为0
		s.sequence = 0
	}

	s.timestamp = now

	// 计算相对时间戳（相对于epoch）
	diff := now - s.epoch
	if diff < 0 {
		return 0, fmt.Errorf("当前时间早于起始时间")
	}

	// 组装ID
	// [0][41位时间戳][10位机器ID][12位序列号]
	id := (diff << timestampShift) | (s.machineID << machineIDShift) | s.sequence

	return id, nil
}

// waitNextMillis 等待下一毫秒
func (s *Snowflake) waitNextMillis(lastTimestamp int64) int64 {
	now := time.Now().UnixMilli()
	for now <= lastTimestamp {
		time.Sleep(100 * time.Microsecond)
		now = time.Now().UnixMilli()
	}
	return now
}

// GetDefaultSnowflake 获取默认的雪花ID生成器（单例模式）
func GetDefaultSnowflake() *Snowflake {
	once.Do(func() {
		// 默认使用machineID=1
		// 生产环境应该从配置文件读取或根据IP/hostname生成
		defaultSnowflake, _ = NewSnowflake(1)
	})
	return defaultSnowflake
}

// GenerateID 生成下一个ID（使用默认生成器）
func GenerateID() (int64, error) {
	return GetDefaultSnowflake().NextID()
}

// ParseSnowflakeID 解析雪花ID，返回时间戳、机器ID、序列号
func ParseSnowflakeID(id int64) (timestamp int64, machineID int64, sequence int64) {
	sequence = id & maxSequence
	machineID = (id >> machineIDShift) & maxMachineID
	timestamp = (id >> timestampShift) + epoch
	return
}

// GetSnowflakeTime 获取雪花ID对应的时间
func GetSnowflakeTime(id int64) time.Time {
	timestamp, _, _ := ParseSnowflakeID(id)
	return time.UnixMilli(timestamp)
}
