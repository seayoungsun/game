package discovery

import (
	"context"
	"fmt"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/kaifa/game-platform/internal/logger"
	"go.uber.org/zap"
)

// ConsulRegistry Consul 服务注册实现
type ConsulRegistry struct {
	client          *consulapi.Client
	agent           *consulapi.Agent
	serviceName     string
	instanceID      string
	instanceAddress string
	instancePort    int
	healthCheck     *consulapi.AgentServiceCheck
}

// NewConsulRegistry 创建 Consul 注册器
func NewConsulRegistry(deps RegistryDeps) (*ConsulRegistry, error) {
	// 创建 Consul 客户端
	config := consulapi.DefaultConfig()
	config.Address = deps.ConsulAddr

	client, err := consulapi.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("创建 Consul 客户端失败: %w", err)
	}

	// 构建健康检查配置
	healthCheck := &consulapi.AgentServiceCheck{
		HTTP:                           deps.HealthCheckURL,
		Interval:                       deps.HealthCheckInterval.String(),
		Timeout:                        deps.HealthCheckTimeout.String(),
		DeregisterCriticalServiceAfter: deps.DeregisterAfter.String(),
	}

	return &ConsulRegistry{
		client:          client,
		agent:           client.Agent(),
		serviceName:     deps.ServiceName,
		instanceID:      deps.InstanceID,
		instanceAddress: deps.InstanceAddress,
		instancePort:    deps.InstancePort,
		healthCheck:     healthCheck,
	}, nil
}

// Register 注册服务实例
func (r *ConsulRegistry) Register(ctx context.Context, instance ServiceInstance) error {
	// 构建服务注册信息
	registration := &consulapi.AgentServiceRegistration{
		ID:      instance.InstanceID,
		Name:    instance.ServiceName,
		Tags:    []string{"v1.0", "production"},
		Address: instance.Address,
		Port:    instance.Port,
		Check:   r.healthCheck,
		Meta:    instance.Meta,
	}

	// 注册服务
	err := r.agent.ServiceRegister(registration)
	if err != nil {
		return fmt.Errorf("注册服务失败: %w", err)
	}

	logger.Logger.Info("服务已注册到 Consul",
		zap.String("service", r.serviceName),
		zap.String("instance_id", instance.InstanceID),
		zap.String("address", instance.Address),
		zap.Int("port", instance.Port),
	)

	return nil
}

// Deregister 注销服务实例
func (r *ConsulRegistry) Deregister(ctx context.Context, instanceID string) error {
	err := r.agent.ServiceDeregister(instanceID)
	if err != nil {
		return fmt.Errorf("注销服务失败: %w", err)
	}

	logger.Logger.Info("服务已从 Consul 注销",
		zap.String("service", r.serviceName),
		zap.String("instance_id", instanceID),
	)

	return nil
}

// KeepAlive 启动心跳保活（Consul 使用健康检查，无需额外心跳）
func (r *ConsulRegistry) KeepAlive(ctx context.Context, instanceID string) (stop func(), err error) {
	// Consul 使用健康检查机制，无需额外心跳
	// 返回空函数，保持接口一致性
	return func() {}, nil
}

// ListInstances 列出所有实例
func (r *ConsulRegistry) ListInstances(ctx context.Context, serviceName string) ([]ServiceInstance, error) {
	// 查询健康服务实例
	entries, _, err := r.client.Health().Service(serviceName, "", true, nil)
	if err != nil {
		return nil, fmt.Errorf("查询服务实例失败: %w", err)
	}

	var instances []ServiceInstance

	for _, entry := range entries {
		service := entry.Service
		instance := ServiceInstance{
			ServiceName:   service.Service,
			InstanceID:    service.ID,
			Address:       service.Address,
			Port:          service.Port,
			Meta:          service.Meta,
			RegisteredAt:  0, // Consul 不提供注册时间
			LastHeartbeat: 0, // Consul 使用健康检查，无心跳时间
		}

		instances = append(instances, instance)
	}

	return instances, nil
}

// GetInstance 获取指定实例
func (r *ConsulRegistry) GetInstance(ctx context.Context, instanceID string) (ServiceInstance, error) {
	// 通过服务名称查询，然后过滤
	instances, err := r.ListInstances(ctx, r.serviceName)
	if err != nil {
		return ServiceInstance{}, err
	}

	for _, inst := range instances {
		if inst.InstanceID == instanceID {
			return inst, nil
		}
	}

	return ServiceInstance{}, fmt.Errorf("实例不存在: %s", instanceID)
}

// IsInstanceAlive 检查实例是否存活
func (r *ConsulRegistry) IsInstanceAlive(ctx context.Context, instanceID string) (bool, error) {
	// 查询服务健康状态
	entries, _, err := r.client.Health().Service(r.serviceName, "", true, nil)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if entry.Service.ID == instanceID {
			// 检查所有健康检查是否通过
			for _, check := range entry.Checks {
				if check.Status != consulapi.HealthPassing {
					return false, nil
				}
			}
			return true, nil
		}
	}

	return false, nil
}
