package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/kaifa/game-platform/internal/logger"
	"go.uber.org/zap"
)

// KafkaBus Kafka 消息总线实现
type KafkaBus struct {
	producer       sarama.SyncProducer
	consumer       sarama.ConsumerGroup
	consumerConfig *sarama.Config
	consumerGroup  string
	brokers        []string
	topicPrefix    string
	instanceID     string
	subscriptions  map[string]MessageHandler
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

// NewKafkaBus 创建 Kafka 消息总线
func NewKafkaBus(deps BusDeps) (MessageBus, error) {
	// 配置 Producer
	producerConfig := sarama.NewConfig()
	producerConfig.Producer.Return.Successes = true
	producerConfig.Producer.Return.Errors = true

	// 设置 acks
	switch deps.ProducerAcks {
	case "0":
		producerConfig.Producer.RequiredAcks = sarama.NoResponse
	case "1":
		producerConfig.Producer.RequiredAcks = sarama.WaitForLocal
	case "all":
		producerConfig.Producer.RequiredAcks = sarama.WaitForAll
	default:
		producerConfig.Producer.RequiredAcks = sarama.WaitForAll
	}

	producerConfig.Producer.Retry.Max = deps.ProducerRetries
	producerConfig.Producer.Flush.Bytes = deps.BatchSize
	producerConfig.Producer.Flush.Frequency = time.Duration(deps.LingerMs) * time.Millisecond

	// 设置压缩
	switch deps.CompressionType {
	case "gzip":
		producerConfig.Producer.Compression = sarama.CompressionGZIP
	case "snappy":
		producerConfig.Producer.Compression = sarama.CompressionSnappy
	case "lz4":
		producerConfig.Producer.Compression = sarama.CompressionLZ4
	default:
		producerConfig.Producer.Compression = sarama.CompressionNone
	}

	// 创建 Producer
	producer, err := sarama.NewSyncProducer(deps.Brokers, producerConfig)
	if err != nil {
		return nil, fmt.Errorf("创建 Kafka Producer 失败: %w", err)
	}

	// 配置 Consumer
	consumerConfig := sarama.NewConfig()
	consumerConfig.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	// 对于跨实例广播消息，使用 OffsetNewest 避免重复消费历史消息
	// 如果需要处理历史消息，可以在订阅时指定
	consumerConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	consumerConfig.Consumer.Return.Errors = true

	// 设置自动提交
	if deps.ConsumerAutoCommit {
		consumerConfig.Consumer.Offsets.AutoCommit.Enable = true
		consumerConfig.Consumer.Offsets.AutoCommit.Interval = 1 * time.Second
	} else {
		consumerConfig.Consumer.Offsets.AutoCommit.Enable = false
	}

	// 设置拉取配置
	consumerConfig.Consumer.Fetch.Min = int32(deps.FetchMinBytes)
	consumerConfig.Consumer.Fetch.Default = 1024 * 1024 // 1MB
	consumerConfig.Consumer.MaxProcessingTime = time.Duration(deps.FetchMaxWaitMs) * time.Millisecond

	// 创建 Consumer Group
	// 注意：为了确保所有实例都能收到同一条消息（用于大厅广播等场景），
	// 每个实例使用不同的 ConsumerGroup（包含 instanceID）
	// 同时设置 offset 为 OffsetNewest，只消费新消息，避免重复消费历史消息
	consumerGroupName := fmt.Sprintf("%s-%s", deps.ConsumerGroup, deps.InstanceID)
	consumer, err := sarama.NewConsumerGroup(deps.Brokers, consumerGroupName, consumerConfig)
	if err != nil {
		producer.Close()
		return nil, fmt.Errorf("创建 Kafka Consumer Group 失败: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	bus := &KafkaBus{
		producer:       producer,
		consumer:       consumer,
		consumerConfig: consumerConfig,
		consumerGroup:  deps.ConsumerGroup,
		brokers:        deps.Brokers,
		topicPrefix:    deps.TopicPrefix,
		instanceID:     deps.InstanceID,
		subscriptions:  make(map[string]MessageHandler),
		ctx:            ctx,
		cancel:         cancel,
	}

	// 启动消费者错误处理
	go bus.handleConsumerErrors()

	logger.Logger.Info("Kafka 消息总线已创建",
		zap.Strings("brokers", deps.Brokers),
		zap.String("consumer_group_base", deps.ConsumerGroup),
		zap.String("consumer_group_actual", consumerGroupName),
		zap.String("instance_id", deps.InstanceID),
		zap.String("offset_initial", "newest"),
		zap.String("note", "每个实例使用独立的 ConsumerGroup，确保所有实例都能收到消息；OffsetNewest 避免重复消费历史消息"),
	)

	return bus, nil
}

// Publish 发布消息
func (b *KafkaBus) Publish(ctx context.Context, topic string, message interface{}) error {
	// 构建消息
	msg := map[string]interface{}{
		"source_instance": b.instanceID,
		"timestamp":       time.Now().Unix(),
		"data":            message,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	// 构建完整 Topic 名称
	fullTopic := b.getFullTopicName(topic)

	// 发布消息
	_, _, err = b.producer.SendMessage(&sarama.ProducerMessage{
		Topic: fullTopic,
		Value: sarama.ByteEncoder(data),
	})

	if err != nil {
		return fmt.Errorf("发布消息失败: %w", err)
	}

	logger.Logger.Debug("消息已发布到 Kafka",
		zap.String("topic", fullTopic),
		zap.String("instance_id", b.instanceID),
	)

	return nil
}

// Subscribe 订阅主题
func (b *KafkaBus) Subscribe(ctx context.Context, topic string, handler MessageHandler) error {
	b.mu.Lock()
	b.subscriptions[topic] = handler
	b.mu.Unlock()

	// 启动消费者（如果还没有启动）
	fullTopic := b.getFullTopicName(topic)

	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		b.consumeTopic(fullTopic, handler)
	}()

	logger.Logger.Info("已订阅 Kafka Topic",
		zap.String("topic", fullTopic),
		zap.String("instance_id", b.instanceID),
	)

	return nil
}

// consumeTopic 消费指定 Topic（使用 ConsumerGroup）
func (b *KafkaBus) consumeTopic(topic string, handler MessageHandler) {
	// 创建消费者组处理器
	consumerHandler := &consumerGroupHandler{
		bus:     b,
		topic:   topic,
		handler: handler,
	}

	// 启动消费者组
	for {
		select {
		case <-b.ctx.Done():
			return
		default:
			// 消费消息
			err := b.consumer.Consume(b.ctx, []string{topic}, consumerHandler)
			if err != nil {
				logger.Logger.Error("消费消息失败",
					zap.String("topic", topic),
					zap.Error(err),
				)
				time.Sleep(5 * time.Second) // 等待后重试
			}
		}
	}
}

// consumerGroupHandler ConsumerGroup 处理器
type consumerGroupHandler struct {
	bus     *KafkaBus
	topic   string
	handler MessageHandler
}

// Setup 会话开始
func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup 会话结束
func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim 消费消息
func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			// 解析消息
			var msg map[string]interface{}
			if err := json.Unmarshal(message.Value, &msg); err != nil {
				logger.Logger.Error("解析消息失败",
					zap.String("topic", message.Topic),
					zap.Error(err),
				)
				session.MarkMessage(message, "")
				continue
			}

			// 检查是否是自己的消息（避免重复处理）
			// source_instance 在包装消息的顶层
			sourceInstance, ok := msg["source_instance"].(string)
			if ok && sourceInstance == h.bus.instanceID {
				logger.Logger.Debug("忽略自己发布的消息",
					zap.String("topic", message.Topic),
					zap.String("instance_id", h.bus.instanceID),
					zap.String("source_instance", sourceInstance),
				)
				session.MarkMessage(message, "")
				continue
			}

			// 调用处理函数
			if err := h.handler(message.Topic, message.Value); err != nil {
				logger.Logger.Error("处理消息失败",
					zap.String("topic", message.Topic),
					zap.Error(err),
				)
			}

			// 标记消息已处理（手动提交模式下）
			if !h.bus.consumerConfig.Consumer.Offsets.AutoCommit.Enable {
				session.MarkMessage(message, "")
			}

		case <-session.Context().Done():
			return nil
		}
	}
}

// handleConsumerErrors 处理消费者错误
func (b *KafkaBus) handleConsumerErrors() {
	for {
		select {
		case err := <-b.consumer.Errors():
			if err != nil {
				logger.Logger.Error("Kafka Consumer 错误",
					zap.Error(err),
				)
			}
		case <-b.ctx.Done():
			return
		}
	}
}

// Unsubscribe 取消订阅
func (b *KafkaBus) Unsubscribe(topic string) error {
	b.mu.Lock()
	delete(b.subscriptions, topic)
	b.mu.Unlock()

	logger.Logger.Info("已取消订阅 Kafka Topic",
		zap.String("topic", topic),
	)

	return nil
}

// CreateTopic 创建 Topic
func (b *KafkaBus) CreateTopic(ctx context.Context, topic string, partitions int, replicationFactor int) error {
	fullTopic := b.getFullTopicName(topic)

	// 使用 Admin API 创建 Topic
	admin, err := sarama.NewClusterAdmin(b.brokers, sarama.NewConfig())
	if err != nil {
		return fmt.Errorf("创建 Kafka Admin 失败: %w", err)
	}
	defer admin.Close()

	topicDetail := &sarama.TopicDetail{
		NumPartitions:     int32(partitions),
		ReplicationFactor: int16(replicationFactor),
		ConfigEntries: map[string]*string{
			"retention.ms": stringPtr("604800000"), // 7天
		},
	}

	err = admin.CreateTopic(fullTopic, topicDetail, false)
	if err != nil {
		// Topic 可能已存在，忽略错误
		if err == sarama.ErrTopicAlreadyExists {
			logger.Logger.Debug("Topic 已存在",
				zap.String("topic", fullTopic),
			)
			return nil
		}
		return fmt.Errorf("创建 Topic 失败: %w", err)
	}

	logger.Logger.Info("Topic 已创建",
		zap.String("topic", fullTopic),
		zap.Int("partitions", partitions),
		zap.Int("replication_factor", replicationFactor),
	)

	return nil
}

// DeleteTopic 删除 Topic
func (b *KafkaBus) DeleteTopic(ctx context.Context, topic string) error {
	fullTopic := b.getFullTopicName(topic)

	admin, err := sarama.NewClusterAdmin(b.brokers, sarama.NewConfig())
	if err != nil {
		return fmt.Errorf("创建 Kafka Admin 失败: %w", err)
	}
	defer admin.Close()

	err = admin.DeleteTopic(fullTopic)
	if err != nil {
		return fmt.Errorf("删除 Topic 失败: %w", err)
	}

	logger.Logger.Info("Topic 已删除",
		zap.String("topic", fullTopic),
	)

	return nil
}

// Close 关闭连接
func (b *KafkaBus) Close() error {
	b.cancel()

	// 等待所有消费者退出
	b.wg.Wait()

	// 关闭 Producer
	if b.producer != nil {
		if err := b.producer.Close(); err != nil {
			logger.Logger.Error("关闭 Kafka Producer 失败", zap.Error(err))
		}
	}

	// 关闭 Consumer
	if b.consumer != nil {
		if err := b.consumer.Close(); err != nil {
			logger.Logger.Error("关闭 Kafka Consumer 失败", zap.Error(err))
		}
	}

	logger.Logger.Info("Kafka 消息总线已关闭")

	return nil
}

// getFullTopicName 获取完整的 Topic 名称
func (b *KafkaBus) getFullTopicName(topic string) string {
	if b.topicPrefix != "" {
		return fmt.Sprintf("%s-%s", b.topicPrefix, topic)
	}
	return topic
}

// GetFullTopicName 公开方法，用于获取完整的 Topic 名称（用于日志）
func (b *KafkaBus) GetFullTopicName(topic string) string {
	return b.getFullTopicName(topic)
}

// stringPtr 返回字符串指针
func stringPtr(s string) *string {
	return &s
}
