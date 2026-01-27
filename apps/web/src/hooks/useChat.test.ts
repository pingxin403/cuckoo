/**
 * useChat Hook Tests
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useChat } from './useChat';
import { chatService } from '@/services/chatService';

// Mock chatService
vi.mock('@/services/chatService', () => ({
  chatService: {
    initialize: vi.fn(),
    isInitialized: vi.fn(() => true),
    connect: vi.fn(() => Promise.resolve()),
    disconnect: vi.fn(),
    sendPrivateMessage: vi.fn((_recipientId, _content) => 
      Promise.resolve(`msg-${Date.now()}`),
    ),
    sendGroupMessage: vi.fn((_groupId, _content) => 
      Promise.resolve(`msg-${Date.now()}`),
    ),
    sendReadReceipt: vi.fn(),
    getConnectionState: vi.fn(() => 'disconnected'),
    getUserId: vi.fn(() => 'user123'),
    getDeviceId: vi.fn(() => 'device456'),
    on: vi.fn(),
    off: vi.fn(),
    getOfflineMessages: vi.fn(() => Promise.resolve({
      messages: [],
      next_cursor: '',
      has_more: false,
      total_count: 0,
    })),
    getOfflineMessageCount: vi.fn(() => Promise.resolve({
      count: 0,
      oldest_timestamp: 0,
      newest_timestamp: 0,
    })),
  },
}));

describe('useChat', () => {
  const mockToken = 'test-token';
  const mockUserId = 'user123';

  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  describe('Initialization', () => {
    it('should initialize chat service with token', () => {
      // Mock isInitialized to return false so initialize is called
      vi.mocked(chatService.isInitialized).mockReturnValue(false);
      
      renderHook(() => useChat({ token: mockToken }));

      expect(chatService.initialize).toHaveBeenCalledWith(mockToken);
    });

    it('should auto-connect by default', async () => {
      renderHook(() => useChat({ token: mockToken }));

      await waitFor(() => {
        expect(chatService.connect).toHaveBeenCalled();
      });
    });

    it('should not auto-connect when disabled', () => {
      renderHook(() => useChat({ token: mockToken, autoConnect: false }));

      expect(chatService.connect).not.toHaveBeenCalled();
    });

    it('should set error when no token provided', () => {
      const { result } = renderHook(() => useChat({ token: '' }));

      // Should not initialize without token
      expect(chatService.initialize).not.toHaveBeenCalled();
      expect(result.current.error).toBeTruthy();
      expect(result.current.error?.message).toContain('token');
    });
  });

  describe('Connection State', () => {
    it('should expose connection state', () => {
      const { result } = renderHook(() => useChat({ token: mockToken }));

      expect(result.current.connectionState).toBe('disconnected');
      expect(result.current.isConnected).toBe(false);
      expect(result.current.isConnecting).toBe(false);
    });

    it('should expose user ID and device ID', () => {
      const { result } = renderHook(() => useChat({ token: mockToken }));

      // Simulate connection
      act(() => {
        const onConnectedHandler = (chatService.on as any).mock.calls.find(
          (call: any[]) => call[0] === 'onConnected',
        )?.[1];
        
        if (onConnectedHandler) {
          onConnectedHandler('user123', 'device456');
        }
      });

      expect(result.current.userId).toBe('user123');
      expect(result.current.deviceId).toBe('device456');
    });
  });

  describe('Message Sending', () => {
    it('should send private message', async () => {
      const { result } = renderHook(() => useChat({ 
        token: mockToken,
        userId: mockUserId,
      }));

      // Simulate connection
      act(() => {
        const onConnectedHandler = (chatService.on as any).mock.calls.find(
          (call: any[]) => call[0] === 'onConnected',
        )?.[1];
        
        if (onConnectedHandler) {
          onConnectedHandler(mockUserId, 'device456');
        }
      });

      let msgId: string = '';
      await act(async () => {
        msgId = await result.current.sendPrivateMessage('user456', 'Hello');
      });

      expect(chatService.sendPrivateMessage).toHaveBeenCalledWith('user456', 'Hello');
      expect(msgId).toBeTruthy();
      
      // Should add optimistic message
      expect(result.current.messages.length).toBeGreaterThan(0);
      expect(result.current.messages[0].content).toBe('Hello');
      expect(result.current.messages[0].status).toBe('pending');
      expect(result.current.messages[0].isOwn).toBe(true);
    });

    it('should send group message', async () => {
      const { result } = renderHook(() => useChat({ 
        token: mockToken,
        userId: mockUserId,
      }));

      // Simulate connection
      act(() => {
        const onConnectedHandler = (chatService.on as any).mock.calls.find(
          (call: any[]) => call[0] === 'onConnected',
        )?.[1];
        
        if (onConnectedHandler) {
          onConnectedHandler(mockUserId, 'device456');
        }
      });

      let msgId: string = '';
      await act(async () => {
        msgId = await result.current.sendGroupMessage('group789', 'Hello group');
      });

      expect(chatService.sendGroupMessage).toHaveBeenCalledWith('group789', 'Hello group');
      expect(msgId).toBeTruthy();
      
      // Should add optimistic message
      expect(result.current.messages.length).toBeGreaterThan(0);
      expect(result.current.messages[0].content).toBe('Hello group');
      expect(result.current.messages[0].recipient_type).toBe('group');
    });
  });

  describe('Message Receiving', () => {
    it('should receive and display incoming messages', () => {
      const { result } = renderHook(() => useChat({ 
        token: mockToken,
        userId: mockUserId,
      }));

      const mockMessage = {
        type: 'message' as const,
        msg_id: 'msg123',
        sender_id: 'user456',
        recipient_id: mockUserId,
        recipient_type: 'user' as const,
        content: 'Hello from user456',
        sequence_number: 1,
        timestamp: Date.now(),
      };

      // Simulate incoming message
      act(() => {
        const onMessageHandler = (chatService.on as any).mock.calls.find(
          (call: any[]) => call[0] === 'onMessage',
        )?.[1];
        
        if (onMessageHandler) {
          onMessageHandler(mockMessage);
        }
      });

      expect(result.current.messages.length).toBe(1);
      expect(result.current.messages[0].content).toBe('Hello from user456');
      expect(result.current.messages[0].status).toBe('received');
      expect(result.current.messages[0].isOwn).toBe(false);
    });

    it('should auto-send read receipt for incoming messages', () => {
      renderHook(() => useChat({ 
        token: mockToken,
        userId: mockUserId,
        autoReadReceipt: true,
      }));

      const mockMessage = {
        type: 'message' as const,
        msg_id: 'msg123',
        sender_id: 'user456',
        recipient_id: mockUserId,
        recipient_type: 'user' as const,
        content: 'Hello',
        sequence_number: 1,
        timestamp: Date.now(),
      };

      // Simulate incoming message
      act(() => {
        const onMessageHandler = (chatService.on as any).mock.calls.find(
          (call: any[]) => call[0] === 'onMessage',
        )?.[1];
        
        if (onMessageHandler) {
          onMessageHandler(mockMessage);
        }
      });

      expect(chatService.sendReadReceipt).toHaveBeenCalledWith('msg123');
    });

    it('should not auto-send read receipt for own messages', () => {
      renderHook(() => useChat({ 
        token: mockToken,
        userId: mockUserId,
        autoReadReceipt: true,
      }));

      const mockMessage = {
        type: 'message' as const,
        msg_id: 'msg123',
        sender_id: mockUserId, // Own message
        recipient_id: 'user456',
        recipient_type: 'user' as const,
        content: 'Hello',
        sequence_number: 1,
        timestamp: Date.now(),
      };

      // Simulate incoming message
      act(() => {
        const onMessageHandler = (chatService.on as any).mock.calls.find(
          (call: any[]) => call[0] === 'onMessage',
        )?.[1];
        
        if (onMessageHandler) {
          onMessageHandler(mockMessage);
        }
      });

      expect(chatService.sendReadReceipt).not.toHaveBeenCalled();
    });
  });

  describe('ACK Handling', () => {
    it('should update message status on ACK', async () => {
      const { result } = renderHook(() => useChat({ 
        token: mockToken,
        userId: mockUserId,
      }));

      // Simulate connection
      act(() => {
        const onConnectedHandler = (chatService.on as any).mock.calls.find(
          (call: any[]) => call[0] === 'onConnected',
        )?.[1];
        
        if (onConnectedHandler) {
          onConnectedHandler(mockUserId, 'device456');
        }
      });

      // Send message
      let msgId: string = '';
      await act(async () => {
        msgId = await result.current.sendPrivateMessage('user456', 'Hello');
      });

      // Simulate ACK
      act(() => {
        const onAckHandler = (chatService.on as any).mock.calls.find(
          (call: any[]) => call[0] === 'onAck',
        )?.[1];
        
        if (onAckHandler) {
          onAckHandler({
            type: 'ack',
            msg_id: msgId,
            status: 'delivered',
            timestamp: Date.now(),
          });
        }
      });

      const message = result.current.messages.find(m => m.msg_id === msgId);
      expect(message?.status).toBe('delivered');
    });
  });

  describe('Read Receipt Handling', () => {
    it('should update message status on read receipt', async () => {
      const { result } = renderHook(() => useChat({ 
        token: mockToken,
        userId: mockUserId,
      }));

      // Simulate connection
      act(() => {
        const onConnectedHandler = (chatService.on as any).mock.calls.find(
          (call: any[]) => call[0] === 'onConnected',
        )?.[1];
        
        if (onConnectedHandler) {
          onConnectedHandler(mockUserId, 'device456');
        }
      });

      // Send message
      let msgId: string = '';
      await act(async () => {
        msgId = await result.current.sendPrivateMessage('user456', 'Hello');
      });

      // Simulate read receipt
      act(() => {
        const onReadReceiptHandler = (chatService.on as any).mock.calls.find(
          (call: any[]) => call[0] === 'onReadReceipt',
        )?.[1];
        
        if (onReadReceiptHandler) {
          onReadReceiptHandler({
            type: 'read_receipt',
            msg_id: msgId,
            reader_id: 'user456',
            read_at: Date.now(),
          });
        }
      });

      const message = result.current.messages.find(m => m.msg_id === msgId);
      expect(message?.status).toBe('read');
    });
  });

  describe('Utility Functions', () => {
    it('should clear messages', async () => {
      const { result } = renderHook(() => useChat({ 
        token: mockToken,
        userId: mockUserId,
      }));

      // Simulate connection and send message
      act(() => {
        const onConnectedHandler = (chatService.on as any).mock.calls.find(
          (call: any[]) => call[0] === 'onConnected',
        )?.[1];
        
        if (onConnectedHandler) {
          onConnectedHandler(mockUserId, 'device456');
        }
      });

      await act(async () => {
        await result.current.sendPrivateMessage('user456', 'Hello');
      });

      expect(result.current.messages.length).toBeGreaterThan(0);

      // Clear messages
      act(() => {
        result.current.clearMessages();
      });

      expect(result.current.messages.length).toBe(0);
    });

    it('should manually connect', async () => {
      const { result } = renderHook(() => useChat({ 
        token: mockToken,
        autoConnect: false,
      }));

      expect(chatService.connect).not.toHaveBeenCalled();

      await act(async () => {
        await result.current.connect();
      });

      expect(chatService.connect).toHaveBeenCalled();
    });

    it('should manually disconnect', () => {
      const { result } = renderHook(() => useChat({ token: mockToken }));

      act(() => {
        result.current.disconnect();
      });

      expect(chatService.disconnect).toHaveBeenCalled();
    });
  });

  describe('Reconnection', () => {
    it('should track reconnection attempts', () => {
      const { result } = renderHook(() => useChat({ token: mockToken }));

      expect(result.current.reconnectInfo).toBeNull();

      // Simulate reconnection
      act(() => {
        const onReconnectingHandler = (chatService.on as any).mock.calls.find(
          (call: any[]) => call[0] === 'onReconnecting',
        )?.[1];
        
        if (onReconnectingHandler) {
          onReconnectingHandler(2, 5);
        }
      });

      expect(result.current.reconnectInfo).toEqual({ attempt: 2, maxAttempts: 5 });
    });

    it('should clear reconnection info on successful connection', () => {
      const { result } = renderHook(() => useChat({ token: mockToken }));

      // Simulate reconnection
      act(() => {
        const onReconnectingHandler = (chatService.on as any).mock.calls.find(
          (call: any[]) => call[0] === 'onReconnecting',
        )?.[1];
        
        if (onReconnectingHandler) {
          onReconnectingHandler(2, 5);
        }
      });

      expect(result.current.reconnectInfo).toBeTruthy();

      // Simulate successful connection
      act(() => {
        const onConnectedHandler = (chatService.on as any).mock.calls.find(
          (call: any[]) => call[0] === 'onConnected',
        )?.[1];
        
        if (onConnectedHandler) {
          onConnectedHandler(mockUserId, 'device456');
        }
      });

      expect(result.current.reconnectInfo).toBeNull();
    });
  });

  describe('Cleanup', () => {
    it('should disconnect on unmount', () => {
      const { unmount } = renderHook(() => useChat({ token: mockToken }));

      unmount();

      expect(chatService.disconnect).toHaveBeenCalled();
    });
  });
});
