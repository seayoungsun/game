// 主应用逻辑
let currentRoomId = null;
let currentGameState = null;
let selectedCards = [];

// 初始化
document.addEventListener('DOMContentLoaded', () => {
    console.log('DOM加载完成，开始初始化');
    
    // 确保表单可见（在初始化之前）
    setTimeout(() => {
        initAuth();
        initWebSocket();
        checkAuth();
        console.log('初始化完成');
    }, 50);
});

// 检查登录状态
function checkAuth() {
    // 从api.js获取函数
    const getAuthToken = window.authToken || (() => null);
    const getCurrentUser = window.currentUser || (() => null);
    
    if (getAuthToken() && getCurrentUser()) {
        showMainPage();
        loadUserInfo();
        wsManager.connect();
    } else {
        showAuthPage();
    }
}

// 初始化认证相关
function initAuth() {
    // 确保登录表单默认显示
    const loginForm = document.getElementById('login-form');
    const registerForm = document.getElementById('register-form');
    if (loginForm) {
        loginForm.style.display = 'block';
        loginForm.classList.add('active');
    }
    if (registerForm) {
        registerForm.style.display = 'none';
        registerForm.classList.remove('active');
    }
    
    // 登录/注册切换
    document.querySelectorAll('.tab-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const tab = btn.dataset.tab;
            document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
            document.querySelectorAll('.auth-form').forEach(f => {
                f.classList.remove('active');
                f.style.display = 'none';
            });
            btn.classList.add('active');
            const targetForm = document.getElementById(`${tab}-form`);
            if (targetForm) {
                targetForm.style.display = 'block';
                targetForm.classList.add('active');
            }
        });
    });

    // 登录表单
    document.getElementById('login-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        const formData = new FormData(e.target);
        const phone = formData.get('phone');
        const password = formData.get('password');
        
        console.log('开始登录:', phone);
        
        try {
            const result = await UserAPI.login(phone, password);
            console.log('登录API返回:', result);
            
            if (result && result.data && result.data.token) {
                console.log('登录成功，token已保存');
                
                // 等待token和用户信息更新到window对象
                await new Promise(resolve => setTimeout(resolve, 100));
                
                // 验证token和用户信息是否已设置
                const token = window.authToken ? window.authToken() : null;
                const user = window.currentUser ? window.currentUser() : null;
                console.log('验证登录状态 - token:', !!token, 'user:', !!user);
                
                if (token && user) {
                    GameUtils.showToast('登录成功', 'success');
                    console.log('准备跳转到主页面');
                    showMainPage();
                    loadUserInfo();
                    wsManager.connect();
                } else {
                    console.error('登录状态验证失败');
                    GameUtils.showToast('登录失败：状态未更新', 'error');
                }
            } else {
                console.error('登录返回数据格式错误:', result);
                GameUtils.showToast('登录失败：返回数据错误', 'error');
            }
        } catch (error) {
            console.error('登录错误:', error);
            showError('login-error', error.message);
            GameUtils.showToast(`登录失败: ${error.message}`, 'error');
        }
    });

    // 注册表单
    document.getElementById('register-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        const formData = new FormData(e.target);
        const phone = formData.get('phone');
        const password = formData.get('password');
        const nickname = formData.get('nickname');
        
        console.log('开始注册:', phone);
        
        try {
            const result = await UserAPI.register(phone, password, nickname);
            console.log('注册API返回:', result);
            
            if (result && result.data && result.data.token) {
                console.log('注册成功，token已保存');
                
                // 等待token和用户信息更新到window对象
                await new Promise(resolve => setTimeout(resolve, 100));
                
                // 验证token和用户信息是否已设置
                const token = window.authToken ? window.authToken() : null;
                const user = window.currentUser ? window.currentUser() : null;
                console.log('验证注册状态 - token:', !!token, 'user:', !!user);
                
                if (token && user) {
                    GameUtils.showToast('注册成功', 'success');
                    console.log('准备跳转到主页面');
                    showMainPage();
                    loadUserInfo();
                    wsManager.connect();
                } else {
                    console.error('注册状态验证失败');
                    GameUtils.showToast('注册失败：状态未更新', 'error');
                }
            } else {
                console.error('注册返回数据格式错误:', result);
                GameUtils.showToast('注册失败：返回数据错误', 'error');
            }
        } catch (error) {
            console.error('注册错误:', error);
            showError('register-error', error.message);
            GameUtils.showToast(`注册失败: ${error.message}`, 'error');
        }
    });

    // 退出登录
    document.getElementById('logout-btn').addEventListener('click', () => {
        UserAPI.logout();
        wsManager.disconnect();
        showAuthPage();
    });
}

// 显示错误信息
function showError(elementId, message) {
    const errorEl = document.getElementById(elementId);
    if (errorEl) {
        errorEl.textContent = message;
        errorEl.classList.add('show');
        setTimeout(() => errorEl.classList.remove('show'), 3000);
    }
}

// 页面切换
function showAuthPage() {
    const authPage = document.getElementById('auth-page');
    const mainPage = document.getElementById('main-page');
    const roomPage = document.getElementById('room-page');
    const gamePage = document.getElementById('game-page');
    
    // 隐藏所有页面
    [mainPage, roomPage, gamePage].forEach(page => {
        if (page) {
            page.classList.remove('active');
            page.style.display = 'none';
        }
    });
    
    // 显示登录页面
    if (authPage) {
        authPage.classList.add('active');
        authPage.style.display = 'block';
    }
}

function showMainPage() {
    console.log('showMainPage 被调用');
    try {
        const authPage = document.getElementById('auth-page');
        const mainPage = document.getElementById('main-page');
        const roomPage = document.getElementById('room-page');
        const gamePage = document.getElementById('game-page');
        
        if (!authPage || !mainPage) {
            console.error('页面元素不存在:', { authPage: !!authPage, mainPage: !!mainPage });
            return;
        }
        
        // 移除所有页面的active类
        [authPage, mainPage, roomPage, gamePage].forEach(page => {
            if (page) {
                page.classList.remove('active');
                // 强制设置display样式
                page.style.display = 'none';
            }
        });
        
        // 添加main-page的active类并显示
        mainPage.classList.add('active');
        mainPage.style.display = 'block';
        
        console.log('页面切换完成 - main-page active:', mainPage.classList.contains('active'));
        console.log('页面元素样式:', {
            authPageDisplay: authPage.style.display || 'inherit',
            mainPageDisplay: mainPage.style.display || 'inherit',
            authPageActive: authPage.classList.contains('active'),
            mainPageActive: mainPage.classList.contains('active')
        });
        
        // 默认显示大厅视图
        setTimeout(() => {
            const lobbyView = document.getElementById('lobby-view');
            // 充值页面
            const rechargeBtn = document.querySelector('[data-view="recharge"]');
            if (rechargeBtn) {
                rechargeBtn.addEventListener('click', () => {
                    showRechargePage();
                });
            }
            
            const lobbyBtn = document.querySelector('[data-view="lobby"]');
            if (lobbyView && lobbyBtn) {
                document.querySelectorAll('.nav-btn').forEach(b => b.classList.remove('active'));
                document.querySelectorAll('.view').forEach(v => v.classList.remove('active'));
                lobbyBtn.classList.add('active');
                lobbyView.classList.add('active');
                loadRooms();
            }
        }, 100);
    } catch (error) {
        console.error('showMainPage 错误:', error);
    }
}

function showRoomPage() {
    const authPage = document.getElementById('auth-page');
    const mainPage = document.getElementById('main-page');
    const roomPage = document.getElementById('room-page');
    const gamePage = document.getElementById('game-page');
    
    [authPage, mainPage, gamePage].forEach(page => {
        if (page) {
            page.classList.remove('active');
            page.style.display = 'none';
        }
    });
    
    if (roomPage) {
        roomPage.classList.add('active');
        roomPage.style.display = 'block';
    }
}

function showGamePage() {
    const authPage = document.getElementById('auth-page');
    const mainPage = document.getElementById('main-page');
    const roomPage = document.getElementById('room-page');
    const gamePage = document.getElementById('game-page');
    
    [authPage, mainPage, roomPage].forEach(page => {
        if (page) {
            page.classList.remove('active');
            page.style.display = 'none';
        }
    });
    
    if (gamePage) {
        gamePage.classList.add('active');
        gamePage.style.display = 'block';
    }
}

// 加载用户信息
async function loadUserInfo() {
    try {
        const profile = await UserAPI.getProfile();
        if (profile.data) {
            const user = profile.data;
            const nicknameEl = document.getElementById('user-nickname');
            const balanceEl = document.getElementById('user-balance');
            if (nicknameEl) nicknameEl.textContent = user.nickname || user.phone;
            if (balanceEl) balanceEl.textContent = `余额: ${user.balance || 0}`;
        }
    } catch (error) {
        console.error('加载用户信息失败:', error);
        // 如果API失败，尝试从localStorage读取
        const user = window.currentUser();
        if (user) {
            const nicknameEl = document.getElementById('user-nickname');
            const balanceEl = document.getElementById('user-balance');
            if (nicknameEl) nicknameEl.textContent = user.nickname || user.phone;
            if (balanceEl) balanceEl.textContent = `余额: ${user.balance || 0}`;
        }
    }
}

// 初始化导航
function initNavigation() {
    document.querySelectorAll('.nav-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const view = btn.dataset.view;
            document.querySelectorAll('.nav-btn').forEach(b => b.classList.remove('active'));
            document.querySelectorAll('.view').forEach(v => {
                v.classList.remove('active');
                v.style.display = 'none';
            });
            btn.classList.add('active');
            const viewEl = document.getElementById(`${view}-view`);
            if (viewEl) {
                viewEl.classList.add('active');
                viewEl.style.display = 'block';
            }
            
            if (view === 'lobby') {
                loadRooms();
            } else if (view === 'leaderboard') {
                loadLeaderboard();
            } else if (view === 'records') {
                loadRecords();
            } else if (view === 'recharge') {
                loadRechargeOrders();
            } else if (view === 'withdraw') {
                showWithdrawPage();
            }
        });
    });
}

// 初始化WebSocket
function initWebSocket() {
    // WebSocket消息处理
    wsManager.on('connected', () => {
        console.log('WebSocket连接成功');
        // 连接成功时不显示Toast，避免干扰
    });

    wsManager.on('room_updated', (message) => {
        console.log('收到房间更新消息:', message);
        if (currentRoomId) {
            loadRoomInfo(currentRoomId);
        }
    });

    wsManager.on('room_created', (message) => {
        console.log('收到房间创建消息:', message);
        console.log('消息数据:', message.raw_data);
        
        // 尝试多种方式获取房间数据
        let room = null;
        if (message.raw_data) {
            if (message.raw_data.room) {
                room = message.raw_data.room;
            } else if (message.raw_data.room_data) {
                room = message.raw_data.room_data;
            } else if (message.data && message.data.room) {
                room = message.data.room;
            }
        }
        
        if (room) {
            console.log('提取到的房间数据:', room);
            // 如果当前在大厅页面，自动添加新房间
            const currentView = window.location.hash.replace('#', '') || 'lobby';
            console.log('当前页面:', currentView);
            if (currentView === 'lobby' || currentView === '') {
                console.log('大厅页面检测到新房间，自动添加:', room);
                // 检查房间类型是否匹配当前筛选
                const gameTypeFilterEl = document.getElementById('game-type-filter');
                const gameTypeFilter = gameTypeFilterEl ? gameTypeFilterEl.value : '';
                console.log('当前筛选条件:', gameTypeFilter, '房间类型:', room.game_type);
                
                if (!gameTypeFilter || gameTypeFilter === '全部' || room.game_type === gameTypeFilter) {
                    // 添加新房间到列表顶部
                    console.log('添加房间到列表:', room);
                    addRoomToList(room);
                    GameUtils.showToast(`新房间已创建: ${room.room_id}`, 'info');
                } else {
                    console.log('房间类型不匹配，不添加');
                }
            } else {
                console.log('当前不在大厅页面，不添加房间');
            }
        } else {
            console.warn('未找到房间数据:', message);
        }
    });

    wsManager.on('room_deleted', (message) => {
        console.log('收到房间删除消息:', message);
        
        // 获取房间ID
        let roomId = null;
        if (message.raw_data && message.raw_data.room_id) {
            roomId = message.raw_data.room_id;
        } else if (message.room_id) {
            roomId = message.room_id;
        }
        
        if (roomId) {
            console.log('房间已解散，移除房间:', roomId);
            // 如果当前在大厅页面，自动移除房间
            const currentView = window.location.hash.replace('#', '') || 'lobby';
            if (currentView === 'lobby' || currentView === '') {
                removeRoomFromList(roomId);
                GameUtils.showToast(`房间 ${roomId} 已解散`, 'info');
            }
            
            // 如果用户正在这个房间，跳转到大厅
            if (currentRoomId === roomId) {
                currentRoomId = null;
                showLobbyPage();
                GameUtils.showToast('房间已解散，已返回大厅', 'warning');
            }
        } else {
            console.warn('未找到房间ID:', message);
        }
    });

    wsManager.on('game_state_update', (message) => {
        console.log('收到游戏状态更新:', message);
        if (message.raw_data && message.raw_data.game_state) {
            console.log('游戏状态数据:', message.raw_data.game_state);
            // 确保在游戏页面
            showGamePage();
            updateGameState(message.raw_data.game_state);
            
            // 显示提示（仅第一次）
            if (message.raw_data.message && message.raw_data.message.includes('游戏已开始')) {
                GameUtils.showToast('游戏已开始', 'success');
            }
        } else {
            console.warn('游戏状态更新消息格式异常:', message);
        }
    });

    wsManager.on('game_state_recovery', (message) => {
        console.log('收到游戏状态恢复:', message);
        if (message.raw_data && message.raw_data.game_state) {
            // 确保在游戏页面
            showGamePage();
            updateGameState(message.raw_data.game_state);
            GameUtils.showToast('游戏状态已恢复', 'success');
        }
    });
    
    wsManager.on('game_started', (message) => {
        console.log('游戏已开始:', message);
        if (currentRoomId) {
            loadGameState(currentRoomId);
        }
    });

    wsManager.on('game_end', (message) => {
        console.log('收到游戏结束消息:', message);
        console.log('消息原始数据:', message.raw_data);
        
        // 尝试多种方式获取结算数据
        let settlement = null;
        if (message.raw_data) {
            if (message.raw_data.settlement) {
                settlement = message.raw_data.settlement;
            } else if (message.data && message.data.settlement) {
                settlement = message.data.settlement;
            }
        }
        
        if (settlement) {
            console.log('提取到的结算数据:', settlement);
            showSettlement(settlement);
            
            // 如果有游戏状态，也更新一下
            if (message.raw_data && message.raw_data.game_state) {
                updateGameState(message.raw_data.game_state);
            }
            
            GameUtils.showToast('游戏已结束，请查看结算结果', 'info');
        } else {
            console.warn('未找到结算数据:', message);
            // 即使没有结算数据，也显示提示
            GameUtils.showToast('游戏已结束', 'info');
        }
    });

    wsManager.on('timer_start', (message) => {
        if (message.raw_data) {
            startTimer(message.raw_data.timeout, message.raw_data.start_time);
        }
    });

    wsManager.on('timer_stop', () => {
        stopTimer();
    });
}

// 加载房间列表
async function loadRooms(gameType = '') {
    const roomsList = document.getElementById('rooms-list');
    roomsList.innerHTML = '<div class="loading">加载中...</div>';
    
    try {
        const result = await GameAPI.getRoomList(gameType);
        const rooms = result.data || [];
        
        if (rooms.length === 0) {
            roomsList.innerHTML = '<div class="loading">暂无房间</div>';
            return;
        }

        roomsList.innerHTML = '';
        rooms.forEach(room => {
            const roomCard = createRoomCard(room);
            roomsList.appendChild(roomCard);
        });
    } catch (error) {
        roomsList.innerHTML = `<div class="loading" style="color: #e74c3c;">加载失败: ${error.message}</div>`;
    }
}

// 添加房间到列表（用于WebSocket推送的新房间）
function addRoomToList(room) {
    console.log('addRoomToList 被调用，房间:', room);
    const roomsList = document.getElementById('rooms-list');
    if (!roomsList) {
        console.warn('rooms-list 元素不存在');
        return;
    }
    
    // 检查房间是否已存在
    const existingCards = roomsList.querySelectorAll('.room-card');
    for (let card of existingCards) {
        const roomId = card.getAttribute('data-room-id');
        if (roomId === room.room_id) {
            console.log('房间已存在，移除旧卡片:', roomId);
            // 房间已存在，更新它
            roomsList.removeChild(card);
            break;
        }
    }
    
    // 创建新房间卡片并添加到列表顶部
    try {
        const roomCard = createRoomCard(room);
        if (roomsList.firstChild) {
            roomsList.insertBefore(roomCard, roomsList.firstChild);
            console.log('房间已添加到列表顶部');
        } else {
            // 如果列表为空，先清空"暂无房间"提示
            if (roomsList.innerHTML.includes('暂无')) {
                roomsList.innerHTML = '';
            }
            roomsList.appendChild(roomCard);
            console.log('房间已添加到空列表');
        }
    } catch (error) {
        console.error('创建房间卡片失败:', error, room);
    }
}

// 从列表中移除房间（用于WebSocket推送的房间删除）
function removeRoomFromList(roomId) {
    console.log('removeRoomFromList 被调用，房间ID:', roomId);
    const roomsList = document.getElementById('rooms-list');
    if (!roomsList) {
        console.warn('rooms-list 元素不存在');
        return;
    }
    
    // 查找并移除房间卡片
    const existingCards = roomsList.querySelectorAll('.room-card');
    for (let card of existingCards) {
        const cardRoomId = card.getAttribute('data-room-id');
        if (cardRoomId === roomId) {
            console.log('找到房间卡片，移除:', roomId);
            roomsList.removeChild(card);
            
            // 如果列表为空，显示"暂无房间"提示
            if (roomsList.children.length === 0) {
                roomsList.innerHTML = '<div style="text-align: center; color: #999; padding: 40px;">暂无房间</div>';
            }
            return;
        }
    }
    
    console.log('未找到房间卡片:', roomId);
}

// 创建房间卡片
function createRoomCard(room) {
    const card = document.createElement('div');
    card.className = 'room-card';
    card.setAttribute('data-room-id', room.room_id); // 添加data属性便于查找
    
    const statusClass = room.status === 1 ? 'waiting' : room.status === 2 ? 'playing' : 'ended';
    const statusText = room.status === 1 ? '等待中' : room.status === 2 ? '游戏中' : '已结束';
    
    // 检查当前用户是否在房间中
    const currentUserId = (window.currentUser() || {})?.id;
    let isInRoom = false;
    if (room.players && Array.isArray(room.players)) {
        isInRoom = room.players.some(p => p.user_id === currentUserId);
    } else if (room.players && typeof room.players === 'string') {
        try {
            const players = JSON.parse(room.players);
            if (Array.isArray(players)) {
                isInRoom = players.some(p => p.user_id === currentUserId);
            }
        } catch (e) {
            console.error('解析玩家列表失败:', e);
        }
    }
    
    card.innerHTML = `
        <div class="room-card-header">
            <div class="room-id">${room.room_id}${isInRoom ? ' (我的房间)' : ''}</div>
            <div class="room-status ${statusClass}">${statusText}</div>
        </div>
        <div class="room-info-item">
            <label>游戏:</label><span>${getGameTypeName(room.game_type)}</span>
        </div>
        <div class="room-info-item">
            <label>底注:</label><span>${room.base_bet}</span>
        </div>
        <div class="room-info-item">
            <label>人数:</label><span>${room.current_players}/${room.max_players}</span>
        </div>
        ${room.has_password ? '<div class="room-info-item"><label>🔒</label><span>密码房间</span></div>' : ''}
    `;
    
    card.addEventListener('click', () => {
        if (room.status === 1) {
            // 如果用户已在房间中，直接进入房间页面
            if (isInRoom) {
                currentRoomId = room.room_id;
                showRoomPage();
                loadRoomInfo(room.room_id);
                wsManager.joinRoom(room.room_id);
            } else {
                joinRoomPrompt(room.room_id, room.has_password);
            }
        } else {
            GameUtils.showToast('房间已开始或已结束');
        }
    });
    
    return card;
}

// 获取游戏类型名称
function getGameTypeName(type) {
    const names = {
        'running': '跑得快',
        'texas': '德州扑克',
        'bull': '牛牛'
    };
    return names[type] || type;
}

// 加入房间提示
async function joinRoomPrompt(roomId, hasPassword) {
    if (hasPassword) {
        showModal('password-modal');
        document.getElementById('password-form').onsubmit = async (e) => {
            e.preventDefault();
            const password = e.target.password.value;
            await joinRoom(roomId, password);
            closeModal('password-modal');
        };
    } else {
        await joinRoom(roomId, '');
    }
}

// 加入房间
async function joinRoom(roomId, password = '') {
    try {
        const result = await GameAPI.joinRoom(roomId, password);
        
        // 即使API返回成功（用户已在房间中），也直接进入房间页面
        currentRoomId = roomId;
        showRoomPage();
        loadRoomInfo(roomId);
        wsManager.joinRoom(roomId);
        
        // 根据返回的消息判断是否需要提示
        if (result && result.message && result.message.includes('已在房间中')) {
            GameUtils.showToast('已进入房间', 'success');
        } else {
            GameUtils.showToast('加入房间成功', 'success');
        }
    } catch (error) {
        // 如果错误是"已在房间中"，也直接进入房间页面
        if (error.message && error.message.includes('已在房间中')) {
            currentRoomId = roomId;
            showRoomPage();
            loadRoomInfo(roomId);
            wsManager.joinRoom(roomId);
            GameUtils.showToast('已进入房间', 'success');
        } else {
            GameUtils.showToast(`加入房间失败: ${error.message}`, 'error');
        }
    }
}

// 加载房间信息
async function loadRoomInfo(roomId) {
    try {
        const result = await GameAPI.getRoom(roomId);
        const room = result.data;
        
        document.getElementById('room-title').textContent = `房间: ${room.room_id}`;
        document.getElementById('room-id').textContent = room.room_id;
        document.getElementById('room-game-type').textContent = getGameTypeName(room.game_type);
        document.getElementById('room-base-bet').textContent = room.base_bet;
        document.getElementById('room-status').textContent = 
            room.status === 1 ? '等待中' : room.status === 2 ? '游戏中' : '已结束';
        
        // 渲染玩家列表
        renderPlayers(room.players || []);
        
        // 更新操作按钮
        updateRoomActions(room);
        
        // 如果房间状态是游戏中，自动加载游戏状态
        if (room.status === 2) {
            console.log('房间已在游戏中，自动加载游戏状态');
            setTimeout(() => {
                loadGameState(roomId);
            }, 300);
        }
    } catch (error) {
        GameUtils.showToast(`加载房间信息失败: ${error.message}`, 'error');
    }
}

// 渲染玩家列表
function renderPlayers(players) {
    const playersList = document.getElementById('players-list');
    if (!playersList) return;
    
    playersList.innerHTML = '';
    
    if (!players) {
        return;
    }
    
    // 处理JSON字符串
    let playersArray = [];
    try {
        if (typeof players === 'string') {
            playersArray = JSON.parse(players);
        } else if (Array.isArray(players)) {
            playersArray = players;
        } else if (typeof players === 'object') {
            playersArray = Object.values(players);
        }
    } catch (e) {
        console.error('解析玩家列表失败:', e);
        return;
    }
    
    const currentUserId = (window.currentUser() || {})?.id;
    
    playersArray.forEach((player, index) => {
        const playerItem = document.createElement('div');
        playerItem.className = 'player-item';
        if (player.user_id === currentUserId) {
            playerItem.classList.add('me');
        }
        
        playerItem.innerHTML = `
            <div>
                <div class="player-name">${player.nickname || `玩家${player.position || index + 1}`}</div>
                <div class="player-status ${player.ready ? 'ready' : 'not-ready'}">
                    ${player.ready ? '已准备' : '未准备'}
                </div>
            </div>
        `;
        
        playersList.appendChild(playerItem);
    });
}

// 更新房间操作按钮
function updateRoomActions(room) {
    if (!room) return;
    
    let players = [];
    try {
        if (typeof room.players === 'string') {
            players = JSON.parse(room.players);
        } else if (Array.isArray(room.players)) {
            players = room.players;
        }
    } catch (e) {
        console.error('解析玩家列表失败:', e);
        players = [];
    }
    
    const currentUserId = (window.currentUser() || {})?.id;
    const myPlayer = players.find(p => p.user_id === currentUserId);
    const allReady = players.length >= 2 && players.every(p => p.ready || p.user_id === currentUserId);
    const isCreator = room.creator_id === currentUserId;
    
    const readyBtn = document.getElementById('ready-btn');
    const cancelReadyBtn = document.getElementById('cancel-ready-btn');
    const startBtn = document.getElementById('start-game-btn');
    
    if (myPlayer && myPlayer.ready) {
        readyBtn.style.display = 'none';
        cancelReadyBtn.style.display = 'block';
        startBtn.style.display = isCreator && allReady && room.status === 1 ? 'block' : 'none';
    } else {
        readyBtn.style.display = room.status === 1 ? 'block' : 'none';
        cancelReadyBtn.style.display = 'none';
        startBtn.style.display = 'none';
    }
}

// 初始化房间操作
function initRoomActions() {
    document.getElementById('ready-btn').addEventListener('click', async () => {
        if (!currentRoomId) return;
        try {
            await GameAPI.ready(currentRoomId);
            GameUtils.showToast('已准备', 'success');
            loadRoomInfo(currentRoomId);
        } catch (error) {
            GameUtils.showToast(`操作失败: ${error.message}`, 'error');
        }
    });

    document.getElementById('cancel-ready-btn').addEventListener('click', async () => {
        if (!currentRoomId) return;
        try {
            await GameAPI.cancelReady(currentRoomId);
            GameUtils.showToast('已取消准备', 'success');
            loadRoomInfo(currentRoomId);
        } catch (error) {
            GameUtils.showToast(`操作失败: ${error.message}`, 'error');
        }
    });

    document.getElementById('start-game-btn').addEventListener('click', async () => {
        if (!currentRoomId) return;
        try {
            console.log('开始游戏，房间ID:', currentRoomId);
            const result = await GameAPI.startGame(currentRoomId);
            console.log('开始游戏返回:', result);
            
            // 如果返回了游戏状态，直接更新
            if (result && result.data && result.data.game_state) {
                console.log('API返回了游戏状态，直接更新');
                showGamePage();
                updateGameState(result.data.game_state);
                GameUtils.showToast('游戏开始', 'success');
            } else {
                // 如果没有返回游戏状态，等待一下再加载状态（确保后端已创建游戏状态）
                console.log('API未返回游戏状态，延迟加载');
                setTimeout(() => {
                    loadGameState(currentRoomId);
                }, 500);
                GameUtils.showToast('游戏开始', 'success');
            }
        } catch (error) {
            console.error('开始游戏失败:', error);
            GameUtils.showToast(`开始游戏失败: ${error.message}`, 'error');
        }
    });

    document.getElementById('leave-room-btn').addEventListener('click', async () => {
        if (!currentRoomId) return;
        if (confirm('确定要离开房间吗？')) {
            try {
                await GameAPI.leaveRoom(currentRoomId);
                wsManager.leaveRoom();
                currentRoomId = null;
                showMainPage();
                GameUtils.showToast('已离开房间', 'success');
            } catch (error) {
                GameUtils.showToast(`离开房间失败: ${error.message}`, 'error');
            }
        }
    });
}

// 显示充值页面
function showRechargePage() {
    // 隐藏所有视图
    document.querySelectorAll('.view').forEach(view => {
        view.classList.remove('active');
        view.style.display = 'none';
    });
    
    // 显示充值视图
    const rechargeView = document.getElementById('recharge-view');
    if (rechargeView) {
        rechargeView.classList.add('active');
        rechargeView.style.display = 'block';
        loadRechargeOrders();
    }
    
    // 更新导航按钮状态
    document.querySelectorAll('.nav-btn').forEach(btn => {
        btn.classList.remove('active');
    });
    const rechargeBtn = document.querySelector('[data-view="recharge"]');
    if (rechargeBtn) {
        rechargeBtn.classList.add('active');
    }
}

// 初始化充值功能
function initRecharge() {
    // 创建充值订单表单
    const rechargeForm = document.getElementById('recharge-form');
    if (rechargeForm) {
        rechargeForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const amount = parseFloat(document.getElementById('recharge-amount').value);
            const chainType = document.getElementById('recharge-chain-type').value;
            
            if (!amount || amount <= 0) {
                GameUtils.showToast('请输入有效的充值金额', 'error');
                return;
            }
            
            if (!chainType) {
                GameUtils.showToast('请选择链类型', 'error');
                return;
            }
            
            try {
                const result = await PaymentAPI.createRechargeOrder(amount, chainType);
                if (result && result.data) {
                    showRechargeOrderDetail(result.data);
                    GameUtils.showToast('充值订单创建成功', 'success');
                    loadRechargeOrders(); // 刷新订单列表
                }
            } catch (error) {
                console.error('创建充值订单失败:', error);
                GameUtils.showToast(`创建订单失败: ${error.message}`, 'error');
            }
        });
    }
    
    // 检查交易按钮
    const checkOrderBtn = document.getElementById('check-order-btn');
    if (checkOrderBtn) {
        checkOrderBtn.addEventListener('click', async () => {
            const orderDetailEl = document.getElementById('recharge-order-detail');
            const orderId = orderDetailEl?.getAttribute('data-order-id');
            if (orderId) {
                try {
                    const result = await PaymentAPI.checkRechargeTransaction(orderId);
                    if (result && result.data) {
                        showRechargeOrderDetail(result.data);
                        GameUtils.showToast('交易检查完成', 'success');
                        
                        // 如果已支付，刷新用户信息和订单列表
                        if (result.data.status === 2) {
                            loadUserInfo();
                            loadRechargeOrders();
                        }
                    }
                } catch (error) {
                    console.error('检查交易失败:', error);
                    GameUtils.showToast(`检查失败: ${error.message}`, 'error');
                }
            }
        });
    }
    
    // 刷新订单按钮
    const refreshOrderBtn = document.getElementById('refresh-order-btn');
    if (refreshOrderBtn) {
        refreshOrderBtn.addEventListener('click', () => {
            const orderDetailEl = document.getElementById('recharge-order-detail');
            const orderId = orderDetailEl?.getAttribute('data-order-id');
            if (orderId) {
                loadRechargeOrder(orderId);
            }
        });
    }
}

// 显示充值订单详情
function showRechargeOrderDetail(order) {
    const orderDetailEl = document.getElementById('recharge-order-detail');
    const orderInfoEl = document.getElementById('order-info');
    
    if (!orderDetailEl || !orderInfoEl) return;
    
    orderDetailEl.style.display = 'block';
    orderDetailEl.setAttribute('data-order-id', order.order_id);
    
    const statusText = order.status === 1 ? '待支付' : order.status === 2 ? '已支付' : '已取消';
    const statusClass = order.status === 1 ? 'warning' : order.status === 2 ? 'success' : 'error';
    
    let html = `
        <div class="info-item">
            <label>订单号:</label>
            <span>${order.order_id}</span>
        </div>
        <div class="info-item">
            <label>充值金额:</label>
            <span style="color: #27ae60; font-weight: bold;">${order.amount} USDT</span>
        </div>
        <div class="info-item">
            <label>链类型:</label>
            <span>${order.chain_type === 'trc20' ? 'TRC20' : 'ERC20'}</span>
        </div>
        <div class="info-item">
            <label>状态:</label>
            <span class="${statusClass}">${statusText}</span>
        </div>
        <div class="info-item">
            <label>充值地址:</label>
            <div style="word-break: break-all; background: #f8f9fa; padding: 10px; border-radius: 4px; margin-top: 5px;">
                <code style="font-size: 12px;">${order.deposit_addr}</code>
                <button onclick="copyToClipboard('${order.deposit_addr}')" class="btn btn-small" style="margin-left: 10px;">复制</button>
            </div>
        </div>
    `;
    
    if (order.tx_hash) {
        const txHashLink = order.chain_type === 'trc20' 
            ? `https://tronscan.org/#/transaction/${order.tx_hash}`
            : `https://etherscan.io/tx/${order.tx_hash}`;
        html += `
            <div class="info-item">
                <label>交易哈希:</label>
                <a href="${txHashLink}" target="_blank" style="word-break: break-all; color: #3498db;">
                    ${order.tx_hash}
                </a>
            </div>
            <div class="info-item">
                <label>确认次数:</label>
                <span>${order.confirm_count || 0} / ${order.required_conf || 12}</span>
            </div>
        `;
    }
    
    if (order.expire_at) {
        const expireTime = new Date(order.expire_at * 1000).toLocaleString('zh-CN');
        html += `
            <div class="info-item">
                <label>过期时间:</label>
                <span>${expireTime}</span>
            </div>
        `;
    }
    
    orderInfoEl.innerHTML = html;
}

// 加载充值订单详情
async function loadRechargeOrder(orderId) {
    try {
        const result = await PaymentAPI.getRechargeOrder(orderId);
        if (result && result.data) {
            showRechargeOrderDetail(result.data);
        }
    } catch (error) {
        console.error('加载订单详情失败:', error);
        GameUtils.showToast(`加载失败: ${error.message}`, 'error');
    }
}

// 加载充值订单列表
async function loadRechargeOrders() {
    const ordersListEl = document.getElementById('recharge-orders-list');
    if (!ordersListEl) return;
    
    try {
        ordersListEl.innerHTML = '<div class="loading">加载中...</div>';
        const result = await PaymentAPI.getRechargeOrders(1, 20);
        
        if (result && result.data && result.data.orders) {
            const orders = result.data.orders;
            
            if (orders.length === 0) {
                ordersListEl.innerHTML = '<div style="text-align: center; color: #999; padding: 20px;">暂无充值记录</div>';
                return;
            }
            
            let html = '<div style="overflow-x: auto;"><table style="width: 100%; border-collapse: collapse;">';
            html += '<thead><tr>';
            html += '<th style="padding: 10px; border: 1px solid #ddd;">订单号</th>';
            html += '<th style="padding: 10px; border: 1px solid #ddd;">金额</th>';
            html += '<th style="padding: 10px; border: 1px solid #ddd;">链类型</th>';
            html += '<th style="padding: 10px; border: 1px solid #ddd;">状态</th>';
            html += '<th style="padding: 10px; border: 1px solid #ddd;">创建时间</th>';
            html += '<th style="padding: 10px; border: 1px solid #ddd;">操作</th>';
            html += '</tr></thead><tbody>';
            
            orders.forEach(order => {
                const statusText = order.status === 1 ? '待支付' : order.status === 2 ? '已支付' : '已取消';
                const statusClass = order.status === 1 ? 'warning' : order.status === 2 ? 'success' : 'error';
                const createTime = new Date(order.created_at * 1000).toLocaleString('zh-CN');
                
                html += `<tr>`;
                html += `<td style="padding: 10px; border: 1px solid #ddd;">${order.order_id}</td>`;
                html += `<td style="padding: 10px; border: 1px solid #ddd;">${order.amount} USDT</td>`;
                html += `<td style="padding: 10px; border: 1px solid #ddd;">${order.chain_type === 'trc20' ? 'TRC20' : 'ERC20'}</td>`;
                html += `<td style="padding: 10px; border: 1px solid #ddd;"><span class="${statusClass}">${statusText}</span></td>`;
                html += `<td style="padding: 10px; border: 1px solid #ddd;">${createTime}</td>`;
                html += `<td style="padding: 10px; border: 1px solid #ddd;">`;
                // 使用安全的转义函数
                const orderJson = JSON.stringify(order).replace(/'/g, "&#39;").replace(/"/g, "&quot;");
                html += `<button onclick="window.loadRechargeOrderSafe('${order.order_id}')" class="btn btn-small">查看</button>`;
                html += `</td>`;
                html += `</tr>`;
            });
            
            html += '</tbody></table></div>';
            ordersListEl.innerHTML = html;
        } else {
            ordersListEl.innerHTML = '<div style="text-align: center; color: #999; padding: 20px;">暂无充值记录</div>';
        }
    } catch (error) {
        console.error('加载充值订单列表失败:', error);
        ordersListEl.innerHTML = `<div style="color: #e74c3c; text-align: center; padding: 20px;">加载失败: ${error.message}</div>`;
    }
}

// 安全的加载订单函数（供onclick使用）
window.loadRechargeOrderSafe = async function(orderId) {
    await loadRechargeOrder(orderId);
};

// 复制到剪贴板
function copyToClipboard(text) {
    navigator.clipboard.writeText(text).then(() => {
        GameUtils.showToast('已复制到剪贴板', 'success');
    }).catch(() => {
        // 降级方案
        const textArea = document.createElement('textarea');
        textArea.value = text;
        document.body.appendChild(textArea);
        textArea.select();
        document.execCommand('copy');
        document.body.removeChild(textArea);
        GameUtils.showToast('已复制到剪贴板', 'success');
    });
}

// 导出到全局作用域
window.copyToClipboard = copyToClipboard;

// 初始化创建房间
function initCreateRoom() {
    document.getElementById('create-room-btn').addEventListener('click', () => {
        showModal('create-room-modal');
    });

    document.getElementById('create-room-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        const formData = new FormData(e.target);
        
        const data = {
            game_type: formData.get('game_type'),
            room_type: formData.get('room_type'),
            base_bet: parseFloat(formData.get('base_bet')),
            max_players: parseInt(formData.get('max_players')),
            password: formData.get('password') || ''
        };

        try {
            const result = await GameAPI.createRoom(data);
            const roomId = result.data.room_id;
            closeModal('create-room-modal');
            await joinRoom(roomId, data.password);
        } catch (error) {
            GameUtils.showToast(`创建房间失败: ${error.message}`, 'error');
        }
    });
}

// 加载游戏状态
async function loadGameState(roomId) {
    showGamePage();
    try {
        const result = await GameAPI.getGameState(roomId);
        if (result.data) {
            updateGameState(result.data);
        }
    } catch (error) {
        GameUtils.showToast(`加载游戏状态失败: ${error.message}`, 'error');
    }
}

// 更新游戏状态
function updateGameState(gameState) {
    console.log('更新游戏状态:', gameState);
    currentGameState = gameState;
    
    const gameRoomIdEl = document.getElementById('game-room-id');
    const gameRoundEl = document.getElementById('game-round');
    if (gameRoomIdEl) gameRoomIdEl.textContent = gameState.room_id || currentRoomId || '';
    if (gameRoundEl) gameRoundEl.textContent = gameState.round || 0;
    
    const currentUserId = (window.currentUser() || {})?.id;
    console.log('当前用户ID:', currentUserId);
    console.log('游戏状态中的玩家:', gameState.players);
    
    // 渲染其他玩家
    renderOpponents(gameState.players || {}, currentUserId);
    
    // 渲染我的手牌
    if (gameState.players) {
        let myPlayer = null;
        
        // 处理players可能是对象
        if (currentUserId && gameState.players[currentUserId]) {
            myPlayer = gameState.players[currentUserId];
            console.log('找到我的玩家信息 (by key):', myPlayer);
        } else if (Array.isArray(gameState.players)) {
            myPlayer = gameState.players.find(p => p.user_id === currentUserId);
            console.log('找到我的玩家信息 (from array):', myPlayer);
        } else if (typeof gameState.players === 'object') {
            const playersArray = Object.values(gameState.players);
            myPlayer = playersArray.find(p => p && p.user_id === currentUserId);
            console.log('找到我的玩家信息 (from object values):', myPlayer);
        }
        
        if (myPlayer) {
            const myCards = myPlayer.cards || [];
            console.log('我的手牌:', myCards);
            const cardCountEl = document.getElementById('my-card-count');
            if (cardCountEl) {
                cardCountEl.textContent = myCards.length;
            }
            renderMyCards(myCards);
        } else {
            console.warn('未找到我的玩家信息');
            const cardCountEl = document.getElementById('my-card-count');
            if (cardCountEl) {
                cardCountEl.textContent = '0';
            }
            renderMyCards([]);
        }
    } else {
        console.warn('游戏状态中没有玩家信息');
    }
    
    // 渲染上次出的牌（牛牛游戏不显示）
    const isBullGame = gameState.game_type === 'bull';
    const lastCardsEl = document.getElementById('last-cards-display');
    const lastPlayerEl = document.getElementById('last-player-name');
    
    if (isBullGame) {
        // 牛牛游戏：隐藏上次出牌区域，或显示所有玩家的出牌结果
        if (lastCardsEl) {
            lastCardsEl.innerHTML = '<div style="color: #999;">牛牛游戏：选择5张牌进行结算</div>';
        }
        if (lastPlayerEl) {
            lastPlayerEl.textContent = '';
        }
    } else {
        // 其他游戏：正常显示上次出的牌
        if (gameState.last_cards && gameState.last_cards.length > 0) {
            GameUtils.renderPlayedCards(
                lastCardsEl,
                gameState.last_cards
            );
            const lastPlayerId = gameState.last_player;
            if (gameState.players && gameState.players[lastPlayerId]) {
                lastPlayerEl.textContent = `玩家${gameState.players[lastPlayerId].position}`;
            }
        } else {
            if (lastCardsEl) {
                lastCardsEl.innerHTML = '<div style="color: #999;">暂无</div>';
            }
            if (lastPlayerEl) {
                lastPlayerEl.textContent = '';
            }
        }
    }
    
    // 更新操作按钮状态
    const isMyTurn = gameState.current_player === currentUserId;
    const canPass = !isBullGame && gameState.last_cards && gameState.last_cards.length > 0; // 牛牛游戏不能过牌
    
    console.log('操作按钮状态:', { isMyTurn, canPass, currentPlayer: gameState.current_player, myUserId: currentUserId, isBullGame });
    
    const passBtn = document.getElementById('pass-btn');
    const playBtn = document.getElementById('play-btn');
    if (passBtn) {
        // 牛牛游戏隐藏过牌按钮，其他游戏显示
        if (isBullGame) {
            passBtn.style.display = 'none';
        } else {
            passBtn.style.display = 'inline-block';
            passBtn.disabled = !isMyTurn || !canPass;
        }
        console.log('过牌按钮:', passBtn.disabled ? '禁用' : '启用');
    }
    if (playBtn) {
        playBtn.disabled = !isMyTurn;
        // 更新按钮文本
        if (isBullGame) {
            playBtn.textContent = '确定（选择5张牌）';
        } else {
            playBtn.textContent = '出牌';
        }
        console.log('出牌按钮:', playBtn.disabled ? '禁用' : '启用');
    }
}

// 渲染其他玩家
function renderOpponents(players, currentUserId) {
    const opponentsArea = document.getElementById('opponents-area');
    if (!opponentsArea) return;
    
    opponentsArea.innerHTML = '';
    
    if (!players) return;
    
    // 处理players可能是对象或数组
    let playersArray = [];
    if (Array.isArray(players)) {
        playersArray = players;
    } else if (typeof players === 'object') {
        playersArray = Object.values(players);
    }
    
    // 检查是否是牛牛游戏
    const isBullGame = currentGameState && currentGameState.game_type === 'bull';
    
    playersArray.forEach(player => {
        if (!player || player.user_id === currentUserId) return;
        
        const opponentCard = document.createElement('div');
        opponentCard.className = 'opponent-card';
        
        const status = player.is_passed ? '已过' : 
                      player.is_finished ? '已出完' : 
                      player.user_id === currentGameState?.current_player ? '出牌中' : '等待中';
        
        if (isBullGame) {
            // 牛牛游戏：显示牛牛结果
            let bullText = '';
            if (player.bull_type !== undefined && player.bull_type !== null) {
                if (player.bull_type === 0) {
                    bullText = '无牛';
                } else if (player.bull_type === 10) {
                    bullText = '牛牛';
                } else if (player.bull_type === 11) {
                    bullText = '四花';
                } else if (player.bull_type === 12) {
                    bullText = '五花';
                } else if (player.bull_type === 13) {
                    bullText = '炸弹';
                } else if (player.bull_type === 14) {
                    bullText = '五小牛';
                } else if (player.bull_type >= 1 && player.bull_type <= 9) {
                    bullText = `${player.bull_num || player.bull_type}牛`;
                } else {
                    bullText = '等待中';
                }
            } else if (player.is_finished) {
                bullText = '已出牌';
            } else {
                bullText = `${player.card_count || 5}张`;
            }
            
            opponentCard.innerHTML = `
                <div class="opponent-name">玩家${player.position || player.user_id}</div>
                <div class="opponent-card-count" style="font-size: 16px; font-weight: bold; color: ${player.is_finished ? '#27ae60' : '#3498db'};">
                    ${bullText}
                </div>
                <div class="opponent-status">${status}</div>
            `;
        } else {
            // 其他游戏：正常显示
            opponentCard.innerHTML = `
                <div class="opponent-name">玩家${player.position || player.user_id}</div>
                <div class="opponent-card-count">${player.card_count || player.cards?.length || 0}张</div>
                <div class="opponent-status">${status}</div>
            `;
        }
        
        opponentsArea.appendChild(opponentCard);
    });
}

// 渲染我的手牌
function renderMyCards(cards, preserveSelection = false) {
    const container = document.getElementById('my-cards');
    if (!container) return;
    
    // 只在第一次渲染时清空selectedCards（不保留选择状态）
    if (!preserveSelection) {
        selectedCards = [];
    }
    
    if (!cards || !Array.isArray(cards) || cards.length === 0) {
        container.innerHTML = '<div style="color: #999; text-align: center; padding: 20px;">暂无手牌</div>';
        selectedCards = []; // 清空选择
        return;
    }
    
    // 检查是否是牛牛游戏
    const isBullGame = currentGameState && currentGameState.game_type === 'bull';
    
    if (isBullGame) {
        // 牛牛游戏：显示5张牌和牛牛结果
        container.innerHTML = '';
        
        // 创建手牌容器
        const cardsWrapper = document.createElement('div');
        cardsWrapper.style.display = 'flex';
        cardsWrapper.style.gap = '10px';
        cardsWrapper.style.marginBottom = '10px';
        cardsWrapper.style.justifyContent = 'center';
        
        // 渲染5张牌
        const sortedCards = GameUtils.sortCards(cards);
        sortedCards.forEach(cardValue => {
            const isSelected = selectedCards.includes(cardValue);
            const cardElement = GameUtils.createCardElement(cardValue, true, isSelected);
            cardElement.addEventListener('click', () => {
                toggleCardSelection(cardValue);
            });
            cardsWrapper.appendChild(cardElement);
        });
        
        container.appendChild(cardsWrapper);
        
        // 显示牛牛结果区域
        const bullResultArea = document.createElement('div');
        bullResultArea.id = 'bull-result-area';
        bullResultArea.style.cssText = 'text-align: center; margin-top: 10px;';
        container.appendChild(bullResultArea);
        
        // 计算并显示牛牛结果（如果已选择5张牌）
        if (selectedCards.length === 5) {
            const bullResult = GameUtils.calculateBull(selectedCards);
            bullResultArea.innerHTML = `
                <div style="font-size: 18px; font-weight: bold; color: #e74c3c;">
                    当前选择：${bullResult.text}
                </div>
            `;
        } else if (cards.length === 5) {
            // 自动计算并显示所有5张牌的牛牛
            const bullResult = GameUtils.calculateBull(cards);
            bullResultArea.innerHTML = `
                <div style="font-size: 18px; font-weight: bold; color: #27ae60;">
                    你的牛牛：${bullResult.text}
                </div>
                <div style="font-size: 14px; color: #7f8c8d; margin-top: 5px;">
                    （请选择5张牌进行结算，已选择${selectedCards.length}张）
                </div>
            `;
        }
    } else {
        // 其他游戏：正常渲染手牌
        GameUtils.renderCards(container, cards, selectedCards, (cardValue) => {
            toggleCardSelection(cardValue);
        });
    }
}

// 切换牌的选择状态
function toggleCardSelection(cardValue) {
    const index = selectedCards.indexOf(cardValue);
    if (index > -1) {
        selectedCards.splice(index, 1);
    } else {
        // 牛牛游戏：最多选择5张牌
        const isBullGame = currentGameState && currentGameState.game_type === 'bull';
        if (isBullGame && selectedCards.length >= 5) {
            GameUtils.showToast('牛牛游戏只能选择5张牌', 'error');
            return;
        }
        selectedCards.push(cardValue);
    }
    
    // 更新UI
    const getCurrentUser = window.currentUser || (() => null);
    const currentUserId = (getCurrentUser() || {})?.id;
    
    // 从游戏状态中获取手牌
    let cards = [];
    if (currentGameState && currentGameState.players) {
        if (currentGameState.players[currentUserId]) {
            cards = currentGameState.players[currentUserId].cards || [];
        } else if (Array.isArray(currentGameState.players)) {
            const myPlayer = currentGameState.players.find(p => p.user_id === currentUserId);
            if (myPlayer) cards = myPlayer.cards || [];
        } else if (typeof currentGameState.players === 'object') {
            const myPlayer = Object.values(currentGameState.players).find(p => p.user_id === currentUserId);
            if (myPlayer) cards = myPlayer.cards || [];
        }
    }
    
    // 重新渲染手牌（对于牛牛游戏会显示牛牛结果，保留选择状态）
    renderMyCards(cards, true);
}

// 初始化游戏操作
function initGameActions() {
    const playBtn = document.getElementById('play-btn');
    const passBtn = document.getElementById('pass-btn');
    
    if (playBtn) {
        playBtn.addEventListener('click', async () => {
            console.log('点击出牌按钮，选中的牌:', selectedCards);
            if (!currentRoomId) {
                GameUtils.showToast('请先进入房间', 'error');
                return;
            }
            // 检查是否是牛牛游戏
            const isBullGame = currentGameState && currentGameState.game_type === 'bull';
            
            if (isBullGame) {
                // 牛牛游戏：必须选择5张牌
                if (selectedCards.length !== 5) {
                    GameUtils.showToast('牛牛游戏必须选择5张牌', 'error');
                    return;
                }
            } else {
                // 其他游戏：至少选择1张牌
                if (selectedCards.length === 0) {
                    GameUtils.showToast('请选择要出的牌', 'error');
                    return;
                }
            }
            
            try {
                console.log('调用出牌API:', currentRoomId, selectedCards);
                const result = await GameAPI.playCards(currentRoomId, selectedCards);
                console.log('出牌API返回:', result);
                
                selectedCards = [];
                
                // 检查游戏是否结束
                if (result.game_end && result.data && result.data.settlement) {
                    console.log('游戏结束，显示结算页面');
                    // 显示结算弹窗
                    showSettlement(result.data.settlement);
                    // 更新游戏状态（如果存在）
                    if (result.data.game_state) {
                        updateGameState(result.data.game_state);
                    }
                    GameUtils.showToast('游戏已结束', 'success');
                    return;
                }
                
                // 更新游戏状态
                if (result.data) {
                    updateGameState(result.data);
                } else {
                    // 如果没有返回游戏状态，重新加载
                    setTimeout(() => {
                        loadGameState(currentRoomId);
                    }, 500);
                }
                
                GameUtils.showToast('出牌成功', 'success');
            } catch (error) {
                console.error('出牌失败:', error);
                GameUtils.showToast(`出牌失败: ${error.message}`, 'error');
            }
        });
    }

    if (passBtn) {
        passBtn.addEventListener('click', async () => {
            console.log('点击过牌按钮');
            if (!currentRoomId) {
                GameUtils.showToast('请先进入房间', 'error');
                return;
            }
            
            try {
                console.log('调用过牌API:', currentRoomId);
                const result = await GameAPI.pass(currentRoomId);
                console.log('过牌API返回:', result);
                
                // 更新游戏状态
                if (result.data) {
                    updateGameState(result.data);
                } else {
                    // 如果没有返回游戏状态，重新加载
                    setTimeout(() => {
                        loadGameState(currentRoomId);
                    }, 500);
                }
                
                GameUtils.showToast('过牌成功', 'success');
            } catch (error) {
                console.error('过牌失败:', error);
                GameUtils.showToast(`过牌失败: ${error.message}`, 'error');
            }
        });
    }
}

// 加载排行榜
async function loadLeaderboard() {
    const gameType = document.getElementById('lb-game-type').value;
    const period = document.getElementById('lb-period').value;
    
    try {
        const result = await GameAPI.getLeaderboard(gameType, period);
        const leaderboard = result.data;
        
        renderLeaderboard(leaderboard?.rankings || []);
    } catch (error) {
        GameUtils.showToast(`加载排行榜失败: ${error.message}`, 'error');
    }
}

// 渲染排行榜
function renderLeaderboard(rankings) {
    const list = document.getElementById('leaderboard-list');
    list.innerHTML = '';
    
    if (rankings.length === 0) {
        list.innerHTML = '<div class="loading">暂无数据</div>';
        return;
    }
    
    rankings.forEach((item, index) => {
        const itemEl = document.createElement('div');
        itemEl.className = 'leaderboard-item';
        
        let rankClass = '';
        if (item.rank === 1) rankClass = 'first';
        else if (item.rank === 2) rankClass = 'second';
        else if (item.rank === 3) rankClass = 'third';
        
        itemEl.innerHTML = `
            <div class="rank ${rankClass}">${item.rank}</div>
            <div class="user-avatar">${item.nickname?.[0] || '?'}</div>
            <div class="user-name">${item.nickname || `用户${item.user_id}`}</div>
            <div class="score">${item.score || 0}</div>
        `;
        
        list.appendChild(itemEl);
    });
}

// 加载记录
async function loadRecords() {
    try {
        const result = await GameAPI.getMyRecords();
        const records = result.data?.records || [];
        
        renderRecords(records);
    } catch (error) {
        GameUtils.showToast(`加载记录失败: ${error.message}`, 'error');
    }
}

// 渲染记录
function renderRecords(records) {
    const list = document.getElementById('records-list');
    list.innerHTML = '';
    
    if (records.length === 0) {
        list.innerHTML = '<div class="loading">暂无记录</div>';
        return;
    }
    
    records.forEach(record => {
        const itemEl = document.createElement('div');
        itemEl.className = 'record-item';
        
        const balance = record.my_balance || 0;
        const balanceClass = balance >= 0 ? 'positive' : 'negative';
        const balanceSign = balance >= 0 ? '+' : '';
        
        itemEl.innerHTML = `
            <div class="record-info">
                <div class="record-title">${getGameTypeName(record.game_type)} - ${record.room_id}</div>
                <div class="record-meta">
                    ${GameUtils.formatTime(record.start_time)} | 第${record.my_rank || '?'}名
                </div>
            </div>
            <div class="record-result">
                <div class="record-balance ${balanceClass}">${balanceSign}${balance}</div>
            </div>
        `;
        
        list.appendChild(itemEl);
    });
}

// 显示结算
function showSettlement(settlement) {
    const modal = document.getElementById('settlement-modal');
    const result = document.getElementById('settlement-result');
    
    result.innerHTML = '';
    
    if (settlement.players) {
        const playersList = Object.values(settlement.players);
        playersList.sort((a, b) => a.rank - b.rank);
        
        playersList.forEach(player => {
            const item = document.createElement('div');
            item.style.margin = '10px 0';
            item.style.padding = '10px';
            item.style.background = '#f8f9fa';
            item.style.borderRadius = '6px';
            
            const balance = player.balance || 0;
            const balanceClass = balance >= 0 ? 'positive' : 'negative';
            const balanceSign = balance >= 0 ? '+' : '';
            
            item.innerHTML = `
                <div style="display: flex; justify-content: space-between;">
                    <span>第${player.rank}名</span>
                    <span class="${balanceClass}">${balanceSign}${balance}</span>
                </div>
            `;
            
            result.appendChild(item);
        });
    }
    
    showModal('settlement-modal');
}

// 计时器
let timerInterval = null;
function startTimer(timeout, startTime) {
    stopTimer();
    
    const timerEl = document.getElementById('game-timer');
    let remaining = timeout;
    
    if (startTime) {
        const elapsed = Math.floor((Date.now() / 1000) - startTime);
        remaining = Math.max(0, timeout - elapsed);
    }
    
    timerEl.textContent = remaining;
    
    timerInterval = setInterval(() => {
        remaining--;
        if (remaining <= 0) {
            stopTimer();
            timerEl.textContent = '0';
            timerEl.style.background = '#e74c3c';
        } else {
            timerEl.textContent = remaining;
            if (remaining <= 10) {
                timerEl.style.background = '#e74c3c';
            } else {
                timerEl.style.background = '#3498db';
            }
        }
    }, 1000);
}

function stopTimer() {
    if (timerInterval) {
        clearInterval(timerInterval);
        timerInterval = null;
    }
    document.getElementById('game-timer').style.background = '#95a5a6';
}

// 弹窗控制
function showModal(modalId) {
    document.getElementById(modalId).classList.add('show');
}

function closeModal(modalId) {
    document.getElementById(modalId).classList.remove('show');
}

// 初始化大厅
function initLobby() {
    // 刷新房间列表
    document.getElementById('refresh-rooms-btn').addEventListener('click', () => {
        const gameType = document.getElementById('game-type-select').value;
        loadRooms(gameType);
    });

    // 游戏类型筛选
    document.getElementById('game-type-select').addEventListener('change', (e) => {
        loadRooms(e.target.value);
    });

    // 创建房间
    initCreateRoom();
}

// 显示提现页面
function showWithdrawPage() {
    // 隐藏所有视图
    document.querySelectorAll('.view').forEach(view => {
        view.classList.remove('active');
        view.style.display = 'none';
    });
    
    // 显示提现视图
    const withdrawView = document.getElementById('withdraw-view');
    if (withdrawView) {
        withdrawView.classList.add('active');
        withdrawView.style.display = 'block';
        loadWithdrawOrders();
    }
    
    // 更新导航按钮状态
    document.querySelectorAll('.nav-btn').forEach(btn => {
        btn.classList.remove('active');
    });
    const withdrawBtn = document.querySelector('[data-view="withdraw"]');
    if (withdrawBtn) {
        withdrawBtn.classList.add('active');
    }
}

// 初始化提现功能
function initWithdraw() {
    // 创建提现订单表单
    const withdrawForm = document.getElementById('withdraw-form');
    if (withdrawForm) {
        withdrawForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const amount = parseFloat(document.getElementById('withdraw-amount').value);
            const chainType = document.getElementById('withdraw-chain-type').value;
            const toAddress = document.getElementById('withdraw-to-address').value.trim();
            
            if (!amount || amount <= 0) {
                GameUtils.showToast('请输入有效的提现金额', 'error');
                return;
            }
            
            if (amount < 10) {
                GameUtils.showToast('最小提现金额为10 USDT', 'error');
                return;
            }
            
            if (!chainType) {
                GameUtils.showToast('请选择链类型', 'error');
                return;
            }
            
            if (!toAddress) {
                GameUtils.showToast('请输入提现地址', 'error');
                return;
            }
            
            // 验证地址格式
            if (chainType === 'trc20' && (!toAddress.startsWith('T') || toAddress.length !== 34)) {
                GameUtils.showToast('TRC20地址格式错误，应为T开头的34位地址', 'error');
                return;
            }
            
            if (chainType === 'erc20' && (!toAddress.startsWith('0x') || toAddress.length !== 42)) {
                GameUtils.showToast('ERC20地址格式错误，应为0x开头的42位地址', 'error');
                return;
            }
            
            try {
                const result = await PaymentAPI.createWithdrawOrder(amount, chainType, toAddress);
                if (result && result.data) {
                    GameUtils.showToast('提现订单创建成功，等待审核', 'success');
                    loadWithdrawOrders(); // 刷新订单列表
                    loadUserInfo(); // 刷新用户信息（余额可能会冻结）
                    // 清空表单
                    withdrawForm.reset();
                }
            } catch (error) {
                console.error('创建提现订单失败:', error);
                GameUtils.showToast(`创建订单失败: ${error.message}`, 'error');
            }
        });
    }
}

// 加载提现订单列表
async function loadWithdrawOrders() {
    const ordersListEl = document.getElementById('withdraw-orders-list');
    if (!ordersListEl) return;
    
    ordersListEl.innerHTML = '<div class="loading">加载中...</div>';
    
    try {
        const result = await PaymentAPI.getWithdrawOrders(1, 20);
        if (result && result.data && result.data.orders) {
            const orders = result.data.orders;
            
            if (orders.length === 0) {
                ordersListEl.innerHTML = '<div style="text-align: center; padding: 20px; color: #666;">暂无提现记录</div>';
                return;
            }
            
            let html = '<table style="width: 100%; border-collapse: collapse; margin-top: 10px;">';
            html += '<thead><tr style="background: #f5f5f5;">';
            html += '<th style="padding: 10px; text-align: left; border: 1px solid #ddd;">订单号</th>';
            html += '<th style="padding: 10px; text-align: left; border: 1px solid #ddd;">金额</th>';
            html += '<th style="padding: 10px; text-align: left; border: 1px solid #ddd;">链类型</th>';
            html += '<th style="padding: 10px; text-align: left; border: 1px solid #ddd;">提现地址</th>';
            html += '<th style="padding: 10px; text-align: left; border: 1px solid #ddd;">状态</th>';
            html += '<th style="padding: 10px; text-align: left; border: 1px solid #ddd;">交易哈希</th>';
            html += '<th style="padding: 10px; text-align: left; border: 1px solid #ddd;">创建时间</th>';
            html += '<th style="padding: 10px; text-align: left; border: 1px solid #ddd;">操作</th>';
            html += '</tr></thead><tbody>';
            
            orders.forEach(order => {
                const statusMap = {
                    1: '<span style="color: #ff9800;">待审核</span>',
                    2: '<span style="color: #4caf50;">已通过</span>',
                    3: '<span style="color: #f44336;">已拒绝</span>'
                };
                
                const createdAt = order.created_at ? new Date(order.created_at * 1000).toLocaleString('zh-CN') : '-';
                const txHash = order.tx_hash || '-';
                const shortAddress = order.to_address ? `${order.to_address.substring(0, 10)}...${order.to_address.substring(order.to_address.length - 8)}` : '-';
                
                html += '<tr>';
                html += `<td style="padding: 10px; border: 1px solid #ddd;">${order.order_id}</td>`;
                html += `<td style="padding: 10px; border: 1px solid #ddd;">${order.amount} USDT</td>`;
                html += `<td style="padding: 10px; border: 1px solid #ddd;">${order.chain_type?.toUpperCase() || '-'}</td>`;
                html += `<td style="padding: 10px; border: 1px solid #ddd;" title="${order.to_address || ''}">${shortAddress}</td>`;
                html += `<td style="padding: 10px; border: 1px solid #ddd;">${statusMap[order.status] || '-'}</td>`;
                html += `<td style="padding: 10px; border: 1px solid #ddd;" title="${txHash}">${txHash !== '-' ? txHash.substring(0, 10) + '...' : '-'}</td>`;
                html += `<td style="padding: 10px; border: 1px solid #ddd;">${createdAt}</td>`;
                html += `<td style="padding: 10px; border: 1px solid #ddd;">
                    <button onclick="window.loadWithdrawOrderSafe('${order.order_id}')" class="btn btn-small">查看</button>
                </td>`;
                html += '</tr>';
            });
            
            html += '</tbody></table>';
            ordersListEl.innerHTML = html;
        } else {
            ordersListEl.innerHTML = '<div style="text-align: center; padding: 20px; color: #666;">暂无提现记录</div>';
        }
    } catch (error) {
        console.error('加载提现订单列表失败:', error);
        ordersListEl.innerHTML = `<div style="text-align: center; padding: 20px; color: #f44336;">加载失败: ${error.message}</div>`;
    }
}

// 加载提现订单详情
async function loadWithdrawOrder(orderId) {
    try {
        const result = await PaymentAPI.getWithdrawOrder(orderId);
        if (result && result.data) {
            const order = result.data;
            let info = '<div style="line-height: 1.8;">';
            info += `<p><strong>订单号:</strong> ${order.order_id}</p>`;
            info += `<p><strong>金额:</strong> ${order.amount} USDT</p>`;
            info += `<p><strong>链类型:</strong> ${order.chain_type?.toUpperCase() || '-'}</p>`;
            info += `<p><strong>提现地址:</strong> ${order.to_address || '-'}</p>`;
            info += `<p><strong>状态:</strong> ${order.status === 1 ? '待审核' : order.status === 2 ? '已通过' : '已拒绝'}</p>`;
            if (order.tx_hash) {
                info += `<p><strong>交易哈希:</strong> ${order.tx_hash}</p>`;
            }
            if (order.remark) {
                info += `<p><strong>备注:</strong> ${order.remark}</p>`;
            }
            if (order.audit_at) {
                info += `<p><strong>审核时间:</strong> ${new Date(order.audit_at * 1000).toLocaleString('zh-CN')}</p>`;
            }
            info += `<p><strong>创建时间:</strong> ${order.created_at ? new Date(order.created_at * 1000).toLocaleString('zh-CN') : '-'}</p>`;
            info += '</div>';
            
            // 这里可以显示订单详情弹窗
            GameUtils.showToast(`订单详情已加载`, 'success');
            console.log('提现订单详情:', order);
        }
    } catch (error) {
        console.error('加载提现订单详情失败:', error);
        GameUtils.showToast(`加载订单失败: ${error.message}`, 'error');
    }
}

// 安全的加载订单函数（供onclick使用）
window.loadWithdrawOrderSafe = async function(orderId) {
    await loadWithdrawOrder(orderId);
};

// 延迟初始化，确保DOM已加载
setTimeout(() => {
    try {
        initNavigation();
        initRoomActions();
        initGameActions();
        initLobby();
        initRecharge();
        initWithdraw();
        console.log('所有功能初始化完成');
    } catch (error) {
        console.error('功能初始化失败:', error);
    }
}, 100);

// 关闭结算弹窗（延迟绑定，确保DOM已加载）
setTimeout(() => {
    const closeBtn = document.getElementById('close-settlement-btn');
    if (closeBtn) {
        closeBtn.addEventListener('click', () => {
            closeModal('settlement-modal');
            showMainPage();
            currentRoomId = null;
        });
    }
}, 200);

// 排行榜筛选
document.getElementById('lb-game-type').addEventListener('change', loadLeaderboard);
document.getElementById('lb-period').addEventListener('change', loadLeaderboard);

