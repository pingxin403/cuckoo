/**
 * Chat Component
 * 
 * Complete chat interface built with reusable components.
 * This is a refactored version of ChatDemo using modular components.
 */

import { useChat } from '@/hooks/useChat';
import { MessageList } from './MessageList';
import { MessageInput, type RecipientType } from './MessageInput';
import { ConnectionStatus } from './ConnectionStatus';
import { OfflineSyncStatus } from './OfflineSyncStatus';

interface ChatProps {
  /** JWT authentication token */
  token: string;
  
  /** Current user ID (for marking own messages) */
  userId?: string;
  
  /** Auto-connect on mount (default: true) */
  autoConnect?: boolean;
  
  /** Auto-send read receipts (default: true) */
  autoReadReceipt?: boolean;
  
  /** Show recipient input (default: true) */
  showRecipientInput?: boolean;
  
  /** Container style */
  style?: React.CSSProperties;
}

export function Chat({
  token,
  userId,
  autoConnect = true,
  autoReadReceipt = true,
  showRecipientInput = true,
  style,
}: ChatProps) {
  const {
    messages,
    error,
    isConnected,
    isConnecting,
    userId: authenticatedUserId,
    deviceId,
    sendPrivateMessage,
    sendGroupMessage,
    clearMessages,
    reconnectInfo,
    offlineCount,
    isSyncingOffline,
    syncOfflineMessages,
  } = useChat({
    token,
    userId,
    autoConnect,
    autoReadReceipt,
  });

  const handleSend = async (recipientId: string, content: string, recipientType: RecipientType) => {
    if (recipientType === 'user') {
      await sendPrivateMessage(recipientId, content);
    } else {
      await sendGroupMessage(recipientId, content);
    }
  };

  return (
    <div style={{ 
      display: 'flex', 
      flexDirection: 'column', 
      height: '600px',
      border: '1px solid #ddd',
      borderRadius: '8px',
      overflow: 'hidden',
      ...style,
    }}>
      {/* Header with Connection Status */}
      <ConnectionStatus
        isConnected={isConnected}
        isConnecting={isConnecting}
        userId={authenticatedUserId}
        deviceId={deviceId}
        error={error}
        reconnectInfo={reconnectInfo}
      />

      {/* Offline Sync Status */}
      <OfflineSyncStatus
        offlineCount={offlineCount}
        isSyncing={isSyncingOffline}
        onSync={syncOfflineMessages}
      />

      {/* Messages Area */}
      <MessageList messages={messages} />

      {/* Input Area */}
      <MessageInput
        onSend={handleSend}
        onClear={clearMessages}
        disabled={!isConnected}
        showRecipientInput={showRecipientInput}
      />
    </div>
  );
}
