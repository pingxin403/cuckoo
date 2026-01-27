/**
 * IM Client SDK - Usage Examples
 * 
 * This file contains practical examples of how to use the IM Client SDK.
 */

import { IMClient } from './IMClient';
import type { IncomingMessage } from './types';

// ============================================================================
// Example 1: Basic Usage
// ============================================================================

export function basicExample() {
  // Create client instance
  const client = new IMClient({
    gatewayUrl: 'wss://gateway.example.com:8080/ws',
    token: 'your-jwt-token-here',
    debug: true,
  });

  // Register event handlers
  client.on('onConnected', (userId, deviceId) => {
    console.log('✅ Connected:', userId, deviceId);
  });

  client.on('onMessage', (message) => {
    console.log('📨 New message:', message.content);
    // Send read receipt
    client.sendReadReceipt(message.msg_id);
  });

  client.on('onError', (error) => {
    console.error('❌ Error:', error);
  });

  // Connect
  client.connect()
    .then(() => console.log('Connection established'))
    .catch((error) => console.error('Connection failed:', error));

  // Send a message after 2 seconds
  setTimeout(async () => {
    try {
      const msgId = await client.sendMessage('user_123', 'Hello!');
      console.log('Message sent:', msgId);
    } catch (error) {
      console.error('Failed to send message:', error);
    }
  }, 2000);

  // Disconnect after 10 seconds
  setTimeout(() => {
    client.disconnect();
    console.log('Disconnected');
  }, 10000);
}

// ============================================================================
// Example 2: React Hook
// ============================================================================

import { useEffect, useState, useCallback } from 'react';

export function useChatClient(gatewayUrl: string, token: string) {
  const [client, setClient] = useState<IMClient | null>(null);
  const [messages, setMessages] = useState<IncomingMessage[]>([]);
  const [connectionState, setConnectionState] = useState<string>('disconnected');
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    // Create client
    const imClient = new IMClient({
      gatewayUrl,
      token,
      debug: import.meta.env.DEV,
      deduplication: {
        enabled: true,
        storageType: 'indexeddb',
        ttl: 7 * 24 * 60 * 60 * 1000,
      },
    });

    // Event handlers
    imClient.on('onConnected', (userId, deviceId) => {
      console.log('Connected:', userId, deviceId);
      setError(null);
    });

    imClient.on('onDisconnected', (code, reason) => {
      console.log('Disconnected:', code, reason);
    });

    imClient.on('onError', (err) => {
      console.error('Client error:', err);
      setError(err);
    });

    imClient.on('onMessage', (message) => {
      setMessages(prev => [...prev, message]);
      // Auto-send read receipt
      imClient.sendReadReceipt(message.msg_id);
    });

    imClient.on('onStateChange', (state) => {
      setConnectionState(state);
    });

    // Connect
    imClient.connect().catch(setError);

    // eslint-disable-next-line react-hooks/set-state-in-effect
    setClient(imClient);

    // Cleanup
    return () => {
      imClient.destroy();
    };
  }, [gatewayUrl, token]);

  const sendMessage = useCallback(async (recipientId: string, content: string, recipientType: 'user' | 'group' = 'user') => {
    if (!client) {
      throw new Error('Client not initialized');
    }
    return client.sendMessage(recipientId, content, recipientType);
  }, [client]);

  const clearMessages = useCallback(() => {
    setMessages([]);
  }, []);

  return {
    client,
    messages,
    connectionState,
    error,
    sendMessage,
    clearMessages,
  };
}

// ============================================================================
// Example 3: Message Queue with Retry
// ============================================================================

export class MessageQueue {
  private client: IMClient;
  private queue: Map<string, { recipientId: string; content: string; retryCount: number }> = new Map();
  private maxRetries = 3;

  constructor(client: IMClient) {
    this.client = client;

    // Listen for ACKs
    client.on('onAck', (ack) => {
      if (ack.status === 'delivered' || ack.status === 'received') {
        this.queue.delete(ack.msg_id);
      }
    });
  }

  async sendMessage(recipientId: string, content: string): Promise<string> {
    const msgId = await this.client.sendMessage(recipientId, content);
    
    // Add to queue
    this.queue.set(msgId, {
      recipientId,
      content,
      retryCount: 0,
    });

    // Set timeout for retry
    setTimeout(() => this.retryMessage(msgId), 5000);

    return msgId;
  }

  private async retryMessage(msgId: string): Promise<void> {
    const message = this.queue.get(msgId);
    if (!message) {
return;
} // Already ACKed

    if (message.retryCount >= this.maxRetries) {
      console.error('Max retries reached for message:', msgId);
      this.queue.delete(msgId);
      return;
    }

    console.log(`Retrying message ${msgId} (attempt ${message.retryCount + 1})`);
    
    try {
      const newMsgId = await this.client.sendMessage(message.recipientId, message.content);
      this.queue.delete(msgId);
      this.queue.set(newMsgId, {
        ...message,
        retryCount: message.retryCount + 1,
      });
      
      setTimeout(() => this.retryMessage(newMsgId), 5000);
    } catch (error) {
      console.error('Retry failed:', error);
      message.retryCount++;
      setTimeout(() => this.retryMessage(msgId), 5000);
    }
  }

  getPendingMessages(): number {
    return this.queue.size;
  }
}

// ============================================================================
// Example 4: Offline Message Sync
// ============================================================================

export class OfflineMessageSync {
  private client: IMClient;
  private lastSyncTimestamp: number = 0;

  constructor(client: IMClient) {
    this.client = client;

    // Sync on reconnection
    client.on('onConnected', () => {
      this.syncOfflineMessages();
    });
  }

  private async syncOfflineMessages(): Promise<void> {
    try {
      // Fetch offline messages from server
      const response = await fetch(`/api/v1/offline?since=${this.lastSyncTimestamp}`, {
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
        },
      });

      if (!response.ok) {
        throw new Error('Failed to fetch offline messages');
      }

      const data = await response.json();
      const messages: IncomingMessage[] = data.messages;

      console.log(`Synced ${messages.length} offline messages`);

      // Process each message
      for (const message of messages) {
        // Trigger onMessage event
        this.client['events'].onMessage?.(message);
      }

      // Update last sync timestamp
      if (messages.length > 0) {
        this.lastSyncTimestamp = Math.max(...messages.map(m => m.timestamp));
      }
    } catch (error) {
      console.error('Failed to sync offline messages:', error);
    }
  }

  private getToken(): string {
    // Get token from storage or state management
    return localStorage.getItem('auth_token') || '';
  }
}

// ============================================================================
// Example 5: Group Chat Manager
// ============================================================================

export class GroupChatManager {
  private client: IMClient;
  private groups: Map<string, { name: string; members: string[] }> = new Map();

  constructor(client: IMClient) {
    this.client = client;
  }

  async sendGroupMessage(groupId: string, content: string): Promise<string> {
    return this.client.sendMessage(groupId, content, 'group');
  }

  async addGroup(groupId: string, name: string, members: string[]): Promise<void> {
    this.groups.set(groupId, { name, members });
  }

  getGroup(groupId: string) {
    return this.groups.get(groupId);
  }

  getAllGroups() {
    return Array.from(this.groups.entries()).map(([id, data]) => ({
      id,
      ...data,
    }));
  }
}

// ============================================================================
// Example 6: Read Receipt Tracker
// ============================================================================

export class ReadReceiptTracker {
  private receipts: Map<string, Set<string>> = new Map(); // msgId -> Set of reader IDs

  constructor(client: IMClient) {
    client.on('onReadReceipt', (receipt) => {
      this.addReceipt(receipt.msg_id, receipt.reader_id);
    });
  }

  private addReceipt(msgId: string, readerId: string): void {
    if (!this.receipts.has(msgId)) {
      this.receipts.set(msgId, new Set());
    }
    this.receipts.get(msgId)!.add(readerId);
  }

  getReadCount(msgId: string): number {
    return this.receipts.get(msgId)?.size || 0;
  }

  getReaders(msgId: string): string[] {
    return Array.from(this.receipts.get(msgId) || []);
  }

  hasRead(msgId: string, userId: string): boolean {
    return this.receipts.get(msgId)?.has(userId) || false;
  }
}

// ============================================================================
// Example 7: Connection Monitor
// ============================================================================

export class ConnectionMonitor {
  private client: IMClient;
  private reconnectAttempts: number = 0;
  private onStatusChange?: (status: string) => void;

  constructor(client: IMClient, onStatusChange?: (status: string) => void) {
    this.client = client;
    this.onStatusChange = onStatusChange;

    // Use client to set up event listeners
    client.on('onStateChange', (state) => {
      this.handleStateChange(state);
    });

    client.on('onReconnecting', (attempt, maxAttempts) => {
      this.reconnectAttempts = attempt;
      this.onStatusChange?.(`Reconnecting... (${attempt}/${maxAttempts})`);
    });

    client.on('onConnected', () => {
      this.reconnectAttempts = 0;
      this.onStatusChange?.('Connected');
    });

    client.on('onDisconnected', () => {
      this.onStatusChange?.('Disconnected');
    });

    client.on('onError', (error) => {
      this.onStatusChange?.(`Error: ${error.message}`);
    });
  }

  private handleStateChange(state: string): void {
    console.log('Connection state:', state);
    
    switch (state) {
      case 'connected':
        this.onStatusChange?.('🟢 Connected');
        break;
      case 'connecting':
        this.onStatusChange?.('🟡 Connecting...');
        break;
      case 'reconnecting':
        this.onStatusChange?.('🟡 Reconnecting...');
        break;
      case 'disconnected':
        this.onStatusChange?.('🔴 Disconnected');
        break;
      case 'failed':
        this.onStatusChange?.('🔴 Connection Failed');
        break;
    }
  }

  getReconnectAttempts(): number {
    return this.reconnectAttempts;
  }

  getClient(): IMClient {
    return this.client;
  }
}
