class WebSocketService {
  constructor() {
    this.client = null;
    this.listeners = new Map();
    this.connecting = false;
    this.autoReconnectTimer = null;
  }

  async connect() {
    if (this.connecting || this.isConnected()) return;
    
    this.connecting = true;
    try {
      const socket = new WebSocket(`ws://localhost:34444/ws?id=canme`);
      
      await new Promise((resolve, reject) => {
        socket.onopen = resolve;
        socket.onerror = reject;
        socket.onmessage = (msg) => this.handleMessage(msg);
        socket.onclose = () => { this.client = null; };
      });
      
      this.client = socket;
    } finally {
      this.connecting = false;
    }
  }

  // 确保连接，带简单重试
  async ensureConnected(retries = 2) {
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

  // 自动重连
  startAutoReconnect(interval = 5000) {
    if (this.autoReconnectTimer) return;
    
    this.autoReconnectTimer = setInterval(async () => {
      if (!this.isConnected() && !this.connecting) {
        await this.ensureConnected(1);
      }
    }, interval);
  }

  stopAutoReconnect() {
    if (this.autoReconnectTimer) {
      clearInterval(this.autoReconnectTimer);
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
      const callbacks = this.listeners.get(data.namespace) || [];
      callbacks.forEach(cb => cb(data));
    } catch (e) {
      console.error('Parse message error:', e);
    }
  }

  disconnect() {
    this.stopAutoReconnect();
    this.client?.close();
    this.client = null;
  }

  isConnected() {
    return this.client?.readyState === WebSocket.OPEN;
  }
}

export default new WebSocketService();