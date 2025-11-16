// 游戏相关工具函数
const GameUtils = {
    // 牌的点数映射（跑得快规则：3最小，A=14，2=15，小王=16，大王=17）
    rankMap: {
        3: '3', 4: '4', 5: '5', 6: '6', 7: '7', 8: '8', 9: '9', 10: '10',
        11: 'J', 12: 'Q', 13: 'K', 14: 'A', 15: '2', 16: '小王', 17: '大王'
    },
    
    // 花色映射（后端格式：0=红桃，1=方块，2=黑桃，3=梅花）
    suitMap: {
        0: '♥', // 红桃
        1: '♦', // 方块
        2: '♠', // 黑桃
        3: '♣'  // 梅花
    },
    
    // 花色名称
    suitNameMap: {
        0: 'heart',   // 红桃
        1: 'diamond', // 方块
        2: 'spade',   // 黑桃
        3: 'club'     // 梅花
    },

    // 获取牌面显示文本
    // 后端格式：花色*100 + 点数（例如：红桃3=0*100+3=3，方块6=1*100+6=106）
    getCardText(cardValue) {
        // 特殊牌：小王(16)，大王(17)
        if (cardValue === 16) return '小王';
        if (cardValue === 17) return '大王';
        
        // 解析花色和点数
        const suit = Math.floor(cardValue / 100);
        const rank = cardValue % 100;
        
        // 获取点数文本
        const rankText = this.rankMap[rank] || rank;
        
        // 如果是大小王，不需要花色
        if (rank === 16 || rank === 17) {
            return rankText;
        }
        
        // 获取花色符号
        const suitSymbol = this.suitMap[suit] || '';
        
        return rankText + suitSymbol;
    },

    // 获取牌的花色
    getCardSuit(cardValue) {
        // 特殊牌：小王(16)，大王(17)
        if (cardValue === 16 || cardValue === 17) return 'joker';
        
        const suit = Math.floor(cardValue / 100);
        return this.suitNameMap[suit] || 'unknown';
    },

    // 获取牌的点数
    getCardRank(cardValue) {
        // 特殊牌：小王(16)，大王(17)
        if (cardValue === 16 || cardValue === 17) return cardValue;
        
        return cardValue % 100;
    },

    // 牛牛游戏：计算牛牛牌型
    // 返回: {bullType, bullNum, maxCard, text}
    // bullType: 0=无牛, 1-9=有牛, 10=牛牛, 11=四花, 12=五花, 13=炸弹, 14=五小牛
    calculateBull(cards) {
        if (!cards || cards.length !== 5) {
            return {bullType: 0, bullNum: 0, maxCard: 0, text: '无牛'};
        }

        // 转换牌为点数（A=1, 2-10=2-10, J/Q/K=10）
        const points = cards.map(card => {
            const rank = this.getCardRank(card);
            if (rank >= 11) return 10; // J, Q, K
            if (rank === 1) return 1; // A
            return rank;
        });

        const values = cards.map(card => this.getCardRank(card));

        // 检查特殊牌型
        // 五小牛：5张牌都小于5且和小于等于10
        if (values.every(v => v < 5 && v !== 1)) {
            const sum = points.reduce((a, b) => a + b, 0);
            if (sum <= 10) {
                return {bullType: 14, bullNum: 0, maxCard: Math.max(...values), text: '五小牛'};
            }
        }

        // 五花：5张都是J/Q/K
        if (values.every(v => v >= 11)) {
            return {bullType: 12, bullNum: 0, maxCard: 13, text: '五花'};
        }

        // 四花：4张是J/Q/K
        const faceCount = values.filter(v => v >= 11).length;
        if (faceCount === 4) {
            return {bullType: 11, bullNum: 0, maxCard: Math.max(...values), text: '四花'};
        }

        // 炸弹：4张同点数
        const rankCount = {};
        values.forEach(v => {
            rankCount[v] = (rankCount[v] || 0) + 1;
            if (rankCount[v] >= 4) {
                return {bullType: 13, bullNum: 0, maxCard: v, text: '炸弹'};
            }
        });

        // 计算所有可能的3张牌组合
        const combinations = [
            [0, 1, 2], [0, 1, 3], [0, 1, 4],
            [0, 2, 3], [0, 2, 4], [0, 3, 4],
            [1, 2, 3], [1, 2, 4], [1, 3, 4],
            [2, 3, 4]
        ];

        // 找出3张牌的和是10的倍数的组合
        for (const combo of combinations) {
            const sum = points[combo[0]] + points[combo[1]] + points[combo[2]];
            if (sum % 10 === 0) {
                // 找到剩余2张牌
                const remaining = [0, 1, 2, 3, 4].filter(i => !combo.includes(i));
                const remainingSum = points[remaining[0]] + points[remaining[1]];
                const bullNum = remainingSum % 10;
                const maxCard = Math.max(...values);

                if (bullNum === 0) {
                    return {bullType: 10, bullNum: 0, maxCard, text: '牛牛'};
                } else {
                    return {bullType: bullNum, bullNum, maxCard, text: `${bullNum}牛`};
                }
            }
        }

        // 无牛
        const maxCard = Math.max(...values);
        return {bullType: 0, bullNum: 0, maxCard, text: '无牛'};
    },

    // 获取牛牛类型显示文本
    getBullTypeText(bullType, bullNum) {
        const typeMap = {
            0: '无牛',
            10: '牛牛',
            11: '四花',
            12: '五花',
            13: '炸弹',
            14: '五小牛'
        };
        if (typeMap[bullType]) {
            return typeMap[bullType];
        }
        if (bullType >= 1 && bullType <= 9) {
            return `${bullNum}牛`;
        }
        return '未知';
    },

    // 创建牌元素
    createCardElement(cardValue, clickable = true, isSelected = false) {
        const card = document.createElement('div');
        card.className = 'card';
        card.dataset.value = cardValue;
        
        const suit = this.getCardSuit(cardValue);
        const rank = this.getCardRank(cardValue);
        const text = this.getCardText(cardValue);

        card.textContent = text;
        
        // 添加花色类
        if (suit === 'heart' || suit === 'diamond') {
            card.style.color = '#e74c3c';
        } else if (suit === 'club' || suit === 'spade') {
            card.style.color = '#333';
        } else {
            card.style.color = '#f39c12';
        }

        if (isSelected) {
            card.classList.add('selected');
        }

        if (!clickable) {
            card.classList.add('disabled');
        }

        return card;
    },

    // 排序手牌（从小到大）
    sortCards(cards) {
        return [...cards].sort((a, b) => {
            // 特殊牌：小王(16)，大王(17)
            if (a === 16 && b === 17) return -1; // 小王在前
            if (a === 17 && b === 16) return 1;  // 大王在后
            
            // 如果一个是大小王，另一个不是
            if (a === 16 || a === 17) return 1;  // 大小王放后面
            if (b === 16 || b === 17) return -1;
            
            // 解析花色和点数
            const suitA = Math.floor(a / 100);
            const rankA = a % 100;
            const suitB = Math.floor(b / 100);
            const rankB = b % 100;
            
            // 首先按点数排序（从小到大）
            if (rankA !== rankB) {
                return rankA - rankB;
            }
            
            // 点数相同，按花色排序（红桃0 < 方块1 < 黑桃2 < 梅花3）
            return suitA - suitB;
        });
    },

    // 渲染手牌
    renderCards(container, cards, selectedCards = [], onCardClick = null) {
        container.innerHTML = '';
        
        // 先排序手牌
        const sortedCards = this.sortCards(cards || []);
        
        sortedCards.forEach(cardValue => {
            const isSelected = selectedCards.includes(cardValue);
            const cardElement = this.createCardElement(cardValue, !!onCardClick, isSelected);
            
            if (onCardClick) {
                cardElement.addEventListener('click', () => {
                    onCardClick(cardValue);
                });
            }
            
            container.appendChild(cardElement);
        });
    },

    // 渲染出的牌
    renderPlayedCards(container, cards) {
        container.innerHTML = '';
        if (!cards || cards.length === 0) {
            container.innerHTML = '<div style="color: #999;">暂无</div>';
            return;
        }
        
        cards.forEach(cardValue => {
            const cardElement = this.createCardElement(cardValue, false);
            container.appendChild(cardElement);
        });
    },

    // 格式化时间戳
    formatTime(timestamp) {
        if (!timestamp) return '';
        const date = new Date(timestamp * 1000);
        return date.toLocaleString('zh-CN');
    },

    // 格式化时长
    formatDuration(seconds) {
        if (seconds < 60) return `${seconds}秒`;
        const minutes = Math.floor(seconds / 60);
        const secs = seconds % 60;
        return `${minutes}分${secs}秒`;
    },

    // 显示Toast提示
    showToast(message, type = 'info') {
        const toast = document.getElementById('toast');
        toast.textContent = message;
        toast.className = `toast show ${type}`;
        
        setTimeout(() => {
            toast.classList.remove('show');
        }, 3000);
    }
};

window.GameUtils = GameUtils;

