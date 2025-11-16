package storage

import (
	"context"
	"time"

	"github.com/kaifa/game-platform/pkg/models"
)

// GameStateStorage 定义游戏状态存储接口
// 将游戏状态的存储和业务逻辑分离，便于：
// 1. 测试：可以使用内存实现进行单元测试
// 2. 扩展：可以轻松切换存储方式（Redis/内存/数据库）
// 3. 优化：可以添加多级缓存（内存+Redis+DB）
type GameStateStorage interface {
	// Get 获取游戏状态
	Get(ctx context.Context, roomID string) (*models.GameState, error)

	// Save 保存游戏状态
	Save(ctx context.Context, state *models.GameState, expiration time.Duration) error

	// Delete 删除游戏状态
	Delete(ctx context.Context, roomID string) error

	// Exists 检查游戏状态是否存在
	Exists(ctx context.Context, roomID string) (bool, error)
}
