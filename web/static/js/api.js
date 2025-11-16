// API 基础配置
const API_BASE_URL = 'http://localhost:8080/api/v1';
const WS_URL = 'ws://localhost:8081/ws';

// 存储 token 和用户信息
let authToken = localStorage.getItem('auth_token');
let currentUser = JSON.parse(localStorage.getItem('current_user') || 'null');

// API 请求封装
async function apiRequest(url, options = {}) {
    const headers = {
        'Content-Type': 'application/json',
        ...options.headers
    };

    if (authToken) {
        headers['Authorization'] = `Bearer ${authToken}`;
    }

    try {
        const response = await fetch(`${API_BASE_URL}${url}`, {
            ...options,
            headers
        });

        const data = await response.json();

        if (!response.ok) {
            throw new Error(data.message || '请求失败');
        }

        return data;
    } catch (error) {
        console.error('API请求失败:', error);
        throw error;
    }
}

// 用户相关API
const UserAPI = {
    // 注册
    async register(phone, password, nickname) {
        const result = await apiRequest('/users/register', {
            method: 'POST',
            body: JSON.stringify({ phone, password, nickname })
        });
        if (result.data && result.data.token) {
            authToken = result.data.token;
            currentUser = result.data.user;
            localStorage.setItem('auth_token', authToken);
            localStorage.setItem('current_user', JSON.stringify(currentUser));
            // 更新window上的函数，确保全局可用
            window.authToken = () => authToken;
            window.currentUser = () => currentUser;
        }
        return result;
    },

    // 登录
    async login(phone, password) {
        console.log('UserAPI.login 调用:', { phone });
        const result = await apiRequest('/users/login', {
            method: 'POST',
            body: JSON.stringify({ phone, password })
        });
        console.log('UserAPI.login 返回:', result);
        
        if (result && result.data) {
            if (result.data.token) {
                authToken = result.data.token;
                currentUser = result.data.user || result.data;
                localStorage.setItem('auth_token', authToken);
                localStorage.setItem('current_user', JSON.stringify(currentUser));
                
                // 更新window上的函数，确保全局可用
                window.authToken = () => authToken;
                window.currentUser = () => currentUser;
                
                console.log('Token和用户信息已保存:', { 
                    hasToken: !!authToken, 
                    hasUser: !!currentUser,
                    userId: currentUser?.id 
                });
            } else {
                console.warn('登录返回中没有token');
            }
        } else {
            console.warn('登录返回数据格式异常:', result);
        }
        return result;
    },

    // 获取用户信息
    async getProfile() {
        return await apiRequest('/users/profile');
    },

    // 获取用户统计
    async getStats() {
        return await apiRequest('/users/stats');
    },

    // 退出登录
    logout() {
        authToken = null;
        currentUser = null;
        localStorage.removeItem('auth_token');
        localStorage.removeItem('current_user');
    }
};

// 游戏相关API
const GameAPI = {
    // 获取游戏列表
    async getGameList() {
        return await apiRequest('/games/list');
    },

    // 获取房间列表
    async getRoomList(gameType = '') {
        const url = gameType ? `/games/rooms?game_type=${gameType}` : '/games/rooms';
        return await apiRequest(url);
    },

    // 获取房间详情
    async getRoom(roomId) {
        return await apiRequest(`/games/rooms/${roomId}`);
    },

    // 创建房间
    async createRoom(data) {
        return await apiRequest('/games/rooms', {
            method: 'POST',
            body: JSON.stringify(data)
        });
    },

    // 加入房间
    async joinRoom(roomId, password = '') {
        return await apiRequest(`/games/rooms/${roomId}/join`, {
            method: 'POST',
            body: JSON.stringify({ password })
        });
    },

    // 离开房间
    async leaveRoom(roomId) {
        return await apiRequest(`/games/rooms/${roomId}/leave`, {
            method: 'POST'
        });
    },

    // 准备
    async ready(roomId) {
        return await apiRequest(`/games/rooms/${roomId}/ready`, {
            method: 'POST'
        });
    },

    // 取消准备
    async cancelReady(roomId) {
        return await apiRequest(`/games/rooms/${roomId}/cancel-ready`, {
            method: 'POST'
        });
    },

    // 开始游戏
    async startGame(roomId) {
        return await apiRequest(`/games/rooms/${roomId}/start`, {
            method: 'POST'
        });
    },

    // 获取游戏状态
    async getGameState(roomId) {
        return await apiRequest(`/games/rooms/${roomId}/game-state`);
    },

    // 出牌
    async playCards(roomId, cards) {
        return await apiRequest(`/games/rooms/${roomId}/play`, {
            method: 'POST',
            body: JSON.stringify({ cards })
        });
    },

    // 过牌
    async pass(roomId) {
        return await apiRequest(`/games/rooms/${roomId}/pass`, {
            method: 'POST'
        });
    },

    // 获取排行榜
    async getLeaderboard(gameType = 'running', period = 'total', page = 1, pageSize = 20) {
        return await apiRequest(`/games/leaderboard?game_type=${gameType}&period=${period}&page=${page}&page_size=${pageSize}`);
    },

    // 获取我的排名
    async getMyRank(gameType = 'running', period = 'total') {
        return await apiRequest(`/games/leaderboard/my-rank?game_type=${gameType}&period=${period}`);
    },

    // 获取我的记录
    async getMyRecords(gameType = '', page = 1, pageSize = 20) {
        const url = gameType 
            ? `/games/records?game_type=${gameType}&page=${page}&page_size=${pageSize}`
            : `/games/records?page=${page}&page_size=${pageSize}`;
        return await apiRequest(url);
    },

    // 获取记录详情
    async getRecordDetail(recordId) {
        return await apiRequest(`/games/records/${recordId}`);
    },

    // 获取房间记录
    async getRoomRecords(roomId) {
        return await apiRequest(`/games/rooms/${roomId}/records`);
    }
};

// 支付相关API
const PaymentAPI = {
    // 创建充值订单
    async createRechargeOrder(amount, chainType) {
        return await apiRequest('/payments/recharge', {
            method: 'POST',
            body: JSON.stringify({ amount, chain_type: chainType })
        });
    },
    
    // 获取充值订单
    async getRechargeOrder(orderId) {
        return await apiRequest(`/payments/recharge/${orderId}`);
    },
    
    // 获取用户充值订单列表
    async getRechargeOrders(page = 1, pageSize = 20) {
        return await apiRequest(`/payments/recharge?page=${page}&page_size=${pageSize}`);
    },
    
    // 检查充值交易
    async checkRechargeTransaction(orderId) {
        return await apiRequest(`/payments/recharge/${orderId}/check`, {
            method: 'POST'
        });
    },
    
    // 创建提现订单
    async createWithdrawOrder(amount, chainType, toAddress) {
        return await apiRequest('/payments/withdraw', {
            method: 'POST',
            body: JSON.stringify({ amount, chain_type: chainType, to_address: toAddress })
        });
    },
    
    // 获取提现订单
    async getWithdrawOrder(orderId) {
        return await apiRequest(`/payments/withdraw/${orderId}`);
    },
    
    // 获取用户提现订单列表
    async getWithdrawOrders(page = 1, pageSize = 20) {
        return await apiRequest(`/payments/withdraw?page=${page}&page_size=${pageSize}`);
    },
    
    // 审核提现订单（管理员）
    async auditWithdrawOrder(orderId, approve, remark = '') {
        return await apiRequest(`/payments/withdraw/${orderId}/audit`, {
            method: 'POST',
            body: JSON.stringify({ approve, remark })
        });
    }
};

// 导出
window.UserAPI = UserAPI;
window.GameAPI = GameAPI;
window.PaymentAPI = PaymentAPI;
window.authToken = () => authToken;
window.currentUser = () => currentUser;

