/**
 * Chat Service
 * 
 * Singleton service that manages the IM Client connection and provides
 * a centralized interface for chat functionality.
 */

import { IMClient } from '@/sdk/im-client';
import type { IMClientConfig, IncomingMessage, AckMessage, ReadReceiptMessage } from '@/sdk/im-client';

class ChatService {
  private client: IMClient | null = null;
  private config: IMClientConfig | null = null;

  /**
   * Initialize the chat service with configuration
   */
  initialize(token: string): void {
    if (this.client) {
      console.warn('ChatService already initialized');
      return;
    }

    // Get gateway URL from environment variables
    const gatewayUrl = import.meta.env.VITE_IM_GATEWAY_WS_URL || 'ws://localhost:8080/ws';

    this.config = {
      gatewayUrl,
      token,
      heartbeatInterval: 30000,
      connectionTimeout: 10000,
      reconnect: {
        enabled: true,
        maxAttempts: 5,
        initialDelay: 1000,
        maxDelay: 30000,
        backoffMultiplier: 2,
      },
      deduplication: {
        enabled: true,
        storageType: import.meta.env.PROD ? 'indexeddb' : 'memory',
        ttl: 7 * 24 * 60 * 60 * 1000, // 7 days
      },
      debug: import.meta.env.DEV,
    };

    this.client = new IMClient(this.config);
  }

  /**
   * Get the IM Client instance
   */
  getClient(): IMClient {
    if (!this.client) {
      throw new Error('ChatService not initialized. Call initialize() first.');
    }
    return this.client;
  }

  /**
   * Check if service is initialized
   */
  isInitialized(): boolean {
    return this.client !== null;
  }

  /**
   * Connect to the IM Gateway
   */
  async connect(): Promise<void> {
    const client = this.getClient();
    await client.connect();
  }

  /**
   * Disconnect from the IM Gateway
   */
  disconnect(): void {
    if (this.client) {
      this.client.disconnect();
    }
  }

  /**
   * Send a message to a user
   */
  async sendPrivateMessage(recipientId: string, content: string): Promise<string> {
    const client = this.getClient();
    return client.sendMessage(recipientId, content, 'user');
  }

  /**
   * Send a message to a group
   */
  async sendGroupMessage(groupId: string, content: string): Promise<string> {
    const client = this.getClient();
    return client.sendMessage(groupId, content, 'group');
  }

  /**
   * Send a read receipt
   */
  sendReadReceipt(msgId: string): void {
    const client = this.getClient();
    client.sendReadReceipt(msgId);
  }

  /**
   * Get current connection state
   */
  getConnectionState(): string {
    if (!this.client) {
return 'disconnected';
}
    return this.client.getState();
  }

  /**
   * Get authenticated user ID
   */
  getUserId(): string | undefined {
    return this.client?.getUserId();
  }

  /**
   * Get device ID
   */
  getDeviceId(): string | undefined {
    return this.client?.getDeviceId();
  }

  /**
   * Register event handlers
   */
  on(event: string, handler: (...args: any[]) => void): void {
    const client = this.getClient();
    (client as any).on(event, handler);
  }

  /**
   * Unregister event handlers
   */
  off(event: string): void {
    const client = this.getClient();
    (client as any).off(event);
  }

  /**
   * Clear deduplication storage
   */
  async clearDeduplication(): Promise<void> {
    const client = this.getClient();
    await client.clearDeduplication();
  }

  /**
   * Destroy the service and clean up resources
   */
  destroy(): void {
    if (this.client) {
      this.client.destroy();
      this.client = null;
      this.config = null;
    }
  }

  /**
   * Get offline messages
   */
  async getOfflineMessages(cursor = '', limit = 50): Promise<{
    messages: any[];
    next_cursor: string;
    has_more: boolean;
    total_count: number;
  }> {
    const gatewayUrl = import.meta.env.VITE_IM_GATEWAY_WS_URL || 'ws://localhost:8080/ws';
    const baseUrl = gatewayUrl.replace(/^ws/, 'http').replace(/\/ws$/, '');
    
    const url = new URL(`${baseUrl}/api/v1/offline`);
    if (cursor) {
url.searchParams.set('cursor', cursor);
}
    url.searchParams.set('limit', limit.toString());

    const response = await fetch(url.toString(), {
      headers: {
        'Authorization': `Bearer ${this.config?.token || ''}`,
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch offline messages: ${response.statusText}`);
    }

    return response.json();
  }

  /**
   * Get offline message count
   */
  async getOfflineMessageCount(): Promise<{
    count: number;
    oldest_timestamp: number;
    newest_timestamp: number;
  }> {
    const gatewayUrl = import.meta.env.VITE_IM_GATEWAY_WS_URL || 'ws://localhost:8080/ws';
    const baseUrl = gatewayUrl.replace(/^ws/, 'http').replace(/\/ws$/, '');
    
    const url = `${baseUrl}/api/v1/offline/count`;

    const response = await fetch(url, {
      headers: {
        'Authorization': `Bearer ${this.config?.token || ''}`,
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch offline message count: ${response.statusText}`);
    }

    return response.json();
  }
}

// Export singleton instance
export const chatService = new ChatService();

// Export types for convenience
export type { IncomingMessage, AckMessage, ReadReceiptMessage };
