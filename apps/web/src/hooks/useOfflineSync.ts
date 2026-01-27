/**
 * useOfflineSync Hook
 * 
 * React hook for syncing offline messages when reconnecting.
 * Fetches missed messages and merges them with local message history.
 */

import { useState, useEffect, useCallback } from 'react';
import { chatService } from '@/services/chatService';
import type { ChatMessage } from './useChat';

export interface UseOfflineSyncOptions {
  /** Enable automatic sync on connect (default: true) */
  autoSync?: boolean;
  
  /** Page size for fetching offline messages (default: 50) */
  pageSize?: number;
  
  /** Maximum messages to fetch (default: 500) */
  maxMessages?: number;
}

export interface UseOfflineSyncReturn {
  /** Offline message count */
  offlineCount: number;
  
  /** Is syncing offline messages */
  isSyncing: boolean;
  
  /** Sync error */
  syncError: Error | null;
  
  /** Synced offline messages */
  offlineMessages: ChatMessage[];
  
  /** Manually trigger sync */
  syncOfflineMessages: () => Promise<void>;
  
  /** Fetch offline message count */
  fetchOfflineCount: () => Promise<void>;
  
  /** Clear synced messages */
  clearOfflineMessages: () => void;
}

export function useOfflineSync(
  isConnected: boolean,
  options: UseOfflineSyncOptions = {},
): UseOfflineSyncReturn {
  const {
    autoSync = true,
    pageSize = 50,
    maxMessages = 500,
  } = options;

  const [offlineCount, setOfflineCount] = useState(0);
  const [isSyncing, setIsSyncing] = useState(false);
  const [syncError, setSyncError] = useState<Error | null>(null);
  const [offlineMessages, setOfflineMessages] = useState<ChatMessage[]>([]);

  // Fetch offline message count
  const fetchOfflineCount = useCallback(async () => {
    if (!chatService.isInitialized()) {
      return;
    }

    try {
      const result = await chatService.getOfflineMessageCount();
      setOfflineCount(result.count);
    } catch (err) {
      console.error('[useOfflineSync] Failed to fetch offline count:', err);
      setSyncError(err as Error);
    }
  }, []);

  // Sync offline messages
  const syncOfflineMessages = useCallback(async () => {
    if (!chatService.isInitialized() || isSyncing) {
      return;
    }

    setIsSyncing(true);
    setSyncError(null);

    try {
      const messages: ChatMessage[] = [];
      let cursor = '';
      let hasMore = true;
      let fetchedCount = 0;

      // Fetch messages in pages
      while (hasMore && fetchedCount < maxMessages) {
        const result = await chatService.getOfflineMessages(cursor, pageSize);
        
        // Convert to ChatMessage format
        const chatMessages: ChatMessage[] = result.messages.map(msg => ({
          type: 'message',
          msg_id: msg.msg_id,
          sender_id: msg.sender_id,
          recipient_id: msg.recipient_id,
          recipient_type: msg.recipient_type || 'user',
          content: msg.content,
          sequence_number: msg.sequence_number,
          timestamp: msg.timestamp,
          status: 'received',
          isOwn: false,
        }));

        messages.push(...chatMessages);
        fetchedCount += chatMessages.length;

        cursor = result.next_cursor;
        hasMore = result.has_more;

        // Break if no more messages
        if (!hasMore || !cursor) {
          break;
        }
      }

      // Sort by timestamp (oldest first)
      messages.sort((a, b) => a.timestamp - b.timestamp);

      setOfflineMessages(messages);
      setOfflineCount(0); // Reset count after sync
      
      console.log(`[useOfflineSync] Synced ${messages.length} offline messages`);
    } catch (err) {
      console.error('[useOfflineSync] Failed to sync offline messages:', err);
      setSyncError(err as Error);
    } finally {
      setIsSyncing(false);
    }
  }, [isSyncing, pageSize, maxMessages]);

  // Clear offline messages
  const clearOfflineMessages = useCallback(() => {
    setOfflineMessages([]);
    setOfflineCount(0);
    setSyncError(null);
  }, []);

  // Auto-sync on connect
  useEffect(() => {
    if (isConnected && autoSync && chatService.isInitialized()) {
      // Fetch count first
      fetchOfflineCount();
      
      // Then sync messages
      syncOfflineMessages();
    }
  }, [isConnected, autoSync, fetchOfflineCount, syncOfflineMessages]);

  return {
    offlineCount,
    isSyncing,
    syncError,
    offlineMessages,
    syncOfflineMessages,
    fetchOfflineCount,
    clearOfflineMessages,
  };
}
