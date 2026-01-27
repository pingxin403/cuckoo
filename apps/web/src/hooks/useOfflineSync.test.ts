/**
 * useOfflineSync Hook Tests
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useOfflineSync } from './useOfflineSync';
import { chatService } from '@/services/chatService';

// Mock chatService
vi.mock('@/services/chatService', () => ({
  chatService: {
    isInitialized: vi.fn(() => true),
    getOfflineMessages: vi.fn(),
    getOfflineMessageCount: vi.fn(),
  },
}));

describe('useOfflineSync', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  describe('Initialization', () => {
    it('should initialize with default values', () => {
      const { result } = renderHook(() => useOfflineSync(false));

      expect(result.current.offlineCount).toBe(0);
      expect(result.current.isSyncing).toBe(false);
      expect(result.current.syncError).toBeNull();
      expect(result.current.offlineMessages).toEqual([]);
    });

    it('should not auto-sync when not connected', () => {
      renderHook(() => useOfflineSync(false, { autoSync: true }));

      expect(chatService.getOfflineMessageCount).not.toHaveBeenCalled();
      expect(chatService.getOfflineMessages).not.toHaveBeenCalled();
    });
  });

  describe('Offline Message Count', () => {
    it('should fetch offline message count', async () => {
      (chatService.getOfflineMessageCount as any).mockResolvedValue({
        count: 5,
        oldest_timestamp: Date.now() - 1000,
        newest_timestamp: Date.now(),
      });

      const { result } = renderHook(() => useOfflineSync(false));

      await act(async () => {
        await result.current.fetchOfflineCount();
      });

      expect(chatService.getOfflineMessageCount).toHaveBeenCalled();
      expect(result.current.offlineCount).toBe(5);
    });

    it('should handle fetch count error', async () => {
      const error = new Error('Network error');
      (chatService.getOfflineMessageCount as any).mockRejectedValue(error);

      const { result } = renderHook(() => useOfflineSync(false));

      await act(async () => {
        await result.current.fetchOfflineCount();
      });

      expect(result.current.syncError).toBeTruthy();
    });
  });

  describe('Offline Message Sync', () => {
    it('should sync offline messages', async () => {
      const mockMessages = [
        {
          msg_id: 'msg1',
          sender_id: 'user456',
          recipient_id: 'user123',
          recipient_type: 'user',
          content: 'Message 1',
          sequence_number: 1,
          timestamp: Date.now() - 2000,
        },
        {
          msg_id: 'msg2',
          sender_id: 'user789',
          recipient_id: 'user123',
          recipient_type: 'user',
          content: 'Message 2',
          sequence_number: 2,
          timestamp: Date.now() - 1000,
        },
      ];

      (chatService.getOfflineMessages as any).mockResolvedValue({
        messages: mockMessages,
        next_cursor: '',
        has_more: false,
        total_count: 2,
      });

      const { result } = renderHook(() => useOfflineSync(false));

      await act(async () => {
        await result.current.syncOfflineMessages();
      });

      expect(chatService.getOfflineMessages).toHaveBeenCalled();
      expect(result.current.offlineMessages.length).toBe(2);
      expect(result.current.offlineMessages[0].content).toBe('Message 1');
      expect(result.current.offlineMessages[1].content).toBe('Message 2');
      expect(result.current.offlineCount).toBe(0); // Reset after sync
    });

    it('should handle pagination', async () => {
      const page1Messages = Array.from({ length: 50 }, (_, i) => ({
        msg_id: `msg${i}`,
        sender_id: 'user456',
        recipient_id: 'user123',
        recipient_type: 'user',
        content: `Message ${i}`,
        sequence_number: i,
        timestamp: Date.now() - (50 - i) * 1000,
      }));

      const page2Messages = Array.from({ length: 30 }, (_, i) => ({
        msg_id: `msg${i + 50}`,
        sender_id: 'user456',
        recipient_id: 'user123',
        recipient_type: 'user',
        content: `Message ${i + 50}`,
        sequence_number: i + 50,
        timestamp: Date.now() - (30 - i) * 1000,
      }));

      (chatService.getOfflineMessages as any)
        .mockResolvedValueOnce({
          messages: page1Messages,
          next_cursor: 'cursor1',
          has_more: true,
          total_count: 80,
        })
        .mockResolvedValueOnce({
          messages: page2Messages,
          next_cursor: '',
          has_more: false,
          total_count: 80,
        });

      const { result } = renderHook(() => useOfflineSync(false, { pageSize: 50 }));

      await act(async () => {
        await result.current.syncOfflineMessages();
      });

      expect(chatService.getOfflineMessages).toHaveBeenCalledTimes(2);
      expect(result.current.offlineMessages.length).toBe(80);
    });

    it('should respect maxMessages limit', async () => {
      const mockMessages = Array.from({ length: 100 }, (_, i) => ({
        msg_id: `msg${i}`,
        sender_id: 'user456',
        recipient_id: 'user123',
        recipient_type: 'user',
        content: `Message ${i}`,
        sequence_number: i,
        timestamp: Date.now() - (100 - i) * 1000,
      }));

      (chatService.getOfflineMessages as any).mockResolvedValue({
        messages: mockMessages,
        next_cursor: 'cursor1',
        has_more: true,
        total_count: 200,
      });

      const { result } = renderHook(() => useOfflineSync(false, { 
        pageSize: 100,
        maxMessages: 100,
      }));

      await act(async () => {
        await result.current.syncOfflineMessages();
      });

      expect(result.current.offlineMessages.length).toBe(100);
    });

    it('should sort messages by timestamp', async () => {
      const mockMessages = [
        {
          msg_id: 'msg2',
          sender_id: 'user456',
          recipient_id: 'user123',
          recipient_type: 'user',
          content: 'Message 2',
          sequence_number: 2,
          timestamp: Date.now() - 1000,
        },
        {
          msg_id: 'msg1',
          sender_id: 'user456',
          recipient_id: 'user123',
          recipient_type: 'user',
          content: 'Message 1',
          sequence_number: 1,
          timestamp: Date.now() - 2000,
        },
      ];

      (chatService.getOfflineMessages as any).mockResolvedValue({
        messages: mockMessages,
        next_cursor: '',
        has_more: false,
        total_count: 2,
      });

      const { result } = renderHook(() => useOfflineSync(false));

      await act(async () => {
        await result.current.syncOfflineMessages();
      });

      expect(result.current.offlineMessages[0].msg_id).toBe('msg1');
      expect(result.current.offlineMessages[1].msg_id).toBe('msg2');
    });

    it('should handle sync error', async () => {
      const error = new Error('Sync failed');
      (chatService.getOfflineMessages as any).mockRejectedValue(error);

      const { result } = renderHook(() => useOfflineSync(false));

      await act(async () => {
        await result.current.syncOfflineMessages();
      });

      expect(result.current.syncError).toBeTruthy();
      expect(result.current.isSyncing).toBe(false);
    });

    it('should not sync when already syncing', async () => {
      (chatService.getOfflineMessages as any).mockImplementation(() => 
        new Promise(resolve => setTimeout(() => resolve({
          messages: [],
          next_cursor: '',
          has_more: false,
          total_count: 0,
        }), 100)),
      );

      const { result } = renderHook(() => useOfflineSync(false));

      // Start first sync
      act(() => {
        result.current.syncOfflineMessages();
      });

      expect(result.current.isSyncing).toBe(true);

      // Try to start second sync
      await act(async () => {
        await result.current.syncOfflineMessages();
      });

      // Should only call once
      expect(chatService.getOfflineMessages).toHaveBeenCalledTimes(1);
    });
  });

  describe('Auto-sync', () => {
    it('should auto-sync on connect', async () => {
      (chatService.getOfflineMessageCount as any).mockResolvedValue({
        count: 3,
        oldest_timestamp: Date.now() - 1000,
        newest_timestamp: Date.now(),
      });

      (chatService.getOfflineMessages as any).mockResolvedValue({
        messages: [],
        next_cursor: '',
        has_more: false,
        total_count: 0,
      });

      const { rerender } = renderHook(
        ({ isConnected }) => useOfflineSync(isConnected, { autoSync: true }),
        { initialProps: { isConnected: false } },
      );

      expect(chatService.getOfflineMessageCount).not.toHaveBeenCalled();

      // Simulate connection
      rerender({ isConnected: true });

      await waitFor(() => {
        expect(chatService.getOfflineMessageCount).toHaveBeenCalled();
        expect(chatService.getOfflineMessages).toHaveBeenCalled();
      });
    });

    it('should not auto-sync when disabled', () => {
      const { rerender } = renderHook(
        ({ isConnected }) => useOfflineSync(isConnected, { autoSync: false }),
        { initialProps: { isConnected: false } },
      );

      rerender({ isConnected: true });

      expect(chatService.getOfflineMessageCount).not.toHaveBeenCalled();
      expect(chatService.getOfflineMessages).not.toHaveBeenCalled();
    });
  });

  describe('Clear Messages', () => {
    it('should clear offline messages', async () => {
      (chatService.getOfflineMessages as any).mockResolvedValue({
        messages: [
          {
            msg_id: 'msg1',
            sender_id: 'user456',
            recipient_id: 'user123',
            recipient_type: 'user',
            content: 'Message 1',
            sequence_number: 1,
            timestamp: Date.now(),
          },
        ],
        next_cursor: '',
        has_more: false,
        total_count: 1,
      });

      const { result } = renderHook(() => useOfflineSync(false));

      await act(async () => {
        await result.current.syncOfflineMessages();
      });

      expect(result.current.offlineMessages.length).toBe(1);

      act(() => {
        result.current.clearOfflineMessages();
      });

      expect(result.current.offlineMessages.length).toBe(0);
      expect(result.current.offlineCount).toBe(0);
      expect(result.current.syncError).toBeNull();
    });
  });
});
