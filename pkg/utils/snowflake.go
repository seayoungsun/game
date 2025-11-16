package utils

import (
	"errors"
	"sync"
	"time"
)

// Snowflake ID 生成器
// 64位结构：
// | 1位符号位 | 41位时间戳 | 10位机器ID | 12位序列号 |
// |    0     |  毫秒时间  |  0-1023   |  0-4095   |

const (
	epoch          int64 = 1640995200000                // 起始时间戳（2022-01-01 00:00:00 UTC，毫秒）
	machineIDBits  uint8 = 10                           // 机器ID位数（支持1024个节点）
	sequenceBits   uint8 = 12                           // 序列号位数（每毫秒4096个ID）
	machineIDMax   int64 = -1 ^ (-1 << machineIDBits)   // 机器ID最大值：1023
	sequenceMask   int64 = -1 ^ (-1 << sequenceBits)    // 序列号掩码：4095
	machineIDShift uint8 = sequenceBits                 // 机器ID左移位数：12
	timestampShift uint8 = sequenceBits + machineIDBits // 时间戳左移位数：22
)

// SnowflakeGenerator 雪花算法ID生成器
type SnowflakeGenerator struct {
	mu        sync.Mutex
	machineID int64 // 机器ID（0-1023）
	sequence  int64 // 当前序列号（0-4095）
	lastTime  int64 // 上次生成ID的时间戳（毫秒）
}

// 全局雪花算法生成器
var (
	globalSnowflake *SnowflakeGenerator
	snowflakeOnce   sync.Once
)

// InitSnowflake 初始化全局雪花算法生成器
// machineID: 机器ID（0-1023），多实例部署时每个实例应使用不同的ID
func InitSnowflake(machineID int64) error {
	if machineID < 0 || machineID > machineIDMax {
		return errors.New("机器ID必须在0-1023之间")
	}

	snowflakeOnce.Do(func() {
		globalSnowflake = &SnowflakeGenerator{
			machineID: machineID,
			sequence:  0,
			lastTime:  0,
		}
	})

	return nil
}

// GetSnowflakeGenerator 获取全局雪花算法生成器
func GetSnowflakeGenerator() *SnowflakeGenerator {
	return globalSnowflake
}

// GenerateID 生成唯一ID
func (s *SnowflakeGenerator) GenerateID() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 获取当前时间戳（毫秒）
	now := time.Now().UnixMilli()

	if now < s.lastTime {
		// 时钟回退，拒绝生成（保证ID单调递增）
		return 0, errors.New("时钟回退，拒绝生成ID")
	}

	if now == s.lastTime {
		// 同一毫秒内生成多个ID
		s.sequence = (s.sequence + 1) & sequenceMask
		if s.sequence == 0 {
			// 序列号用完，等待下一毫秒
			for now <= s.lastTime {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		// 新的毫秒，序列号重置为0
		s.sequence = 0
	}

	s.lastTime = now

	// 组装ID
	// | 时间戳(41位) | 机器ID(10位) | 序列号(12位) |
	id := ((now - epoch) << timestampShift) | // 时间戳部分
		(s.machineID << machineIDShift) | // 机器ID部分
		s.sequence // 序列号部分

	return id, nil
}

// ParseSnowflakeID 解析雪花算法ID（用于调试）
func ParseSnowflakeID(id int64) map[string]interface{} {
	// 提取时间戳
	timestamp := (id >> timestampShift) + epoch
	t := time.UnixMilli(timestamp)

	// 提取机器ID
	machineID := (id >> machineIDShift) & machineIDMax

	// 提取序列号
	sequence := id & sequenceMask

	return map[string]interface{}{
		"id":         id,
		"timestamp":  timestamp,
		"time":       t.Format("2006-01-02 15:04:05.000"),
		"machine_id": machineID,
		"sequence":   sequence,
	}
}
