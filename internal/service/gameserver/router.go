package gameserver

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/kaifa/game-platform/internal/discovery"
	"github.com/kaifa/game-platform/internal/logger"
	"go.uber.org/zap"
)

// Router 游戏服务器路由
type Router struct {
	registry discovery.Registry
	strategy string // "least_conn", "round_robin", "random"
}

// NewRouter 创建游戏服务器路由
func NewRouter(registry discovery.Registry, strategy string) *Router {
	return &Router{
		registry: registry,
		strategy: strategy,
	}
}

// SelectInstance 选择游戏服务器实例
func (r *Router) SelectInstance(ctx context.Context) (discovery.ServiceInstance, error) {
	instances, err := r.registry.ListInstances(ctx, "game-server")
	if err != nil {
		return discovery.ServiceInstance{}, fmt.Errorf("获取服务实例列表失败: %w", err)
	}

	if len(instances) == 0 {
		return discovery.ServiceInstance{}, fmt.Errorf("没有可用的游戏服务器实例")
	}

	switch r.strategy {
	case "least_conn":
		return r.selectLeastConnections(ctx, instances)
	case "round_robin":
		return r.selectRoundRobin(ctx, instances)
	case "random":
		return r.selectRandom(instances), nil
	default:
		return instances[0], nil
	}
}

// GetRoomInstance 获取房间所在实例
func (r *Router) GetRoomInstance(ctx context.Context, roomID string) (discovery.ServiceInstance, error) {
	// 从 Consul KV 查询房间到实例的映射
	// 注意：这里需要 Consul KV 支持，如果使用 Redis，需要实现 RedisRegistry 的 GetRoomInstanceID 方法
	// 暂时先通过服务发现获取所有实例，然后选择第一个（后续会实现房间映射存储）

	instances, err := r.registry.ListInstances(ctx, "game-server")
	if err != nil {
		return discovery.ServiceInstance{}, err
	}

	if len(instances) == 0 {
		return discovery.ServiceInstance{}, fmt.Errorf("没有可用的游戏服务器实例")
	}

	// TODO: 从 Consul KV 或 Redis 查询房间映射
	// 暂时返回第一个实例
	return instances[0], nil
}

// selectLeastConnections 选择连接数最少的实例
func (r *Router) selectLeastConnections(ctx context.Context, instances []discovery.ServiceInstance) (discovery.ServiceInstance, error) {
	if len(instances) == 0 {
		return discovery.ServiceInstance{}, fmt.Errorf("没有可用实例")
	}

	minConnections := math.MaxInt
	selected := instances[0]

	for _, inst := range instances {
		connCount := r.getConnectionCount(ctx, inst.InstanceID)
		if connCount < minConnections {
			minConnections = connCount
			selected = inst
		}
	}

	return selected, nil
}

// getConnectionCount 获取实例的连接数
func (r *Router) getConnectionCount(ctx context.Context, instanceID string) int {
	// TODO: 从 Consul KV 或 Redis 获取连接数统计
	// 或通过 HTTP 调用实例的 /stats 接口
	// 暂时返回 0
	return 0
}

// selectRoundRobin 轮询选择
func (r *Router) selectRoundRobin(ctx context.Context, instances []discovery.ServiceInstance) (discovery.ServiceInstance, error) {
	if len(instances) == 0 {
		return discovery.ServiceInstance{}, fmt.Errorf("没有可用实例")
	}

	// TODO: 使用 Redis 原子计数器实现真正的轮询
	// 暂时返回第一个实例
	return instances[0], nil
}

// selectRandom 随机选择
func (r *Router) selectRandom(instances []discovery.ServiceInstance) discovery.ServiceInstance {
	if len(instances) == 0 {
		return discovery.ServiceInstance{}
	}

	// 使用时间戳作为随机种子
	index := int(time.Now().UnixNano()) % len(instances)
	return instances[index]
}

// AssignRoomToInstance 为房间分配实例
func (r *Router) AssignRoomToInstance(ctx context.Context, roomID string) (discovery.ServiceInstance, error) {
	// 选择实例
	instance, err := r.SelectInstance(ctx)
	if err != nil {
		return discovery.ServiceInstance{}, err
	}

	// TODO: 存储房间到实例的映射到 Consul KV 或 Redis
	// 例如：room:instance:<roomID> -> <instanceID>

	logger.Logger.Info("房间已分配实例",
		zap.String("room_id", roomID),
		zap.String("instance_id", instance.InstanceID),
		zap.String("address", instance.Address),
		zap.Int("port", instance.Port),
	)

	return instance, nil
}
