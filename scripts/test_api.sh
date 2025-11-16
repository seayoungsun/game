#!/bin/bash

# API测试脚本

BASE_URL="http://localhost:8080/api/v1"

echo "=========================================="
echo "开始测试游戏平台API"
echo "=========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试函数
test_api() {
    local name=$1
    local method=$2
    local url=$3
    local headers=$4
    local data=$5
    
    echo -e "${YELLOW}测试: $name${NC}"
    
    if [ -n "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" \
            -H "Content-Type: application/json" \
            $headers \
            -d "$data")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" \
            $headers)
    fi
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" -eq 200 ] || [ "$http_code" -eq 201 ]; then
        echo -e "${GREEN}✓ 成功 (HTTP $http_code)${NC}"
        echo "$body" | python3 -m json.tool 2>/dev/null | head -20
    else
        echo -e "${RED}✗ 失败 (HTTP $http_code)${NC}"
        echo "$body"
    fi
    echo ""
}

# 1. 测试健康检查
echo "1️⃣  测试健康检查"
curl -s http://localhost:8080/health | python3 -m json.tool
echo ""

# 2. 注册用户
echo "2️⃣  测试用户注册"
REGISTER_RESPONSE=$(curl -s -X POST "$BASE_URL/users/register" \
    -H "Content-Type: application/json" \
    -d '{
        "phone": "13800138000",
        "password": "123456",
        "nickname": "测试用户1"
    }')
echo "$REGISTER_RESPONSE" | python3 -m json.tool | head -30

# 提取token
TOKEN=$(echo "$REGISTER_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data['data']['token']) if data.get('code') == 200 else print('')" 2>/dev/null)

if [ -z "$TOKEN" ]; then
    echo -e "${RED}注册失败，无法继续测试${NC}"
    exit 1
fi
echo -e "${GREEN}✓ 注册成功，获取Token${NC}"
echo ""

# 3. 登录
echo "3️⃣  测试用户登录"
test_api "登录" "POST" "$BASE_URL/users/login" "" '{"phone":"13800138000","password":"123456"}'

# 4. 获取用户信息
echo "4️⃣  测试获取用户信息"
test_api "用户信息" "GET" "$BASE_URL/users/profile" "-H \"Authorization: Bearer $TOKEN\""

# 5. 游戏列表
echo "5️⃣  测试游戏列表"
test_api "游戏列表" "GET" "$BASE_URL/games/list" ""

# 6. 创建房间
echo "6️⃣  测试创建房间"
CREATE_ROOM_RESPONSE=$(curl -s -X POST "$BASE_URL/games/rooms" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{
        "game_type": "texas",
        "room_type": "quick",
        "base_bet": 10,
        "max_players": 4
    }')
echo "$CREATE_ROOM_RESPONSE" | python3 -m json.tool | head -30

# 提取房间ID
ROOM_ID=$(echo "$CREATE_ROOM_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data['data']['room_id']) if data.get('code') == 200 else print('')" 2>/dev/null)

if [ -z "$ROOM_ID" ]; then
    echo -e "${RED}创建房间失败${NC}"
    exit 1
fi
echo -e "${GREEN}✓ 创建房间成功，房间ID: $ROOM_ID${NC}"
echo ""

# 7. 房间列表
echo "7️⃣  测试房间列表"
test_api "房间列表" "GET" "$BASE_URL/games/rooms?game_type=texas" ""

# 8. 房间详情
echo "8️⃣  测试房间详情"
test_api "房间详情" "GET" "$BASE_URL/games/rooms/$ROOM_ID" ""

echo ""
echo "=========================================="
echo -e "${GREEN}所有测试完成！${NC}"
echo "=========================================="










