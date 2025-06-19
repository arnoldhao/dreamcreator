import { types } from 'wailsjs/go/models';

class WebSocketService {
  constructor() {
    this.client = null;
    this.listeners = new Map();
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 5;
    this.reconnectDelay = 1000;
  }

  connect() {
    return new Promise((resolve, reject) => {
      // 如果已经连接且状态正常，直接返回
      if (this.client && this.client.readyState === WebSocket.OPEN) {
        resolve();
        return;
      }

      const socket = new WebSocket(`ws://localhost:34444/ws?id=canme`);

      socket.onopen = () => {
        console.log(`WebSocket connected`);
        this.client = socket;
        this.reconnectAttempts = 0; // 重置重连计数
        resolve();
      };

      socket.onerror = (error) => {
        console.error(`WebSocket error:`, error);
        reject(error);
      };

      socket.onmessage = (message) => {
        let data;
        try {
          const jsonData = JSON.parse(message.data);
          data = new types.WSResponse(jsonData);
        } catch (e) {
          console.error('Parse message error:', e);
          return;
        }
        this.notifyListeners(data);
      };

      socket.onclose = (event) => {
        console.log('WebSocket connection closed:', event.code, event.reason);
        this.client = null; // 修复：正确清理连接状态
        
        // 自动重连逻辑
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
          setTimeout(() => {
            this.reconnectAttempts++;
            console.log(`Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts})`);
            this.connect().catch(console.error);
          }, this.reconnectDelay * Math.pow(2, this.reconnectAttempts));
        }
      };
    });
  }

  send(namespace, event, data) {
    const socket = this.client;
    if (socket && socket.readyState === WebSocket.OPEN) {
      const request = new types.WSRequest({
        namespace: namespace,
        event: event,
        data: data
      });
      socket.send(JSON.stringify(request));
    } else {
      console.error(`WebSocket is not connected, attempting to reconnect...`);
      // 尝试重新连接
      this.connect().then(() => {
        if (this.client && this.client.readyState === WebSocket.OPEN) {
          const request = new types.WSRequest({
            namespace: namespace,
            event: event,
            data: data
          });
          this.client.send(JSON.stringify(request));
        }
      }).catch(console.error);
    }
  }

  addListener(namespace, callback) {
    const key = `${namespace}`;
    if (!this.listeners.has(key)) {
      this.listeners.set(key, []);
    }
    this.listeners.get(key).push(callback);
  }

  removeListener(namespace, callback) {
    const key = `${namespace}`;
    if (this.listeners.has(key)) {
      const callbacks = this.listeners.get(key);
      const index = callbacks.indexOf(callback);
      if (index !== -1) {
        callbacks.splice(index, 1);
      }
    }
  }

  notifyListeners(data) {
    const key = `${data.namespace}`;
    if (this.listeners.has(key)) {
      this.listeners.get(key).forEach(callback => callback(data));
    } else {
      console.log(`No listeners for ${key}`);
    }
  }

  disconnect() {
    const socket = this.client;
    if (socket) {
      socket.close();
      this.client = null;
    }
  }
}

export default new WebSocketService();
