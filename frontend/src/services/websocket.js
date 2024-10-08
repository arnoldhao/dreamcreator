class WebSocketService {
  constructor() {
    this.sockets = new Map();
    this.listeners = new Map();
  }

  connect(clientId) {
    return new Promise((resolve, reject) => {
      if (this.sockets.has(clientId)) {
        resolve();
        return;
      }

      const socket = new WebSocket(`ws://localhost:34444/ws?id=${clientId}`);

      socket.onopen = () => {
        this.sockets.set(clientId, socket);
        resolve();
      };

      socket.onerror = (error) => {
        console.error(`WebSocket error for clientId ${clientId}:`, error);
        reject(error);
      };

      socket.onmessage = (event) => {
        let data;
        try {
          data = JSON.parse(event.data);
        } catch (e) {
          console.error('Parse message error:', e);
          return;
        }
        this.notifyListeners(clientId, data);
      };

      socket.onclose = () => {
        this.sockets.delete(clientId);
      };
    });
  }

  send(clientId, message) {
    const socket = this.sockets.get(clientId);
    if (socket && socket.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify(message));
    } else {
      console.error(`WebSocket is not connected for clientId: ${clientId}`);
    }
  }

  addListener(clientId, event, callback) {
    const key = `${clientId}:${event}`;
    if (!this.listeners.has(key)) {
      this.listeners.set(key, []);
    }
    this.listeners.get(key).push(callback);
  }

  removeListener(clientId, event, callback) {
    const key = `${clientId}:${event}`;
    if (this.listeners.has(key)) {
      const callbacks = this.listeners.get(key);
      const index = callbacks.indexOf(callback);
      if (index !== -1) {
        callbacks.splice(index, 1);
      }
    }
  }

  notifyListeners(clientId, data) {
    const key = `${clientId}:${data.event}`;
    if (this.listeners.has(key)) {
      this.listeners.get(key).forEach(callback => callback(data.payload));
    }
  }

  disconnect(clientId) {
    const socket = this.sockets.get(clientId);
    if (socket) {
      socket.close();
      this.sockets.delete(clientId);
    }
  }
}

export default new WebSocketService();
