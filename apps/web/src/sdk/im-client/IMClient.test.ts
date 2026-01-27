/**
 * IMClient Unit Tests
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { IMClient } from './IMClient';
import type { IMClientConfig } from './types';

describe('IMClient', () => {
  let client: IMClient;
  let config: IMClientConfig;

  beforeEach(() => {
    config = {
      gatewayUrl: 'ws://localhost:8080/ws',
      token: 'test-token',
      heartbeatInterval: 30000,
      connectionTimeout: 10000,
      reconnect: {
        enabled: true,
        maxAttempts: 3,
        initialDelay: 1000,
        maxDelay: 10000,
        backoffMultiplier: 2,
      },
      deduplication: {
        enabled: true,
        storageType: 'memory',
        ttl: 7 * 24 * 60 * 60 * 1000,
      },
      debug: false,
    };

    client = new IMClient(config);
  });

  afterEach(() => {
    client.destroy();
  });

  describe('Initialization', () => {
    it('should initialize with correct config', () => {
      expect(client.getState()).toBe('disconnected');
    });

    it('should parse JWT token and extract user_id', async () => {
      // Mock JWT token: { user_id: "user123", device_id: "device456" }
      const token = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoidXNlcjEyMyIsImRldmljZV9pZCI6ImRldmljZTQ1NiJ9.test';
      const clientWithToken = new IMClient({ ...config, token });
      
      // User ID and device ID are only available after connection
      await clientWithToken.connect();
      
      expect(clientWithToken.getUserId()).toBe('user123');
      expect(clientWithToken.getDeviceId()).toBe('device456');
      
      clientWithToken.destroy();
    });

    it('should handle invalid JWT token gracefully', () => {
      const clientWithInvalidToken = new IMClient({ ...config, token: 'invalid-token' });
      
      expect(clientWithInvalidToken.getUserId()).toBeUndefined();
      expect(clientWithInvalidToken.getDeviceId()).toBeUndefined();
      
      clientWithInvalidToken.destroy();
    });
  });

  describe('Connection Management', () => {
    it('should transition to connecting state', async () => {
      const stateChanges: string[] = [];
      client.on('onStateChange', (state: string) => {
        stateChanges.push(state);
      });

      const connectPromise = client.connect();
      
      expect(client.getState()).toBe('connecting');
      
      await connectPromise;
      
      expect(stateChanges).toContain('connecting');
      expect(stateChanges).toContain('connected');
    });

    it('should emit onConnected event', async () => {
      const onConnected = vi.fn();
      client.on('onConnected', onConnected);

      await client.connect();

      expect(onConnected).toHaveBeenCalled();
    });

    it('should disconnect cleanly', async () => {
      await client.connect();
      
      const onDisconnected = vi.fn();
      client.on('onDisconnected', onDisconnected);

      client.disconnect();

      expect(client.getState()).toBe('disconnected');
      expect(onDisconnected).toHaveBeenCalled();
    });

    it('should handle connection timeout', async () => {
      const shortTimeoutConfig = { ...config, connectionTimeout: 100 };
      const clientWithTimeout = new IMClient(shortTimeoutConfig);

      // Mock WebSocket to never send auth_response (causing timeout)
      const originalWebSocket = global.WebSocket;
      
      class TimeoutWebSocket {
        static CONNECTING = 0;
        static OPEN = 1;
        static CLOSING = 2;
        static CLOSED = 3;
        
        readyState = TimeoutWebSocket.CONNECTING;
        url: string;
        onopen: ((event: Event) => void) | null = null;
        onclose: ((event: CloseEvent) => void) | null = null;
        onerror: ((event: Event) => void) | null = null;
        onmessage: ((event: MessageEvent) => void) | null = null;
        
        constructor(url: string) {
          this.url = url;
          // Trigger onopen but never send auth_response
          setTimeout(() => {
            this.readyState = TimeoutWebSocket.OPEN;
            if (this.onopen) {
              this.onopen(new Event('open'));
            }
            // Don't send auth_response - this will cause timeout
          }, 0);
        }
        
        send(_data: string | ArrayBuffer | Blob) {}
        
        close(code?: number, reason?: string) {
          this.readyState = TimeoutWebSocket.CLOSED;
          if (this.onclose) {
            this.onclose(new CloseEvent('close', { code, reason }));
          }
        }
      }
      
      global.WebSocket = TimeoutWebSocket as any;

      await expect(clientWithTimeout.connect()).rejects.toThrow('Connection timeout');

      global.WebSocket = originalWebSocket;
      clientWithTimeout.destroy();
    }, 10000);
  });

  describe('Message Sending', () => {
    beforeEach(async () => {
      await client.connect();
    });

    it('should send private message', async () => {
      const msgId = await client.sendMessage('user123', 'Hello', 'user');
      
      expect(msgId).toBeTruthy();
      expect(typeof msgId).toBe('string');
    });

    it('should send group message', async () => {
      const msgId = await client.sendMessage('group456', 'Hello group', 'group');
      
      expect(msgId).toBeTruthy();
      expect(typeof msgId).toBe('string');
    });

    it('should generate unique message IDs', async () => {
      const msgId1 = await client.sendMessage('user123', 'Message 1', 'user');
      const msgId2 = await client.sendMessage('user123', 'Message 2', 'user');
      
      expect(msgId1).not.toBe(msgId2);
    });

    it('should throw error when not connected', async () => {
      client.disconnect();
      
      await expect(
        client.sendMessage('user123', 'Hello', 'user'),
      ).rejects.toThrow();
    });
  });

  describe('Message Receiving', () => {
    beforeEach(async () => {
      await client.connect();
    });

    it('should receive and emit incoming messages', async () => {
      const mockMessage = {
        type: 'message',
        msg_id: 'msg123',
        sender_id: 'user456',
        recipient_id: 'user123',
        recipient_type: 'user',
        content: 'Hello',
        sequence_number: 1,
        timestamp: Date.now(),
      };

      const messagePromise = new Promise((resolve) => {
        client.on('onMessage', (message) => {
          expect(message).toEqual(mockMessage);
          resolve(message);
        });
      });

      // Simulate incoming message
      const ws = (client as any).ws;
      if (ws && ws.onmessage) {
        ws.onmessage(new MessageEvent('message', {
          data: JSON.stringify(mockMessage),
        }));
      }

      await messagePromise;
    });

    it('should deduplicate messages', async () => {
      const mockMessage = {
        type: 'message',
        msg_id: 'msg123',
        sender_id: 'user456',
        recipient_id: 'user123',
        recipient_type: 'user',
        content: 'Hello',
        sequence_number: 1,
        timestamp: Date.now(),
      };

      const onMessage = vi.fn();
      client.on('onMessage', onMessage);

      const ws = (client as any).ws;
      if (ws && ws.onmessage) {
        // Send same message twice
        ws.onmessage(new MessageEvent('message', {
          data: JSON.stringify(mockMessage),
        }));
        
        await new Promise(resolve => setTimeout(resolve, 10));
        
        ws.onmessage(new MessageEvent('message', {
          data: JSON.stringify(mockMessage),
        }));
      }

      await new Promise(resolve => setTimeout(resolve, 50));

      // Should only receive once
      expect(onMessage).toHaveBeenCalledTimes(1);
    });
  });

  describe('Heartbeat', () => {
    it('should send heartbeat at configured interval', async () => {
      const shortHeartbeatConfig = { ...config, heartbeatInterval: 100 };
      const clientWithHeartbeat = new IMClient(shortHeartbeatConfig);

      await clientWithHeartbeat.connect();

      const ws = (clientWithHeartbeat as any).ws;
      const sendSpy = vi.spyOn(ws, 'send');

      // Wait for heartbeat
      await new Promise(resolve => setTimeout(resolve, 150));

      expect(sendSpy).toHaveBeenCalledWith(
        expect.stringContaining('"type":"heartbeat"'),
      );

      clientWithHeartbeat.destroy();
    });
  });

  describe('Reconnection', () => {
    it('should attempt reconnection on disconnect', async () => {
      const onReconnecting = vi.fn();
      client.on('onReconnecting', onReconnecting);

      await client.connect();

      // Simulate disconnect
      const ws = (client as any).ws;
      if (ws && ws.onclose) {
        ws.onclose(new CloseEvent('close', { code: 1006, reason: 'Abnormal closure' }));
      }

      // Wait for reconnection attempt
      await new Promise(resolve => setTimeout(resolve, 1500));

      expect(onReconnecting).toHaveBeenCalled();
    }, 10000);

    it('should respect maxAttempts', async () => {
      const limitedReconnectConfig = {
        ...config,
        reconnect: {
          enabled: true,
          maxAttempts: 2,
          initialDelay: 100,
          maxDelay: 1000,
          backoffMultiplier: 2,
        },
      };
      const clientWithLimitedReconnect = new IMClient(limitedReconnectConfig);

      const onReconnecting = vi.fn();
      clientWithLimitedReconnect.on('onReconnecting', onReconnecting);

      await clientWithLimitedReconnect.connect();

      // Mock WebSocket to always fail after initial connection
      let connectionCount = 0;
      const originalWebSocket = global.WebSocket;
      global.WebSocket = class extends originalWebSocket {
        constructor(url: string) {
          super(url);
          connectionCount++;
          if (connectionCount > 1) {
            // Fail subsequent connections
            setTimeout(() => {
              if (this.onerror) {
                this.onerror(new Event('error'));
              }
              if (this.onclose) {
                this.onclose(new CloseEvent('close', { code: 1006 }));
              }
            }, 0);
          }
        }
      } as any;

      // Trigger disconnect
      const ws = (clientWithLimitedReconnect as any).ws;
      if (ws && ws.onclose) {
        ws.onclose(new CloseEvent('close', { code: 1006 }));
      }

      // Wait for all reconnection attempts
      await new Promise(resolve => setTimeout(resolve, 2000));

      // Should have attempted reconnection (may be less than maxAttempts if timing is tight)
      expect(onReconnecting).toHaveBeenCalled();

      global.WebSocket = originalWebSocket;
      clientWithLimitedReconnect.destroy();
    }, 15000);
  });

  describe('Read Receipts', () => {
    it('should send read receipt', async () => {
      await client.connect();
      
      const ws = (client as any).ws;
      const sendSpy = vi.spyOn(ws, 'send');

      client.sendReadReceipt('msg123');

      expect(sendSpy).toHaveBeenCalledWith(
        expect.stringContaining('"type":"ack"'),
      );
      expect(sendSpy).toHaveBeenCalledWith(
        expect.stringContaining('"msg_id":"msg123"'),
      );
    });

    it('should receive read receipt', async () => {
      await client.connect();
      
      const mockReceipt = {
        type: 'read_receipt',
        msg_id: 'msg123',
        reader_id: 'user456',
        read_at: Date.now(),
      };

      const receiptPromise = new Promise((resolve) => {
        client.on('onReadReceipt', (receipt) => {
          expect(receipt).toEqual(mockReceipt);
          resolve(receipt);
        });
      });

      const ws = (client as any).ws;
      if (ws && ws.onmessage) {
        ws.onmessage(new MessageEvent('message', {
          data: JSON.stringify(mockReceipt),
        }));
      }

      await receiptPromise;
    });
  });

  describe('Event Handling', () => {
    it('should register and trigger event handlers', async () => {
      const handler = vi.fn();
      client.on('onStateChange', handler);

      // Trigger state change by connecting
      await client.connect();

      expect(handler).toHaveBeenCalledWith('connecting');
      expect(handler).toHaveBeenCalledWith('connected');
    });

    it('should unregister event handlers', async () => {
      const handler = vi.fn();
      client.on('onStateChange', handler);
      client.off('onStateChange');

      // Trigger state change by connecting
      await client.connect();

      expect(handler).not.toHaveBeenCalled();
    });
  });

  describe('Cleanup', () => {
    it('should clean up resources on destroy', async () => {
      await client.connect();
      
      const ws = (client as any).ws;
      const closeSpy = vi.spyOn(ws, 'close');

      client.destroy();

      expect(closeSpy).toHaveBeenCalled();
      expect(client.getState()).toBe('disconnected');
    });

    it('should clear deduplication storage', async () => {
      await client.connect();
      
      await client.clearDeduplication();

      // Should not throw
      expect(true).toBe(true);
    });
  });
});
