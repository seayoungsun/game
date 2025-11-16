# 🔄 负载均衡改造方案

本文档详细说明如何将游戏平台改造为支持负载均衡的架构。

---

## 📋 当前架构分析

### 现状

1. **单实例架构**
   - 游戏服务器（game-server）只有一个实例
   - Hub 管理所有 WebSocket 连接（内存中）
   - 房间状态存储在内存中
   - API 服务直接调用游戏服务器的 `/internal/room/notify` 接口

2. **存在的问题**
   - 无法水平扩展
   - 单点故障
   - 连接数受限于单机性能

---

## 🎯 负载均衡改造目标

1. **支持多实例部署**
   - 多个游戏服务器实例可以同时运行
   - 每个实例独立管理自己的连接

2. **跨实例通信**
   - 房间内的玩家可能连接到不同实例
   - 需要跨实例广播消息

3. **服务发现**
   - API 服务需要知道房间在哪个实例
   - 动态路由到正确的实例

---

## 🔧 需要改造的模块

### 1. 游戏服务器（game-server）

#### 1.1 添加服务注册与发现

**新增文件：`apps/game-server/registry.go`**

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/redis/go-redis/v9"
    "go.uber.org/zap"
)

// ServiceRegistry 服务注册表
type ServiceRegistry struct {
    redis    *redis.Client
    instanceID string
    address    string
    port       int
    stopChan   chan struct{}
}

// NewServiceRegistry 创建服务注册表
func NewServiceRegistry(redis *redis.Client, instanceID, address string, port int) *ServiceRegistry {
    return &ServiceRegistry{
        redis:      redis,
        instanceID: instanceID,
        address:    address,
        port:       port,
        stopChan:   make(chan struct{}),
    }
}

// Register 注册服务实例
func (sr *ServiceRegistry) Register(ctx context.Context) error {
    key := fmt.Sprintf("game-server:instances:%s", sr.instanceID)
    value := map[string]interface{}{
        "instance_id": sr.instanceID,
        "address":     sr.address,
        "port":        sr.port,
        "registered_at": time.Now().Unix(),
    }
    
    data, _ := json.Marshal(value)
    
    // 设置过期时间为 30 秒，需要定期续期
    err := sr.redis.Set(ctx, key, data, 30*time.Second).Err()
    if err != nil {
        return err
    }
    
    // 添加到实例列表
    sr.redis.SAdd(ctx, "game-server:instances", sr.instanceID)
    
    logger.Logger.Info("服务实例已注册",
        zap.String("instance_id", sr.instanceID),
        zap.String("address", sr.address),
        zap.Int("port", sr.port),
    )
    
    return nil
}

// KeepAlive 保持心跳（定期续期）
func (sr *ServiceRegistry) KeepAlive(ctx context.Context) {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            sr.Register(ctx)
        case <-sr.stopChan:
            return
        }
    }
}

// Unregister 注销服务实例
func (sr *ServiceRegistry) Unregister(ctx context.Context) error {
    close(sr.stopChan)
    
    key := fmt.Sprintf("game-server:instances:%s", sr.instanceID)
    sr.redis.Del(ctx, key)
    sr.redis.SRem(ctx, "game-server:instances", sr.instanceID)
    
    logger.Logger.Info("服务实例已注销",
        zap.String("instance_id", sr.instanceID),
    )
    
    return nil
}

// GetInstance 获取实例信息
func (sr *ServiceRegistry) GetInstance(ctx context.Context, instanceID string) (map[string]interface{}, error) {
    key := fmt.Sprintf("game-server:instances:%s", instanceID)
    data, err := sr.redis.Get(ctx, key).Bytes()
    if err != nil {
        return nil, err
    }
    
    var instance map[string]interface{}
    err = json.Unmarshal(data, &instance)
    return instance, err
}

// GetAllInstances 获取所有实例
func (sr *ServiceRegistry) GetAllInstances(ctx context.Context) ([]string, error) {
    return sr.redis.SMembers(ctx, "game-server:instances").Result()
}
```

#### 1.2 添加房间到实例的映射

**修改：`apps/game-server/hub.go`**

```go
// 在 Hub 结构体中添加
type Hub struct {
    // ... 现有字段 ...
    
    // 房间到实例的映射（Redis）
    redis *redis.Client
    
    // 当前实例ID
    instanceID string
}

// 添加方法：注册房间到当前实例
func (h *Hub) RegisterRoom(ctx context.Context, roomID string) error {
    key := fmt.Sprintf("room:instance:%s", roomID)
    return h.redis.Set(ctx, key, h.instanceID, 0).Err()
}

// 添加方法：获取房间所在实例
func (h *Hub) GetRoomInstance(ctx context.Context, roomID string) (string, error) {
    key := fmt.Sprintf("room:instance:%s", roomID)
    return h.redis.Get(ctx, key).Result()
}

// 添加方法：删除房间映射
func (h *Hub) UnregisterRoom(ctx context.Context, roomID string) error {
    key := fmt.Sprintf("room:instance:%s", roomID)
    return h.redis.Del(ctx, key).Err()
}
```

#### 1.3 添加 Redis Pub/Sub 跨实例通信

**新增文件：`apps/game-server/pubsub.go`**

```go
package main

import (
    "context"
    "encoding/json"
    
    "github.com/redis/go-redis/v9"
    "go.uber.org/zap"
)

// PubSubManager 管理跨实例消息发布订阅
type PubSubManager struct {
    redis    *redis.Client
    pubsub   *redis.PubSub
    hub      *Hub
    instanceID string
}

// NewPubSubManager 创建发布订阅管理器
func NewPubSubManager(redis *redis.Client, hub *Hub, instanceID string) *PubSubManager {
    return &PubSubManager{
        redis:      redis,
        hub:        hub,
        instanceID: instanceID,
    }
}

// Start 启动订阅
func (psm *PubSubManager) Start(ctx context.Context) error {
    // 订阅所有实例的广播频道
    psm.pubsub = psm.redis.Subscribe(ctx, "game-server:broadcast")
    
    go func() {
        for {
            msg, err := psm.pubsub.ReceiveMessage(ctx)
            if err != nil {
                logger.Logger.Error("接收 PubSub 消息失败", zap.Error(err))
                continue
            }
            
            var broadcastMsg BroadcastMessage
            if err := json.Unmarshal([]byte(msg.Payload), &broadcastMsg); err != nil {
                logger.Logger.Error("解析 PubSub 消息失败", zap.Error(err))
                continue
            }
            
            // 如果是自己发送的消息，忽略
            if broadcastMsg.InstanceID == psm.instanceID {
                continue
            }
            
            // 处理跨实例消息
            psm.handleBroadcastMessage(&broadcastMsg)
        }
    }()
    
    logger.Logger.Info("PubSub 订阅已启动")
    return nil
}

// BroadcastMessage 跨实例广播消息
type BroadcastMessage struct {
    InstanceID string      `json:"instance_id"`
    RoomID     string      `json:"room_id"`
    UserID     uint        `json:"user_id"`
    Type       string      `json:"type"`
    Data       interface{} `json:"data"`
}

// Publish 发布消息到其他实例
func (psm *PubSubManager) Publish(ctx context.Context, msg *BroadcastMessage) error {
    msg.InstanceID = psm.instanceID
    data, err := json.Marshal(msg)
    if err != nil {
        return err
    }
    
    return psm.redis.Publish(ctx, "game-server:broadcast", data).Err()
}

// handleBroadcastMessage 处理跨实例消息
func (psm *PubSubManager) handleBroadcastMessage(msg *BroadcastMessage) {
    // 如果消息是针对特定房间的，检查当前实例是否有该房间的客户端
    if msg.RoomID != "" {
        psm.hub.mu.RLock()
        roomClients, exists := psm.hub.rooms[msg.RoomID]
        psm.hub.mu.RUnlock()
        
        if exists && len(roomClients) > 0 {
            // 当前实例有该房间的客户端，广播消息
            psm.hub.broadcast <- &Message{
                Type:   msg.Type,
                RoomID: msg.RoomID,
                UserID: msg.UserID,
                RawData: msg.Data,
            }
        }
    } else if msg.UserID != 0 {
        // 如果消息是针对特定用户的，检查当前实例是否有该用户的连接
        psm.hub.mu.RLock()
        client, exists := psm.hub.userClients[msg.UserID]
        psm.hub.mu.RUnlock()
        
        if exists {
            client.SendMessage(&Message{
                Type:   msg.Type,
                UserID: msg.UserID,
                RawData: msg.Data,
            })
        }
    } else {
        // 大厅广播，发送给所有客户端
        psm.hub.broadcast <- &Message{
            Type:   msg.Type,
            RawData: msg.Data,
        }
    }
}

// Stop 停止订阅
func (psm *PubSubManager) Stop() error {
    if psm.pubsub != nil {
        return psm.pubsub.Close()
    }
    return nil
}
```

#### 1.4 修改 main.go 集成新功能

**修改：`apps/game-server/main.go`**

```go
func main() {
    // ... 现有初始化代码 ...
    
    // 生成实例ID（可以使用机器ID + 时间戳）
    instanceID := fmt.Sprintf("%s-%d", cfg.Server.MachineID, time.Now().Unix())
    
    // 初始化服务注册表
    if infra.Redis != nil {
        registry := NewServiceRegistry(
            infra.Redis,
            instanceID,
            "0.0.0.0", // 或从配置读取
            cfg.Server.GamePort,
        )
        
        // 注册服务
        ctx := context.Background()
        if err := registry.Register(ctx); err != nil {
            logger.Logger.Fatal("服务注册失败", zap.Error(err))
        }
        
        // 启动心跳
        go registry.KeepAlive(ctx)
        
        // 优雅关闭时注销
        defer registry.Unregister(ctx)
        
        // 初始化 Hub（传入 Redis 和 instanceID）
        hub = NewHubWithRedis(infra.Redis, instanceID)
        
        // 初始化 PubSub
        pubsubManager := NewPubSubManager(infra.Redis, hub, instanceID)
        if err := pubsubManager.Start(ctx); err != nil {
            logger.Logger.Fatal("PubSub 启动失败", zap.Error(err))
        }
        defer pubsubManager.Stop()
    } else {
        // 降级方案：单实例模式
        hub = NewHub()
    }
    
    go hub.Run()
    
    // ... 其余代码 ...
}
```

---

### 2. API 服务（apps/api）

#### 2.1 添加游戏服务器路由服务

**新增文件：`internal/service/gameserver/router.go`**

```go
package gameserver

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/redis/go-redis/v9"
    "go.uber.org/zap"
)

// Router 游戏服务器路由服务
type Router struct {
    redis *redis.Client
}

// NewRouter 创建路由服务
func NewRouter(redis *redis.Client) *Router {
    return &Router{
        redis: redis,
    }
}

// GetRoomInstance 获取房间所在的实例
func (r *Router) GetRoomInstance(ctx context.Context, roomID string) (string, error) {
    key := fmt.Sprintf("room:instance:%s", roomID)
    instanceID, err := r.redis.Get(ctx, key).Result()
    if err == redis.Nil {
        return "", fmt.Errorf("房间 %s 未找到实例", roomID)
    }
    if err != nil {
        return "", err
    }
    return instanceID, nil
}

// GetInstanceAddress 获取实例地址
func (r *Router) GetInstanceAddress(ctx context.Context, instanceID string) (string, int, error) {
    key := fmt.Sprintf("game-server:instances:%s", instanceID)
    data, err := r.redis.Get(ctx, key).Bytes()
    if err != nil {
        return "", 0, err
    }
    
    var instance map[string]interface{}
    if err := json.Unmarshal(data, &instance); err != nil {
        return "", 0, err
    }
    
    address, _ := instance["address"].(string)
    port, _ := instance["port"].(float64)
    
    return address, int(port), nil
}

// NotifyRoom 通知房间（自动路由到正确的实例）
func (r *Router) NotifyRoom(ctx context.Context, roomID string, data interface{}) error {
    // 获取房间所在实例
    instanceID, err := r.GetRoomInstance(ctx, roomID)
    if err != nil {
        return fmt.Errorf("获取房间实例失败: %w", err)
    }
    
    // 获取实例地址
    address, port, err := r.GetInstanceAddress(ctx, instanceID)
    if err != nil {
        return fmt.Errorf("获取实例地址失败: %w", err)
    }
    
    // 发送 HTTP 请求到对应实例
    url := fmt.Sprintf("http://%s:%d/internal/room/notify", address, port)
    // ... HTTP 请求逻辑 ...
    
    return nil
}

// BroadcastToAllInstances 广播到所有实例（用于大厅消息）
func (r *Router) BroadcastToAllInstances(ctx context.Context, data interface{}) error {
    // 获取所有实例
    instanceIDs, err := r.redis.SMembers(ctx, "game-server:instances").Result()
    if err != nil {
        return err
    }
    
    // 向每个实例发送消息
    for _, instanceID := range instanceIDs {
        address, port, err := r.GetInstanceAddress(ctx, instanceID)
        if err != nil {
            logger.Logger.Warn("获取实例地址失败",
                zap.String("instance_id", instanceID),
                zap.Error(err),
            )
            continue
        }
        
        url := fmt.Sprintf("http://%s:%d/internal/room/notify", address, port)
        // ... 发送 HTTP 请求 ...
    }
    
    return nil
}
```

#### 2.2 修改 RoomService 使用路由服务

**修改：`internal/service/room/service.go`**

```go
// 在 RoomService 中添加 Router
type service struct {
    // ... 现有字段 ...
    router *gameserver.Router
}

// 修改通知方法
func (s *service) notifyGameServer(ctx context.Context, roomID string, action string, userID uint, roomData map[string]interface{}) {
    // 使用 Router 路由到正确的实例
    if s.router != nil {
        err := s.router.NotifyRoom(ctx, roomID, map[string]interface{}{
            "action":    action,
            "user_id":   userID,
            "room_data": roomData,
        })
        if err != nil {
            logger.Logger.Error("通知游戏服务器失败", zap.Error(err))
        }
    } else {
        // 降级方案：直接调用本地实例
        // ... 原有逻辑 ...
    }
}
```

---

### 3. 配置修改

#### 3.1 添加负载均衡配置

**修改：`configs/config.yaml`**

```yaml
server:
  mode: release
  port: 8080
  game_port: 8081
  admin_port: 8082
  machine_id: 0  # 每个实例使用不同的 machine_id
  instance_id: ""  # 可选，不设置则自动生成

# 负载均衡配置
load_balancer:
  enabled: true
  # 服务发现类型：redis, consul, etcd
  discovery_type: "redis"
  # 心跳间隔（秒）
  heartbeat_interval: 10
  # 实例过期时间（秒）
  instance_ttl: 30
```

---

### 4. Nginx 配置（负载均衡器）

**新增：`nginx/game-server-lb.conf`**

```nginx
upstream game_servers {
    # 使用 Redis 动态发现实例（需要 lua 脚本）
    # 或者使用静态配置
    server 10.0.0.1:8081;
    server 10.0.0.2:8081;
    server 10.0.0.3:8081;
    
    # 负载均衡策略
    # ip_hash;  # 基于 IP 的会话保持（推荐用于 WebSocket）
    least_conn;  # 最少连接数
}

server {
    listen 80;
    server_name ws.example.com;
    
    location /ws {
        proxy_pass http://game_servers;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_read_timeout 86400;
    }
    
    location /internal/ {
        # 内部接口，只允许内网访问
        allow 10.0.0.0/8;
        deny all;
        
        proxy_pass http://game_servers;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

---

## 📊 架构对比

### 改造前（单实例）

```
客户端 → Nginx → 游戏服务器（单实例）
                ↓
            Hub（内存）
            - 所有连接
            - 所有房间
```

### 改造后（多实例）

```
客户端 → Nginx（负载均衡）→ 游戏服务器实例1
                          → 游戏服务器实例2
                          → 游戏服务器实例3
                          
每个实例：
- Hub（本地连接）
- Redis（服务注册 + 房间映射 + PubSub）
- 跨实例通信
```

---

## 🔄 数据流

### 场景1：玩家加入房间

1. 客户端连接到实例1（通过负载均衡器）
2. 实例1注册房间映射：`room:instance:room123 → instance1`
3. API 服务查询房间实例，发送通知到实例1
4. 实例1广播给房间内所有客户端

### 场景2：跨实例房间

1. 玩家A连接到实例1，加入房间
2. 玩家B连接到实例2，加入同一房间
3. 房间映射：`room:instance:room123 → instance1`（第一个加入的实例）
4. 玩家B的操作通过 API → 路由到实例1 → PubSub → 实例2
5. 实例2收到消息，广播给玩家B

### 场景3：大厅广播

1. API 服务调用 `BroadcastToAllInstances`
2. 向所有实例发送 HTTP 请求
3. 每个实例广播给本地客户端

---

## ✅ 改造清单

### 必须改造

- [ ] 添加服务注册与发现（Redis）
- [ ] 添加房间到实例的映射（Redis）
- [ ] 添加跨实例通信（Redis Pub/Sub）
- [ ] 修改 Hub 支持 Redis
- [ ] 修改 API 服务使用路由服务
- [ ] 配置 Nginx 负载均衡

### 可选优化

- [ ] 使用 Consul/Etcd 替代 Redis 做服务发现
- [ ] 添加健康检查端点
- [ ] 实现优雅下线（迁移连接）
- [ ] 添加监控和告警

---

## 🚀 部署步骤

### 1. 准备多实例环境

```bash
# 实例1
APP_ENV=prod SERVER_MACHINE_ID=0 ./bin/game-server

# 实例2
APP_ENV=prod SERVER_MACHINE_ID=1 ./bin/game-server

# 实例3
APP_ENV=prod SERVER_MACHINE_ID=2 ./bin/game-server
```

### 2. 配置 Nginx

```bash
# 复制配置文件
cp nginx/game-server-lb.conf /etc/nginx/conf.d/

# 重载配置
sudo nginx -t
sudo nginx -s reload
```

### 3. 验证

```bash
# 检查服务注册
redis-cli SMEMBERS game-server:instances

# 检查房间映射
redis-cli GET room:instance:room123

# 测试连接
curl http://ws.example.com/stats
```

---

## ⚠️ 注意事项

1. **会话保持**：WebSocket 连接需要会话保持，建议使用 `ip_hash` 或基于用户ID的路由
2. **跨实例延迟**：跨实例通信会有延迟，需要优化
3. **数据一致性**：房间状态需要同步，建议使用 Redis 存储
4. **故障转移**：实例故障时需要迁移连接（复杂，可选）

---

## 📚 相关文档

- [Redis Pub/Sub 文档](https://redis.io/docs/manual/pubsub/)
- [Nginx 负载均衡](https://nginx.org/en/docs/http/load_balancing.html)
- [服务发现模式](https://microservices.io/patterns/service-registry.html)

