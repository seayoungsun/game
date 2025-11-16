# GameManager 重构说明

## 📋 概述

新的 GameManager 使用依赖注入模式，将游戏业务逻辑与数据访问层分离，提供更好的可测试性和可维护性。

---

## 🏗️ 架构对比

### 旧架构（pkg/services/game_manager.go）

```
GameManager
  ├─ 直接使用 database.DB
  ├─ 直接使用 cache.Get/Set
  └─ 业务逻辑 + 数据访问混杂
```

**问题：**
- 难以测试（需要真实数据库和 Redis）
- 难以替换存储实现
- 职责不清晰

### 新架构（internal/service/game/manager.go）

```
GameManager (纯业务逻辑)
  ├─ GameStateStorage (接口) - 游戏状态存储
  ├─ RoomRepository (接口) - 房间数据访问
  ├─ UserRepository (接口) - 用户数据访问
  ├─ GameRecordRepository (接口) - 游戏记录数据访问
  └─ LeaderboardService (接口) - 排行榜服务
```

**优势：**
- ✅ 易于测试（可以 Mock 所有依赖）
- ✅ 易于替换（实现不同的存储）
- ✅ 职责清晰（各层分工明确）

---

## 🚀 完整功能列表

### 核心方法

| 方法 | 说明 | 状态 |
|------|------|------|
| `StartGame` | 开始游戏（支持跑得快、牛牛） | ✅ 完成 |
| `PlayCards` | 出牌（跑得快游戏） | ✅ 完成 |
| `PlayBullGame` | 出牌（牛牛游戏） | ✅ 完成 |
| `Pass` | 过牌 | ✅ 完成 |
| `GetGameState` | 获取游戏状态 | ✅ 完成 |
| `GetGameStateForUser` | 获取过滤后的游戏状态 | ✅ 完成 |
| `CheckGameEnd` | 检查游戏是否结束 | ✅ 完成 |
| `SettleGame` | 结算游戏（跑得快） | ✅ 完成 |
| `settleBullGame` | 结算游戏（牛牛） | ✅ 完成 |

### 辅助方法

- `checkGameEnd` - 内部检查游戏结束
- `executeSettlement` - 通用结算流程
- `hasCards` - 检查是否拥有牌
- `removeCards` - 移除手牌
- `getNextPlayer` - 获取下一个玩家
- `getActivePlayerCount` - 获取活跃玩家数
- `calculateRank` - 计算名次

---

## 💻 使用示例

### 1. 在 main.go 中初始化

```go
package main

import (
    "github.com/kaifa/game-platform/internal/storage"
    gamesvc "github.com/kaifa/game-platform/internal/service/game"
    // ... 其他导入
)

func main() {
    // ... 初始化基础设施（DB、Redis）
    
    // 1. 创建 Repository 实例
    roomRepo := mysqlrepo.NewRoomRepository(infra.DB)
    userRepo := mysqlrepo.NewUserRepository(infra.DB)
    gameRecordRepo := mysqlrepo.NewGameRecordRepository(infra.DB)
    
    // 2. 创建 Storage 实例
    gameStateStorage := storage.NewRedisGameStateStorage(infra.Redis)
    
    // 3. 创建 Service 实例
    leaderboardService := leaderboardsrv.New(infra.Redis, userRepo)
    
    // 4. 创建重构版 GameManager（使用依赖注入）
    gameManager := gamesvc.NewManager(
        gameStateStorage,    // 游戏状态存储
        roomRepo,           // 房间Repository
        userRepo,           // 用户Repository
        gameRecordRepo,     // 游戏记录Repository
        leaderboardService, // 排行榜服务
    )
    
    // 5. 在 handlers 中使用
    handlers.SetGameManager(gameManager)
}
```

### 2. 在 Handler 中使用

```go
// apps/api/handlers/games.go

var gameManager *gamesvc.Manager

func SetGameManager(manager *gamesvc.Manager) {
    gameManager = manager
}

// 开始游戏
func StartGame(c *gin.Context) {
    userID, _ := c.Get("user_id")
    roomID := c.Param("roomId")
    
    // ✅ 使用新的 GameManager（传入 context）
    gameState, err := gameManager.StartGame(c.Request.Context(), roomID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
        return
    }
    
    // 过滤手牌后返回
    filteredState := gameState.FilterForUser(userID.(uint))
    c.JSON(http.StatusOK, gin.H{"code": 200, "data": filteredState})
}

// 出牌
func PlayCards(c *gin.Context) {
    userID, _ := c.Get("user_id")
    roomID := c.Param("roomId")
    
    var req struct {
        Cards []int `json:"cards" binding:"required"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
        return
    }
    
    // ✅ 使用新的 GameManager
    gameState, err := gameManager.PlayCards(
        c.Request.Context(), 
        roomID, 
        userID.(uint), 
        req.Cards,
    )
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
        return
    }
    
    filteredState := gameState.FilterForUser(userID.(uint))
    c.JSON(http.StatusOK, gin.H{"code": 200, "data": filteredState})
}
```

---

## 🧪 单元测试示例

```go
package game_test

import (
    "context"
    "testing"
    
    gamesvc "github.com/kaifa/game-platform/internal/service/game"
)

// Mock Storage
type MockGameStateStorage struct {
    states map[string]*models.GameState
}

func (m *MockGameStateStorage) Get(ctx context.Context, roomID string) (*models.GameState, error) {
    if state, ok := m.states[roomID]; ok {
        return state, nil
    }
    return nil, errors.New("游戏状态不存在")
}

func (m *MockGameStateStorage) Save(ctx context.Context, state *models.GameState, expiration time.Duration) error {
    m.states[state.RoomID] = state
    return nil
}

// Mock Repository
type MockRoomRepository struct {
    rooms map[string]*models.GameRoom
}

func (m *MockRoomRepository) GetByRoomID(ctx context.Context, roomID string) (*models.GameRoom, error) {
    if room, ok := m.rooms[roomID]; ok {
        return room, nil
    }
    return nil, errors.New("房间不存在")
}

// 测试开始游戏
func TestStartGame(t *testing.T) {
    // 创建 Mock 依赖
    mockStorage := &MockGameStateStorage{states: make(map[string]*models.GameState)}
    mockRoomRepo := &MockRoomRepository{rooms: make(map[string]*models.GameRoom)}
    mockUserRepo := &MockUserRepository{}
    mockGameRecordRepo := &MockGameRecordRepository{}
    mockLeaderboardSvc := &MockLeaderboardService{}
    
    // 创建 GameManager
    manager := gamesvc.NewManager(
        mockStorage,
        mockRoomRepo,
        mockUserRepo,
        mockGameRecordRepo,
        mockLeaderboardSvc,
    )
    
    // 准备测试数据
    mockRoomRepo.rooms["test-room"] = &models.GameRoom{
        RoomID:   "test-room",
        GameType: "running",
        Status:   1,
        // ... 其他字段
    }
    
    // 执行测试
    ctx := context.Background()
    gameState, err := manager.StartGame(ctx, "test-room")
    
    // 断言
    if err != nil {
        t.Fatalf("StartGame failed: %v", err)
    }
    if gameState == nil {
        t.Fatal("GameState should not be nil")
    }
    if gameState.RoomID != "test-room" {
        t.Errorf("Expected roomID 'test-room', got '%s'", gameState.RoomID)
    }
}
```

---

## 🔄 迁移步骤

### 方案 A：逐步迁移（推荐）

1. **新代码使用新 GameManager**
   - 在 main.go 中初始化新的 GameManager
   - 新的 Handler 或功能使用新版本

2. **旧代码继续使用旧 GameManager**
   - 旧的 `pkg/services/game_manager.go` 保持不变
   - 逐步迁移旧功能到新版本

3. **逐步替换**
   - 一个功能一个功能地迁移
   - 确保每个功能迁移后测试通过

### 方案 B：一次性切换

1. 更新 handlers/games.go 使用新 GameManager
2. 更新 main.go 初始化代码
3. 全面测试所有功能
4. 删除旧的 game_manager.go

---

## 📦 依赖关系图

```
apps/api/handlers/games.go
    ↓ 使用
internal/service/game/manager.go (GameManager)
    ↓ 依赖
    ├─ internal/storage/game_state.go (接口)
    │   └─ internal/storage/redis_game_state.go (实现)
    ├─ internal/repository/room/repository.go (接口)
    │   └─ internal/repository/mysql/room_repository.go (实现)
    ├─ internal/repository/user/repository.go (接口)
    │   └─ internal/repository/mysql/user_repository.go (实现)
    ├─ internal/repository/gamerecord/repository.go (接口)
    │   └─ internal/repository/mysql/gamerecord_repository.go (实现)
    └─ internal/service/leaderboard/service.go (服务)
```

---

## 💡 最佳实践

### 1. 始终传递 Context

```go
// ✅ 正确
gameState, err := manager.StartGame(ctx, roomID)

// ❌ 错误
gameState, err := manager.StartGame(context.Background(), roomID)  // 不要总是用 Background
```

### 2. 处理错误

```go
gameState, err := manager.PlayCards(ctx, roomID, userID, cards)
if err != nil {
    // 记录日志
    logger.Error("出牌失败", zap.Error(err))
    // 返回友好的错误消息
    return gin.H{"code": 400, "message": "出牌失败，请重试"}
}
```

### 3. 过滤敏感信息

```go
// 获取游戏状态时，始终过滤其他玩家的手牌
gameState, _ := manager.GetGameStateForUser(ctx, roomID, userID)
// 或者
gameState, _ := manager.GetGameState(ctx, roomID)
filteredState := gameState.FilterForUser(userID)
```

---

## 🎯 下一步计划

1. **完善测试覆盖**
   - 为所有核心方法编写单元测试
   - Mock 所有依赖

2. **性能优化**
   - 添加多级缓存（内存 + Redis）
   - 批量操作优化

3. **功能扩展**
   - 添加游戏回放功能
   - 添加游戏日志记录

4. **监控和告警**
   - 添加游戏状态监控
   - 添加异常告警

---

## ❓ 常见问题

### Q: 为什么要重构？

A: 旧的 GameManager 直接操作数据库和 Redis，难以测试和维护。新架构通过依赖注入分离关注点，使代码更清晰、更易测试。

### Q: 旧的 GameManager 还能用吗？

A: 可以。两个版本可以共存，逐步迁移。

### Q: 性能会受影响吗？

A: 不会。接口调用的开销可以忽略不计，反而通过更好的架构可以更容易地优化性能。

### Q: 如何切换到新版本？

A: 参考上面的"迁移步骤"，建议逐步迁移。

---

## 📚 相关文档

- [UserService 重构示例](../user/service.go)
- [Repository 模式说明](../../repository/README.md)
- [Storage 接口说明](../../storage/README.md)




