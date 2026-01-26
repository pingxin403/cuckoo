/**
 * IM Client SDK - Main Client Class
 * 
 * WebSocket-based instant messaging client with automatic reconnection,
 * heartbeat handling, and message deduplication.
 */

import type {
  IMClientConfig,
  IMClientEvents,
  ConnectionState,
  ClientState,
  WebSocketMessage,
  SendMessageRequest,
  IncomingMessage,
  AckMessage,
  ReadReceiptMessage,
  RecipientType,
} from './types';
import { createDeduplicationStorage, type DeduplicationStorage } from './deduplication';

export class IMClient {
  private config: Required<IMClientConfig>;
  private ws: WebSocket | null = null;
  private state: ClientState;
  private events: Partial<IMClientEvents> = {};
  private heartbeatTimer: number | null = null;
  private reconnectTimer: number | null = null;
  private deduplicationStorage: DeduplicationStorage;
  private pendingMessages: Map<string, SendMessageRequest> = new Map();
  private cleanupTimer: number | null = null;

  constructor(config: IMClientConfig) {
    // Merge with defaults
    this.config = {
      gatewayUrl: config.gatewayUrl,
      token: config.token,
      heartbeatInterval: config.heartbeatInterval ?? 30000,
      connectionTimeout: config.connectionTimeout ?? 10000,
      reconnect: {
        enabled: config.reconnect?.enabled ?? true,
        maxAttempts: config.reconnect?.maxAttempts ?? 5,
        initialDelay: config.reconnect?.initialDelay ?? 1000,
        maxDelay: config.reconnect?.maxDelay ?? 30000,
        backoffMultiplier: config.reconnect?.backoffMultiplier ?? 2,
      },
      deduplication: {
        enabled: config.deduplication?.enabled ?? true,
        storageType: config.deduplication?.storageType ?? 'memory',
        ttl: config.deduplication?.ttl ?? 7 * 24 * 60 * 60 * 1000, // 7 days
      },
      debug: config.debug ?? false,
    };

    this.state = {
      connectionState: 'disconnected',
      reconnectAttempts: 0,
    };

    // Initialize deduplication storage
    this.deduplicationStorage = createDeduplicationStorage(
      this.config.deduplication.storageType,
      this.config.deduplication.ttl,
    );

    // Start periodic cleanup
    this.startCleanupTimer();
  }

  // ============================================================================
  // Public API
  // ============================================================================

  /**
   * Connect to the IM Gateway
   */
  async connect(): Promise<void> {
    if (this.state.connectionState === 'connected' || this.state.connectionState === 'connecting') {
      this.log('Already connected or connecting');
      return;
    }

    this.updateState('connecting');

    return new Promise((resolve, reject) => {
      try {
        // Create WebSocket connection with Authorization header
        this.ws = new WebSocket(this.config.gatewayUrl);

        // Set up event handlers
        this.ws.onopen = () => {
          this.log('WebSocket connection opened');
          // Note: We don't resolve here, wait for auth_response
        };

        this.ws.onmessage = (event) => {
          this.handleMessage(event.data, resolve, reject);
        };

        this.ws.onerror = (error) => {
          this.log('WebSocket error:', error);
          this.handleError(new Error('WebSocket connection error'));
          reject(new Error('WebSocket connection error'));
        };

        this.ws.onclose = (event) => {
          this.handleClose(event.code, event.reason);
        };

        // Set connection timeout
        const timeout = setTimeout(() => {
          if (this.state.connectionState === 'connecting') {
            this.log('Connection timeout');
            this.ws?.close();
            reject(new Error('Connection timeout'));
          }
        }, this.config.connectionTimeout);

        // Clear timeout on successful connection
        const originalResolve = resolve;
        resolve = (...args) => {
          clearTimeout(timeout);
          originalResolve(...args);
        };

      } catch (error) {
        this.log('Failed to create WebSocket:', error);
        reject(error);
      }
    });
  }

  /**
   * Disconnect from the IM Gateway
   */
  disconnect(): void {
    if (this.state.connectionState === 'disconnected') {
      return;
    }

    this.updateState('disconnecting');
    this.stopHeartbeat();
    this.stopReconnect();

    if (this.ws) {
      this.ws.close(1000, 'Client disconnect');
      this.ws = null;
    }

    this.updateState('disconnected');
  }

  /**
   * Send a message to a user or group
   */
  async sendMessage(
    recipientId: string,
    content: string,
    recipientType: RecipientType = 'user',
  ): Promise<string> {
    if (this.state.connectionState !== 'connected') {
      throw new Error('Not connected to IM Gateway');
    }

    if (!this.ws) {
      throw new Error('WebSocket not initialized');
    }

    // Generate unique message ID
    const msgId = this.generateMessageId();

    // Check for duplicate
    if (this.config.deduplication.enabled) {
      const isDuplicate = await this.deduplicationStorage.has(msgId);
      if (isDuplicate) {
        this.log('Duplicate message detected, skipping:', msgId);
        throw new Error('Duplicate message');
      }
    }

    // Create message
    const message: SendMessageRequest = {
      type: 'send_msg',
      msg_id: msgId,
      recipient_id: recipientId,
      recipient_type: recipientType,
      content,
      timestamp: Date.now(),
    };

    // Store in pending messages
    this.pendingMessages.set(msgId, message);

    // Send message
    this.ws.send(JSON.stringify(message));
    this.log('Message sent:', msgId);

    // Add to deduplication storage
    if (this.config.deduplication.enabled) {
      await this.deduplicationStorage.add(msgId);
    }

    return msgId;
  }

  /**
   * Send read receipt for a message
   */
  sendReadReceipt(msgId: string): void {
    if (this.state.connectionState !== 'connected' || !this.ws) {
      this.log('Cannot send read receipt: not connected');
      return;
    }

    const ack: AckMessage = {
      type: 'ack',
      msg_id: msgId,
      status: 'read',
      timestamp: Date.now(),
    };

    this.ws.send(JSON.stringify(ack));
    this.log('Read receipt sent:', msgId);
  }

  /**
   * Get current connection state
   */
  getState(): ConnectionState {
    return this.state.connectionState;
  }

  /**
   * Get user ID (available after connection)
   */
  getUserId(): string | undefined {
    return this.state.userId;
  }

  /**
   * Get device ID (available after connection)
   */
  getDeviceId(): string | undefined {
    return this.state.deviceId;
  }

  /**
   * Register event handlers
   */
  on<K extends keyof IMClientEvents>(event: K, handler: IMClientEvents[K]): void {
    this.events[event] = handler;
  }

  /**
   * Unregister event handlers
   */
  off<K extends keyof IMClientEvents>(event: K): void {
    delete this.events[event];
  }

  /**
   * Clear deduplication storage
   */
  async clearDeduplication(): Promise<void> {
    await this.deduplicationStorage.clear();
  }

  // ============================================================================
  // Private Methods - Message Handling
  // ============================================================================

  private handleMessage(
    data: string,
    connectResolve?: () => void,
    connectReject?: (error: Error) => void,
  ): void {
    try {
      const message: WebSocketMessage = JSON.parse(data);
      this.log('Received message:', message.type);

      switch (message.type) {
        case 'auth_response':
          this.handleAuthResponse(message, connectResolve, connectReject);
          break;

        case 'message':
          this.handleIncomingMessage(message);
          break;

        case 'ack':
          this.handleAck(message);
          break;

        case 'heartbeat_response':
          this.handleHeartbeatResponse(message);
          break;

        case 'read_receipt':
          this.handleReadReceipt(message);
          break;

        case 'error':
          this.handleErrorMessage(message);
          break;

        default:
          this.log('Unknown message type:', (message as any).type);
      }
    } catch (error) {
      this.log('Failed to parse message:', error);
      this.handleError(new Error('Failed to parse message'));
    }
  }

  private handleAuthResponse(
    message: any,
    connectResolve?: () => void,
    connectReject?: (error: Error) => void,
  ): void {
    if (message.success) {
      this.state.userId = message.user_id;
      this.state.deviceId = message.device_id;
      this.state.reconnectAttempts = 0;
      this.updateState('connected');
      
      this.startHeartbeat();
      
      this.log('Authentication successful:', message.user_id, message.device_id);
      this.events.onConnected?.(message.user_id, message.device_id);
      
      connectResolve?.();
    } else {
      this.log('Authentication failed');
      const error = new Error('Authentication failed');
      this.handleError(error);
      connectReject?.(error);
      this.ws?.close();
    }
  }

  private async handleIncomingMessage(message: IncomingMessage): Promise<void> {
    this.log('Incoming message:', message.msg_id);

    // Check for duplicate
    if (this.config.deduplication.enabled) {
      const isDuplicate = await this.deduplicationStorage.has(message.msg_id);
      if (isDuplicate) {
        this.log('Duplicate message received, skipping:', message.msg_id);
        return;
      }
      await this.deduplicationStorage.add(message.msg_id);
    }

    // Send ACK
    if (this.ws) {
      const ack: AckMessage = {
        type: 'ack',
        msg_id: message.msg_id,
        status: 'received',
        timestamp: Date.now(),
      };
      this.ws.send(JSON.stringify(ack));
    }

    // Notify application
    this.events.onMessage?.(message);
  }

  private handleAck(message: AckMessage): void {
    this.log('ACK received:', message.msg_id, message.status);

    // Remove from pending messages
    this.pendingMessages.delete(message.msg_id);

    // Notify application
    this.events.onAck?.(message);
  }

  private handleHeartbeatResponse(_message: any): void {
    this.state.lastHeartbeat = Date.now();
    this.log('Heartbeat response received');
  }

  private handleReadReceipt(_message: ReadReceiptMessage): void {
    this.log('Read receipt received:', _message.msg_id);
    this.events.onReadReceipt?.(_message);
  }

  private handleErrorMessage(message: any): void {
    this.log('Error message received:', message.code, message.message);
    this.handleError(new Error(`${message.code}: ${message.message}`));
  }

  // ============================================================================
  // Private Methods - Connection Management
  // ============================================================================

  private handleClose(code: number, reason: string): void {
    this.log('WebSocket closed:', code, reason);
    
    this.stopHeartbeat();
    this.ws = null;

    const wasConnected = this.state.connectionState === 'connected';
    this.updateState('disconnected');

    this.events.onDisconnected?.(code, reason);

    // Attempt reconnection if enabled and not a normal closure
    if (wasConnected && this.config.reconnect.enabled && code !== 1000) {
      this.attemptReconnect();
    }
  }

  private handleError(error: Error): void {
    this.log('Error:', error.message);
    this.events.onError?.(error);
  }

  private attemptReconnect(): void {
    if (this.state.reconnectAttempts >= this.config.reconnect.maxAttempts) {
      this.log('Max reconnection attempts reached');
      this.updateState('failed');
      return;
    }

    this.state.reconnectAttempts++;
    this.updateState('reconnecting');

    const delay = Math.min(
      this.config.reconnect.initialDelay * 
        Math.pow(this.config.reconnect.backoffMultiplier, this.state.reconnectAttempts - 1),
      this.config.reconnect.maxDelay,
    );

    this.log(`Reconnecting in ${delay}ms (attempt ${this.state.reconnectAttempts}/${this.config.reconnect.maxAttempts})`);
    
    this.events.onReconnecting?.(
      this.state.reconnectAttempts,
      this.config.reconnect.maxAttempts,
    );

    this.reconnectTimer = window.setTimeout(() => {
      this.connect().catch((error) => {
        this.log('Reconnection failed:', error);
        this.attemptReconnect();
      });
    }, delay);
  }

  private stopReconnect(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.state.reconnectAttempts = 0;
  }

  // ============================================================================
  // Private Methods - Heartbeat
  // ============================================================================

  private startHeartbeat(): void {
    this.stopHeartbeat();
    
    this.heartbeatTimer = window.setInterval(() => {
      this.sendHeartbeat();
    }, this.config.heartbeatInterval);

    // Send initial heartbeat
    this.sendHeartbeat();
  }

  private stopHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }

  private sendHeartbeat(): void {
    if (this.ws && this.state.connectionState === 'connected') {
      const heartbeat = {
        type: 'heartbeat',
        timestamp: Date.now(),
      };
      this.ws.send(JSON.stringify(heartbeat));
      this.log('Heartbeat sent');
    }
  }

  // ============================================================================
  // Private Methods - Utilities
  // ============================================================================

  private updateState(newState: ConnectionState): void {
    if (this.state.connectionState !== newState) {
      this.state.connectionState = newState;
      this.log('State changed:', newState);
      this.events.onStateChange?.(newState);
    }
  }

  private generateMessageId(): string {
    const userId = this.state.userId || 'unknown';
    const timestamp = Date.now();
    const random = Math.random().toString(36).substring(2, 15);
    return `msg_${userId}_${timestamp}_${random}`;
  }

  private startCleanupTimer(): void {
    // Run cleanup every hour
    this.cleanupTimer = window.setInterval(() => {
      this.deduplicationStorage.cleanup().catch((error) => {
        this.log('Cleanup failed:', error);
      });
    }, 60 * 60 * 1000);
  }

  private log(...args: any[]): void {
    if (this.config.debug) {
      console.log('[IMClient]', ...args);
    }
  }

  // ============================================================================
  // Cleanup
  // ============================================================================

  /**
   * Destroy the client and clean up resources
   */
  destroy(): void {
    this.disconnect();
    
    if (this.cleanupTimer) {
      clearInterval(this.cleanupTimer);
      this.cleanupTimer = null;
    }

    this.pendingMessages.clear();
    this.events = {};
  }
}
