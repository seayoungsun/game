// WebSocket 连接管理
class WebSocketManager {
    constructor() {
        this.ws = null;
        this.reconnectTimer = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.listeners = {};
        this.isConnected = false;
    }

    connect() {
        if (!authToken) {
            console.error('未登录，无法建立WebSocket连接');
            return;
        }

        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            console.log('WebSocket已连接');
            return;
        }

        try {
            this.ws = new WebSocket(`${WS_URL}?token=${authToken}`);

            this.ws.onopen = () => {
                console.log('WebSocket连接成功');
                this.isConnected = true;
                this.reconnectAttempts = 0;
                this.emit('connected', {});
                this.startHeartbeat();
            };

            this.ws.onmessage = (event) => {
                try {
                    const message = JSON.parse(event.data);
                    this.handleMessage(message);
                } catch (error) {
                    console.error('解析WebSocket消息失败:', error);
                }
            };

            this.ws.onerror = (error) => {
                console.error('WebSocket错误:', error);
                this.emit('error', error);
            };

            this.ws.onclose = () => {
                console.log('WebSocket连接关闭');
                this.isConnected = false;
                this.stopHeartbeat();
                this.emit('disconnected', {});
                
                // 自动重连
                if (this.reconnectAttempts < this.maxReconnectAttempts) {
                    this.reconnectAttempts++;
                    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 10000);
                    console.log(`${delay}ms后尝试重连 (${this.reconnectAttempts}/${this.maxReconnectAttempts})`);
                    this.reconnectTimer = setTimeout(() => this.connect(), delay);
                }
            };
        } catch (error) {
            console.error('WebSocket连接失败:', error);
        }
    }

    disconnect() {
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }
        this.stopHeartbeat();
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
        this.isConnected = false;
    }

    send(type, data = {}) {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            console.error('WebSocket未连接');
            return false;
        }

        try {
            const message = {
                type,
                data: JSON.stringify(data)
            };
            this.ws.send(JSON.stringify(message));
            return true;
        } catch (error) {
            console.error('发送WebSocket消息失败:', error);
            return false;
        }
    }

    handleMessage(message) {
        console.log('收到WebSocket消息:', message);
        this.emit(message.type, message);
    }

    // 事件监听
    on(event, callback) {
        if (!this.listeners[event]) {
            this.listeners[event] = [];
        }
        this.listeners[event].push(callback);
    }

    off(event, callback) {
        if (this.listeners[event]) {
            this.listeners[event] = this.listeners[event].filter(cb => cb !== callback);
        }
    }

    emit(event, data) {
        if (this.listeners[event]) {
            this.listeners[event].forEach(callback => callback(data));
        }
    }

    // 心跳
    startHeartbeat() {
        this.heartbeatTimer = setInterval(() => {
            if (this.isConnected) {
                this.send('ping', {});
            }
        }, 30000); // 30秒
    }

    stopHeartbeat() {
        if (this.heartbeatTimer) {
            clearInterval(this.heartbeatTimer);
            this.heartbeatTimer = null;
        }
    }

    // 加入房间
    joinRoom(roomId) {
        return this.send('join_room', { room_id: roomId });
    }

    // 离开房间
    leaveRoom() {
        return this.send('leave_room', {});
    }

    // 重连请求
    reconnect(roomId) {
        return this.send('reconnect', { room_id: roomId });
    }
}

// 创建全局WebSocket实例
const wsManager = new WebSocketManager();
window.wsManager = wsManager;


