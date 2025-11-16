# 数据库设计文档

## 数据库选择

- **主数据库**: MySQL 8.0+
- **缓存**: Redis 7.0+
- **日志存储**: Elasticsearch 8.0+

## 表结构设计

### 用户相关表

#### users - 用户表
存储用户基本信息。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT | 主键 |
| uid | BIGINT | 用户ID（唯一） |
| phone | VARCHAR(20) | 手机号（唯一） |
| nickname | VARCHAR(50) | 昵称 |
| avatar | VARCHAR(255) | 头像URL |
| balance | DECIMAL(10,2) | 余额 |
| status | TINYINT | 状态：1正常，2封禁 |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |
| deleted_at | TIMESTAMP | 软删除时间 |

**索引：**
- PRIMARY KEY (id)
- UNIQUE KEY uk_uid (uid)
- UNIQUE KEY uk_phone (phone)
- KEY idx_created (created_at)
- KEY idx_status (status)

#### user_wallets - 用户钱包表
存储用户钱包详细信息。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT | 主键 |
| user_id | BIGINT | 用户ID（唯一） |
| balance | DECIMAL(10,2) | 余额 |
| frozen | DECIMAL(10,2) | 冻结金额 |
| total_in | DECIMAL(10,2) | 累计充值 |
| total_out | DECIMAL(10,2) | 累计提现 |
| updated_at | TIMESTAMP | 更新时间 |

#### user_logins - 用户登录记录表
记录用户登录历史。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT | 主键 |
| user_id | BIGINT | 用户ID |
| ip | VARCHAR(50) | IP地址 |
| device | VARCHAR(100) | 设备信息 |
| created_at | TIMESTAMP | 登录时间 |

### 游戏相关表

#### game_rooms - 游戏房间表
存储游戏房间信息。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT | 主键 |
| room_id | VARCHAR(50) | 房间ID（唯一） |
| game_type | VARCHAR(20) | 游戏类型：texas/bull/running |
| room_type | VARCHAR(20) | 房间类型：quick/middle/high |
| base_bet | DECIMAL(10,2) | 底注 |
| max_players | INT | 最大人数 |
| current_players | INT | 当前人数 |
| status | TINYINT | 状态：1等待，2游戏中，3已结束 |
| players | JSON | 玩家列表 |
| creator_id | BIGINT | 创建者ID |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

**索引：**
- PRIMARY KEY (id)
- UNIQUE KEY uk_room_id (room_id)
- KEY idx_game_type (game_type)
- KEY idx_status (status)

#### game_records - 游戏对局记录表
存储游戏对局的摘要信息（详细日志存储在ES）。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT | 主键 |
| room_id | VARCHAR(50) | 房间ID |
| game_type | VARCHAR(20) | 游戏类型 |
| players | JSON | 玩家列表 |
| result | JSON | 结算结果 |
| start_time | TIMESTAMP | 开始时间 |
| end_time | TIMESTAMP | 结束时间 |
| duration | INT | 时长（秒） |
| created_at | TIMESTAMP | 创建时间 |

**索引：**
- PRIMARY KEY (id)
- KEY idx_room_id (room_id)
- KEY idx_game_type (game_type, start_time)

#### game_players - 游戏玩家关联表
记录玩家参与游戏的信息。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT | 主键 |
| room_id | VARCHAR(50) | 房间ID |
| user_id | BIGINT | 用户ID |
| position | INT | 位置 |
| balance | DECIMAL(10,2) | 本局余额变化 |
| created_at | TIMESTAMP | 创建时间 |

### 支付相关表

#### transactions - 交易订单表
存储所有交易记录。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT | 主键 |
| order_id | VARCHAR(64) | 订单号（唯一） |
| user_id | BIGINT | 用户ID |
| type | VARCHAR(20) | 类型：recharge/withdraw/game |
| amount | DECIMAL(10,2) | 金额 |
| status | TINYINT | 状态：1待处理，2成功，3失败 |
| channel | VARCHAR(20) | 支付渠道：alipay/wechat |
| channel_id | VARCHAR(100) | 第三方订单号 |
| remark | VARCHAR(255) | 备注 |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

#### recharge_orders - 充值订单表
存储充值订单。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT | 主键 |
| order_id | VARCHAR(64) | 订单号（唯一） |
| user_id | BIGINT | 用户ID |
| amount | DECIMAL(10,2) | 充值金额 |
| status | TINYINT | 状态：1待支付，2已支付，3已取消 |
| channel | VARCHAR(20) | 支付渠道 |
| channel_id | VARCHAR(100) | 第三方订单号 |
| paid_at | TIMESTAMP | 支付时间 |
| expire_at | TIMESTAMP | 过期时间 |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

#### withdraw_orders - 提现订单表
存储提现订单。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT | 主键 |
| order_id | VARCHAR(64) | 订单号（唯一） |
| user_id | BIGINT | 用户ID |
| amount | DECIMAL(10,2) | 提现金额 |
| status | TINYINT | 状态：1待审核，2已通过，3已拒绝 |
| bank_card | VARCHAR(50) | 银行卡号 |
| bank_name | VARCHAR(50) | 银行名称 |
| real_name | VARCHAR(50) | 真实姓名 |
| remark | VARCHAR(255) | 备注 |
| audit_at | TIMESTAMP | 审核时间 |
| auditor_id | BIGINT | 审核员ID |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |
| deleted_at | TIMESTAMP | 软删除时间 |

## Redis数据结构设计

### 用户会话
```
key: user:session:{user_id}
value: JSON字符串
expire: 24小时
```

### 房间状态
```
key: room:{room_id}
value: Hash
  - status: 状态
  - players: 玩家列表
  - game_state: 游戏状态
expire: 1小时
```

### 游戏对局状态
```
key: game:hand:{room_id}
value: Hash
  - current_player: 当前玩家
  - cards: 牌面
  - history: 历史操作
expire: 30分钟
```

### 排行榜
```
key: leaderboard:{game_type}:{type}
type: ZSET
  - member: user_id
  - score: 积分/金币
```

### 分布式锁
```
key: lock:{resource}
value: 锁持有者标识
expire: 自动过期时间
```

## Elasticsearch索引设计

### 游戏日志索引
```
index: game-logs-YYYY.MM.DD
mapping:
  - @timestamp: date
  - user_id: long
  - room_id: keyword
  - game_type: keyword
  - action: keyword
  - message: text (中文分词)
  - data: object
```

### 用户行为日志
```
index: user-behavior-logs-YYYY.MM.DD
mapping:
  - @timestamp: date
  - user_id: long
  - action: keyword
  - page: keyword
  - data: object
```

### 错误日志
```
index: error-logs-YYYY.MM.DD
mapping:
  - @timestamp: date
  - level: keyword
  - message: text
  - stack: text
```

## 数据备份策略

### MySQL备份
- **全量备份**: 每日凌晨2点
- **增量备份**: 每小时
- **保留时间**: 30天

### Redis备份
- **RDB快照**: 每小时
- **AOF日志**: 实时

### Elasticsearch备份
- **索引快照**: 每日
- **保留时间**: 90天（根据日志级别调整）

## 性能优化

### MySQL优化
1. **索引优化**: 为常用查询字段添加索引
2. **分区表**: 游戏记录表按时间分区
3. **读写分离**: 主从复制，读操作分流
4. **连接池**: 合理设置连接池大小

### Redis优化
1. **数据过期**: 设置合理的TTL
2. **内存策略**: allkeys-lru
3. **持久化**: RDB + AOF混合
4. **集群模式**: 数据量大时使用Redis Cluster

### Elasticsearch优化
1. **索引模板**: 使用模板统一配置
2. **生命周期管理**: 自动删除旧数据
3. **分片策略**: 合理设置分片数
4. **查询优化**: 使用filter代替query

## 数据迁移

使用 `scripts/migrate/main.go` 执行数据库迁移。

```bash
# 执行迁移
make migrate
# 或
cd scripts/migrate && go run main.go
```

## 注意事项

1. **金额字段**: 使用DECIMAL类型，避免浮点数精度问题
2. **时间字段**: 统一使用TIMESTAMP类型
3. **JSON字段**: MySQL 5.7+支持JSON类型
4. **软删除**: 重要数据使用deleted_at实现软删除
5. **索引**: 不要过度索引，影响写入性能


