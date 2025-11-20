package discovery

import (
	"context"
	"time"
)

// ServiceInstance 服务实例信息
type ServiceInstance struct {
	ServiceName   string            `json:"service_name"`   // 服务名称
	InstanceID    string            `json:"instance_id"`    // 实例ID（唯一）
	Address       string            `json:"address"`        // IP 地址
	Port          int               `json:"port"`           // 端口
	Meta          map[string]string `json:"meta,omitempty"` // 元数据
	RegisteredAt  int64             `json:"registered_at"`  // 注册时间戳
	LastHeartbeat int64             `json:"last_heartbeat"` // 最后心跳时间（Consul 不需要）
}

// Registry 服务注册与发现接口
type Registry interface {
	// Register 注册服务实例
	Register(ctx context.Context, instance ServiceInstance) error

	// Deregister 注销服务实例
	Deregister(ctx context.Context, instanceID string) error

	// KeepAlive 启动心跳保活（返回停止函数）
	// 注意：Consul 使用健康检查，此方法可能返回空函数
	KeepAlive(ctx context.Context, instanceID string) (stop func(), err error)

	// ListInstances 列出所有实例
	ListInstances(ctx context.Context, serviceName string) ([]ServiceInstance, error)

	// GetInstance 获取指定实例
	GetInstance(ctx context.Context, instanceID string) (ServiceInstance, error)

	// IsInstanceAlive 检查实例是否存活
	IsInstanceAlive(ctx context.Context, instanceID string) (bool, error)
}

// RegistryDeps 注册器依赖
type RegistryDeps struct {
	Type                string // "consul" 或 "redis"
	ConsulAddr          string
	Redis               interface{} // *redis.Client
	ServiceName         string
	InstanceID          string
	InstanceAddress     string
	InstancePort        int
	HealthCheckURL      string
	HealthCheckInterval time.Duration
	HealthCheckTimeout  time.Duration
	DeregisterAfter     time.Duration
	// Redis 相关
	InstanceTTL       time.Duration
	HeartbeatInterval time.Duration
}

// NewRegistry 创建注册器（工厂方法）
func NewRegistry(deps RegistryDeps) (Registry, error) {
	switch deps.Type {
	case "consul":
		return NewConsulRegistry(deps)
	case "redis":
		return NewRedisRegistry(deps)
	default:
		return nil, nil // 返回 nil 表示不使用服务发现
	}
}
