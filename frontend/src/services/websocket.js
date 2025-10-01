class WebSocketService {
  constructor() {
    this.client = null;
    this.listeners = new Map();
    this.connecting = false;
    this.autoReconnectTimer = null;
    this.heartbeatTimer = null;
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = -1; // 无限重连
    this.reconnectInterval = 1000; // 初始重连间隔1秒
    this.maxReconnectInterval = 30000; // 最大重连间隔30秒
    this.heartbeatInterval = 25000; // 心跳间隔25秒
    this.connectionLost = false;
  }

  async connect() {
    if (this.connecting || this.isConnected()) return;
    
    this.connecting = true;
    try {
      const socket = new WebSocket(`ws://localhost:34444/ws?id=dreamcreator`);
      
      await new Promise((resolve, reject) => {
        const timeout = setTimeout(() => {
          reject(new Error('Connection timeout'));
        }, 10000);

        socket.onopen = () => {
          clearTimeout(timeout);
          this.reconnectAttempts = 0;
          this.connectionLost = false;
          console.log('WebSocket connected');
          resolve();
        };
        
        socket.onerror = (error) => {
          clearTimeout(timeout);
          console.error('WebSocket connection error:', error);
          reject(error);
        };
        
        socket.onmessage = (msg) => this.handleMessage(msg);
        
        socket.onclose = (event) => {
          clearTimeout(timeout);
          console.log('WebSocket connection closed:', event.code, event.reason);
          this.client = null;
          this.connectionLost = true;
          this.stopHeartbeat();
          
          // 自动重连（除非是正常关闭）
          if (event.code !== 1000) {
            this.scheduleReconnect();
          }
        };
      });
      
      this.client = socket;
      this.startHeartbeat();
      
    } finally {
      this.connecting = false;
    }
  }

  // 启动心跳
  startHeartbeat() {
    this.stopHeartbeat();
    this.heartbeatTimer = setInterval(() => {
      if (this.isConnected()) {
        try {
          this.client.send(JSON.stringify({
            namespace: 'system',
            event: 'pong',
            data: { timestamp: Date.now() }
          }));
        } catch (error) {
          console.error('Failed to send heartbeat:', error);
        }
      }
    }, this.heartbeatInterval);
  }

  // 停止心跳
  stopHeartbeat() {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }

  // 计划重连
  scheduleReconnect() {
    if (this.autoReconnectTimer) return;
    
    const delay = Math.min(
      this.reconnectInterval * Math.pow(2, this.reconnectAttempts),
      this.maxReconnectInterval
    );
    
    console.log(`Scheduling reconnect in ${delay}ms (attempt ${this.reconnectAttempts + 1})`);
    
    this.autoReconnectTimer = setTimeout(async () => {
      this.autoReconnectTimer = null;
      
      if (this.maxReconnectAttempts === -1 || this.reconnectAttempts < this.maxReconnectAttempts) {
        this.reconnectAttempts++;
        try {
          await this.connect();
        } catch (error) {
          console.error('Reconnect failed:', error);
          this.scheduleReconnect();
        }
      }
    }, delay);
  }

  // 确保连接，带重试
  async ensureConnected(retries = 3) {
    if (this.isConnected()) return true;
    
    for (let i = 0; i <= retries; i++) {
      try {
        await this.connect();
        if (this.isConnected()) return true;
      } catch (error) {
        if (i < retries) {
          await new Promise(resolve => setTimeout(resolve, 1000 * (i + 1)));
        }
      }
    }
    return false;
  }

  // 启动自动重连（Wails应用启动时调用）
  startAutoReconnect() {
    // 立即尝试连接
    this.connect().catch(error => {
      console.error('Initial connection failed:', error);
      this.scheduleReconnect();
    });
  }

  // 停止自动重连
  stopAutoReconnect() {
    if (this.autoReconnectTimer) {
      clearTimeout(this.autoReconnectTimer);
      this.autoReconnectTimer = null;
    }
  }

  async send(namespace, event, data) {
    if (!(await this.ensureConnected())) {
      throw new Error('WebSocket 连接失败');
    }
    
    this.client.send(JSON.stringify({ namespace, event, data }));
  }

  addListener(namespace, callback) {
    if (!this.listeners.has(namespace)) {
      this.listeners.set(namespace, []);
    }
    this.listeners.get(namespace).push(callback);
  }

  removeListener(namespace, callback) {
    const callbacks = this.listeners.get(namespace);
    if (callbacks) {
      const index = callbacks.indexOf(callback);
      if (index !== -1) callbacks.splice(index, 1);
    }
  }

  handleMessage(message) {
    try {
      const data = JSON.parse(message.data);
      
      // 处理心跳消息
      if (data.namespace === 'system' && data.event === 'heartbeat') {
        // 服务器心跳，更新连接状态
        return;
      }
      
      const callbacks = this.listeners.get(data.namespace) || [];
      callbacks.forEach(cb => cb(data));
    } catch (e) {
      console.error('Parse message error:', e);
    }
  }

  disconnect() {
    this.stopAutoReconnect();
    this.stopHeartbeat();
    if (this.client) {
      this.client.close(1000, 'Normal closure');
      this.client = null;
    }
  }

  isConnected() {
    return this.client?.readyState === WebSocket.OPEN;
  }

  // 获取连接状态
  getConnectionStatus() {
    if (!this.client) return 'disconnected';
    
    switch (this.client.readyState) {
      case WebSocket.CONNECTING: return 'connecting';
      case WebSocket.OPEN: return 'connected';
      case WebSocket.CLOSING: return 'closing';
      case WebSocket.CLOSED: return 'disconnected';
      default: return 'unknown';
    }
  }
}

export default new WebSocketService();
