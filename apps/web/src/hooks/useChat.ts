/**
 * useChat Hook
 * 
 * React hook for managing chat functionality with the IM Client SDK.
 * Provides state management, message handling, and connection management.
 */

import { useEffect, useState, useCallback, useRef } from 'react';
import { chatService } from '@/services/chatService';
import { useOfflineSync } from './useOfflineSync';
import type { IncomingMessage, AckMessage, ReadReceiptMessage } from '@/sdk/im-client';

export interface ChatMessage extends IncomingMessage {
  status?: 'pending' | 'sent' | 'delivered' | 'received' | 'read' | 'failed';
  isOwn?: boolean;
}

export interface UseChatOptions {
  /** JWT token for authentication */
  token: string;
  
  /** Auto-connect on mount (default: true) */
  autoConnect?: boolean;
  
  /** Auto-send read receipts (default: true) */
  autoReadReceipt?: boolean;
  
  /** Current user ID (for marking own messages) */
  userId?: string;
  
  /** Enable offline message sync (default: true) */
  enableOfflineSync?: boolean;
  
  /** Offline sync page size (default: 50) */
  offlineSyncPageSize?: number;
}

export interface UseChatReturn {
  /** All messages */
  messages: ChatMessage[];
  
  /** Connection state */
  connectionState: string;
  
  /** Connection error */
  error: Error | null;
  
  /** Is connected */
  isConnected: boolean;
  
  /** Is connecting */
  isConnecting: boolean;
  
  /** Authenticated user ID */
  userId: string | undefined;
  
  /** Device ID */
  deviceId: string | undefined;
  
  /** Send a private message */
  sendPrivateMessage: (recipientId: string, content: string) => Promise<string>;
  
  /** Send a group message */
  sendGroupMessage: (groupId: string, content: string) => Promise<string>;
  
  /** Send a read receipt */
  sendReadReceipt: (msgId: string) => void;
  
  /** Clear all messages */
  clearMessages: () => void;
  
  /** Manually connect */
  connect: () => Promise<void>;
  
  /** Manually disconnect */
  disconnect: () => void;
  
  /** Reconnection attempt info */
  reconnectInfo: { attempt: number; maxAttempts: number } | null;
  
  /** Offline message count */
  offlineCount: number;
  
  /** Is syncing offline messages */
  isSyncingOffline: boolean;
  
  /** Manually sync offline messages */
  syncOfflineMessages: () => Promise<void>;
}

export function useChat(options: UseChatOptions): UseChatReturn {
  const {
    token,
    autoConnect = true,
    autoReadReceipt = true,
    userId: currentUserId,
    enableOfflineSync = true,
    offlineSyncPageSize = 50,
  } = options;

  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [connectionState, setConnectionState] = useState<string>('disconnected');
  const [error, setError] = useState<Error | null>(null);
  const [userId, setUserId] = useState<string | undefined>();
  const [deviceId, setDeviceId] = useState<string | undefined>();
  const [reconnectInfo, setReconnectInfo] = useState<{ attempt: number; maxAttempts: number } | null>(null);

  const isInitialized = useRef(false);

  // Offline message sync
  const {
    offlineCount,
    isSyncing: isSyncingOffline,
    offlineMessages,
    syncOfflineMessages,
    clearOfflineMessages,
  } = useOfflineSync(connectionState === 'connected', {
    autoSync: enableOfflineSync,
    pageSize: offlineSyncPageSize,
  });

  // Merge offline messages with current messages
  useEffect(() => {
    if (offlineMessages.length > 0) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setMessages(prev => {
        // Create a map of existing message IDs
        const existingIds = new Set(prev.map(m => m.msg_id));
        
        // Filter out duplicates
        const newMessages = offlineMessages.filter(m => !existingIds.has(m.msg_id));
        
        // Merge and sort by timestamp
        const merged = [...prev, ...newMessages].sort((a, b) => a.timestamp - b.timestamp);
        
        return merged;
      });
      
      // Clear offline messages after merging
      clearOfflineMessages();
    }
  }, [offlineMessages, clearOfflineMessages]);

  // Initialize chat service
  useEffect(() => {
    if (!token) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setError(new Error('No authentication token provided'));
      return;
    }

    try {
      // Initialize service
      if (!chatService.isInitialized()) {
        chatService.initialize(token);
      }

      // Register event handlers
      chatService.on('onConnected', (uid: string, did: string) => {
        console.log('[useChat] Connected:', uid, did);
        setUserId(uid);
        setDeviceId(did);
        setError(null);
        setReconnectInfo(null);
      });

      chatService.on('onDisconnected', (code: number, reason: string) => {
        console.log('[useChat] Disconnected:', code, reason);
      });

      chatService.on('onError', (err: Error) => {
        console.error('[useChat] Error:', err);
        setError(err);
      });

      chatService.on('onMessage', (message: IncomingMessage) => {
        console.log('[useChat] New message:', message);
        
        // Add message to state
        const chatMessage: ChatMessage = {
          ...message,
          status: 'received',
          isOwn: currentUserId ? message.sender_id === currentUserId : false,
        };
        
        setMessages(prev => [...prev, chatMessage]);

        // Auto-send read receipt
        if (autoReadReceipt && !chatMessage.isOwn) {
          chatService.sendReadReceipt(message.msg_id);
        }
      });

      chatService.on('onAck', (ack: AckMessage) => {
        console.log('[useChat] ACK received:', ack.msg_id, ack.status);
        
        // Update message status
        setMessages(prev => prev.map(msg => 
          msg.msg_id === ack.msg_id 
            ? { ...msg, status: ack.status }
            : msg,
        ));
      });

      chatService.on('onReadReceipt', (receipt: ReadReceiptMessage) => {
        console.log('[useChat] Read receipt:', receipt.msg_id, receipt.reader_id);
        
        // Update message status to read
        setMessages(prev => prev.map(msg => 
          msg.msg_id === receipt.msg_id 
            ? { ...msg, status: 'read' }
            : msg,
        ));
      });

      chatService.on('onStateChange', (state: string) => {
        console.log('[useChat] State changed:', state);
        setConnectionState(state);
      });

      chatService.on('onReconnecting', (attempt: number, maxAttempts: number) => {
        console.log('[useChat] Reconnecting:', attempt, '/', maxAttempts);
        setReconnectInfo({ attempt, maxAttempts });
      });

      isInitialized.current = true;

      // Auto-connect if enabled
      if (autoConnect) {
        chatService.connect().catch(err => {
          console.error('[useChat] Auto-connect failed:', err);
          setError(err);
        });
      }

    } catch (err) {
      console.error('[useChat] Initialization failed:', err);
      setError(err as Error);
    }

    // Cleanup
    return () => {
      if (isInitialized.current) {
        chatService.disconnect();
        // Note: We don't destroy the service here as it's a singleton
        // and might be used by other components
      }
    };
  }, [token, autoConnect, autoReadReceipt, currentUserId]);

  // Send private message
  const sendPrivateMessage = useCallback(async (recipientId: string, content: string): Promise<string> => {
    try {
      const msgId = await chatService.sendPrivateMessage(recipientId, content);
      
      // Add optimistic message
      const optimisticMessage: ChatMessage = {
        type: 'message',
        msg_id: msgId,
        sender_id: userId || 'unknown',
        recipient_id: recipientId,
        recipient_type: 'user',
        content,
        sequence_number: 0, // Will be updated by ACK
        timestamp: Date.now(),
        status: 'pending',
        isOwn: true,
      };
      
      setMessages(prev => [...prev, optimisticMessage]);
      
      return msgId;
    } catch (err) {
      console.error('[useChat] Failed to send private message:', err);
      throw err;
    }
  }, [userId]);

  // Send group message
  const sendGroupMessage = useCallback(async (groupId: string, content: string): Promise<string> => {
    try {
      const msgId = await chatService.sendGroupMessage(groupId, content);
      
      // Add optimistic message
      const optimisticMessage: ChatMessage = {
        type: 'message',
        msg_id: msgId,
        sender_id: userId || 'unknown',
        recipient_id: groupId,
        recipient_type: 'group',
        content,
        sequence_number: 0, // Will be updated by ACK
        timestamp: Date.now(),
        status: 'pending',
        isOwn: true,
      };
      
      setMessages(prev => [...prev, optimisticMessage]);
      
      return msgId;
    } catch (err) {
      console.error('[useChat] Failed to send group message:', err);
      throw err;
    }
  }, [userId]);

  // Send read receipt
  const sendReadReceipt = useCallback((msgId: string) => {
    chatService.sendReadReceipt(msgId);
  }, []);

  // Clear messages
  const clearMessages = useCallback(() => {
    setMessages([]);
  }, []);

  // Manual connect
  const connect = useCallback(async () => {
    try {
      await chatService.connect();
    } catch (err) {
      console.error('[useChat] Connect failed:', err);
      setError(err as Error);
      throw err;
    }
  }, []);

  // Manual disconnect
  const disconnect = useCallback(() => {
    chatService.disconnect();
  }, []);

  return {
    messages,
    connectionState,
    error,
    isConnected: connectionState === 'connected',
    isConnecting: connectionState === 'connecting' || connectionState === 'reconnecting',
    userId,
    deviceId,
    sendPrivateMessage,
    sendGroupMessage,
    sendReadReceipt,
    clearMessages,
    connect,
    disconnect,
    reconnectInfo,
    offlineCount,
    isSyncingOffline,
    syncOfflineMessages,
  };
}
