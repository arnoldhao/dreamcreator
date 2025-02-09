import { types } from 'wailsjs/go/models';

class WebSocketService {
  constructor() {
    this.client = null;
    this.listeners = new Map();
  }

  connect() {
    return new Promise((resolve, reject) => {
      if (this.client) {
        resolve();
        return;
      }

      const socket = new WebSocket(`ws://localhost:34444/ws?id=canme`);

      socket.onopen = () => {
        console.log(`WebSocket connected`);
        this.client = socket;
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

      socket.onclose = () => {
        this.sockets.delete(clientId);
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
      console.error(`WebSocket is not connected`);
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
