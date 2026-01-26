/**
 * MessageBubble Component
 * 
 * Displays a single message bubble with sender info, content, timestamp, and status.
 */

import type { ChatMessage } from '@/hooks/useChat';

interface MessageBubbleProps {
  message: ChatMessage;
  isOwn: boolean;
}

export function MessageBubble({ message, isOwn }: MessageBubbleProps) {
  const getStatusColor = (status?: ChatMessage['status']) => {
    switch (status) {
      case 'pending': return '#999';
      case 'sent': return '#666';
      case 'delivered': return '#4CAF50';
      case 'read': return '#2196F3';
      case 'failed': return '#f44336';
      default: return '#666';
    }
  };

  const getStatusText = (status?: ChatMessage['status']) => {
    switch (status) {
      case 'pending': return 'Sending...';
      case 'sent': return 'Sent';
      case 'delivered': return 'Delivered';
      case 'read': return 'Read';
      case 'failed': return 'Failed';
      default: return '';
    }
  };

  return (
    <div
      style={{
        marginBottom: '12px',
        display: 'flex',
        justifyContent: isOwn ? 'flex-end' : 'flex-start',
      }}
    >
      <div style={{
        maxWidth: '70%',
        padding: '8px 12px',
        borderRadius: '8px',
        backgroundColor: isOwn ? '#2196F3' : '#fff',
        color: isOwn ? '#fff' : '#000',
        boxShadow: '0 1px 2px rgba(0,0,0,0.1)',
      }}>
        {/* Sender Info */}
        {!isOwn && (
          <div style={{ 
            fontSize: '11px', 
            opacity: 0.7, 
            marginBottom: '4px', 
          }}>
            {message.sender_id}
          </div>
        )}
        
        {/* Message Content */}
        <div style={{ wordBreak: 'break-word' }}>
          {message.content}
        </div>
        
        {/* Message Status */}
        <div style={{ 
          fontSize: '10px', 
          opacity: 0.7, 
          marginTop: '4px',
          textAlign: 'right',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}>
          <span>
            {new Date(message.timestamp).toLocaleTimeString('zh-CN', {
              hour: '2-digit',
              minute: '2-digit',
            })}
          </span>
          {isOwn && message.status && (
            <span style={{ color: getStatusColor(message.status) }}>
              {getStatusText(message.status)}
            </span>
          )}
        </div>
      </div>
    </div>
  );
}
