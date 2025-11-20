package messaging

import "context"

// MessageBus 消息总线接口
type MessageBus interface {
	// Publish 发布消息到主题
	Publish(ctx context.Context, topic string, message interface{}) error

	// Subscribe 订阅主题
	Subscribe(ctx context.Context, topic string, handler MessageHandler) error

	// Unsubscribe 取消订阅
	Unsubscribe(topic string) error

	// CreateTopic 创建主题（Kafka 需要）
	CreateTopic(ctx context.Context, topic string, partitions int, replicationFactor int) error

	// DeleteTopic 删除主题（Kafka 需要）
	DeleteTopic(ctx context.Context, topic string) error

	// Close 关闭连接
	Close() error
}

// MessageHandler 消息处理函数
type MessageHandler func(topic string, message []byte) error

// Message 消息结构
type Message struct {
	Type           string      `json:"type"`            // 消息类型
	RoomID         string      `json:"room_id"`         // 房间ID
	SourceInstance string      `json:"source_instance"` // 发布者实例ID
	Timestamp      int64       `json:"timestamp"`       // 时间戳
	Sequence       int64       `json:"sequence"`        // 消息序号（可选）
	Data           interface{} `json:"data"`            // 消息数据
}

// BusDeps 消息总线依赖
type BusDeps struct {
	Type          string   // "kafka" 或 "redis"
	Brokers       []string // Kafka brokers 或 Redis 地址
	TopicPrefix   string   // Topic 前缀
	ConsumerGroup string   // 消费者组名称
	InstanceID    string   // 实例ID（用于消息去重）
	// Kafka 配置
	ProducerAcks           string
	ProducerRetries        int
	BatchSize              int
	LingerMs               int
	CompressionType        string
	ConsumerAutoCommit     bool
	ConsumerMaxPollRecords int
	FetchMinBytes          int
	FetchMaxWaitMs         int
}

// NewMessageBus 创建消息总线（工厂方法）
func NewMessageBus(deps BusDeps) (MessageBus, error) {
	switch deps.Type {
	case "kafka":
		return NewKafkaBus(deps)
	case "redis":
		return NewRedisBus(deps)
	default:
		return nil, nil // 返回 nil 表示不使用消息总线
	}
}
