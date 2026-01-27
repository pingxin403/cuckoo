/**
 * ChatDemo Component
 * 
 * Demo component showcasing the IM Client SDK integration.
 * Provides a simple chat interface for testing message sending and receiving.
 */

import { useState, useRef, useEffect } from 'react';
import { useChat } from '@/hooks/useChat';
import type { ChatMessage } from '@/hooks/useChat';

interface ChatDemoProps {
  /** JWT authentication token */
  token: string;
  
  /** Current user ID (for marking own messages) */
  userId?: string;
}

export function ChatDemo({ token, userId }: ChatDemoProps) {
  const [recipientId, setRecipientId] = useState('');
  const [messageContent, setMessageContent] = useState('');
  const [recipientType, setRecipientType] = useState<'user' | 'group'>('user');
  const messagesEndRef = useRef<HTMLDivElement>(null);

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
  } = useChat({
    token,
    userId,
    autoConnect: true,
    autoReadReceipt: true,
  });

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleSendMessage = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!messageContent.trim() || !recipientId.trim()) {
      return;
    }

    try {
      if (recipientType === 'user') {
        await sendPrivateMessage(recipientId, messageContent);
      } else {
        await sendGroupMessage(recipientId, messageContent);
      }
      
      setMessageContent('');
    } catch (err) {
      console.error('Failed to send message:', err);
    }
  };

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
      case 'pending': return '发送中...';
      case 'sent': return '已发送';
      case 'delivered': return '已送达';
      case 'read': return '已读';
      case 'failed': return '发送失败';
      default: return '';
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
    }}>
      {/* Header */}
      <div style={{ 
        padding: '16px', 
        borderBottom: '1px solid #ddd',
        backgroundColor: '#f5f5f5',
      }}>
        <h3 style={{ margin: '0 0 8px 0' }}>IM 聊天演示</h3>
        
        {/* Connection Status */}
        <div style={{ 
          display: 'flex', 
          alignItems: 'center', 
          gap: '8px',
          fontSize: '14px',
        }}>
          <div style={{
            width: '8px',
            height: '8px',
            borderRadius: '50%',
            backgroundColor: isConnected ? '#4CAF50' : isConnecting ? '#FFC107' : '#f44336',
          }} />
          <span>
            {isConnected && '已连接'}
            {isConnecting && '连接中...'}
            {!isConnected && !isConnecting && '未连接'}
          </span>
          
          {reconnectInfo && (
            <span style={{ color: '#FFC107', marginLeft: '8px' }}>
              重连中 ({reconnectInfo.attempt}/{reconnectInfo.maxAttempts})
            </span>
          )}
        </div>

        {/* User Info */}
        {authenticatedUserId && (
          <div style={{ fontSize: '12px', color: '#666', marginTop: '4px' }}>
            用户ID: {authenticatedUserId} | 设备ID: {deviceId}
          </div>
        )}

        {/* Error Display */}
        {error && (
          <div style={{ 
            marginTop: '8px',
            padding: '8px',
            backgroundColor: '#ffebee',
            color: '#c62828',
            borderRadius: '4px',
            fontSize: '12px',
          }}>
            错误: {error.message}
          </div>
        )}
      </div>

      {/* Messages Area */}
      <div style={{ 
        flex: 1, 
        overflowY: 'auto', 
        padding: '16px',
        backgroundColor: '#fafafa',
      }}>
        {messages.length === 0 ? (
          <div style={{ 
            textAlign: 'center', 
            color: '#999', 
            marginTop: '40px', 
          }}>
            暂无消息
          </div>
        ) : (
          messages.map((msg) => (
            <div
              key={msg.msg_id}
              style={{
                marginBottom: '12px',
                display: 'flex',
                justifyContent: msg.isOwn ? 'flex-end' : 'flex-start',
              }}
            >
              <div style={{
                maxWidth: '70%',
                padding: '8px 12px',
                borderRadius: '8px',
                backgroundColor: msg.isOwn ? '#2196F3' : '#fff',
                color: msg.isOwn ? '#fff' : '#000',
                boxShadow: '0 1px 2px rgba(0,0,0,0.1)',
              }}>
                {/* Sender Info */}
                {!msg.isOwn && (
                  <div style={{ 
                    fontSize: '11px', 
                    opacity: 0.7, 
                    marginBottom: '4px', 
                  }}>
                    {msg.sender_id}
                  </div>
                )}
                
                {/* Message Content */}
                <div style={{ wordBreak: 'break-word' }}>
                  {msg.content}
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
                    {new Date(msg.timestamp).toLocaleTimeString('zh-CN', {
                      hour: '2-digit',
                      minute: '2-digit',
                    })}
                  </span>
                  {msg.isOwn && msg.status && (
                    <span style={{ color: getStatusColor(msg.status) }}>
                      {getStatusText(msg.status)}
                    </span>
                  )}
                </div>
              </div>
            </div>
          ))
        )}
        <div ref={messagesEndRef} />
      </div>

      {/* Input Area */}
      <form 
        onSubmit={handleSendMessage}
        style={{ 
          padding: '16px', 
          borderTop: '1px solid #ddd',
          backgroundColor: '#fff',
        }}
      >
        {/* Recipient Input */}
        <div style={{ 
          display: 'flex', 
          gap: '8px', 
          marginBottom: '8px', 
        }}>
          <select
            value={recipientType}
            onChange={(e) => setRecipientType(e.target.value as 'user' | 'group')}
            style={{
              padding: '8px',
              border: '1px solid #ddd',
              borderRadius: '4px',
              fontSize: '14px',
            }}
          >
            <option value="user">私聊</option>
            <option value="group">群聊</option>
          </select>
          
          <input
            type="text"
            value={recipientId}
            onChange={(e) => setRecipientId(e.target.value)}
            placeholder={recipientType === 'user' ? '输入用户ID' : '输入群组ID'}
            style={{
              flex: 1,
              padding: '8px',
              border: '1px solid #ddd',
              borderRadius: '4px',
              fontSize: '14px',
            }}
          />
        </div>

        {/* Message Input */}
        <div style={{ display: 'flex', gap: '8px' }}>
          <input
            type="text"
            value={messageContent}
            onChange={(e) => setMessageContent(e.target.value)}
            placeholder="输入消息内容..."
            disabled={!isConnected}
            style={{
              flex: 1,
              padding: '8px',
              border: '1px solid #ddd',
              borderRadius: '4px',
              fontSize: '14px',
            }}
          />
          
          <button
            type="submit"
            disabled={!isConnected || !messageContent.trim() || !recipientId.trim()}
            style={{
              padding: '8px 16px',
              backgroundColor: isConnected ? '#2196F3' : '#ccc',
              color: '#fff',
              border: 'none',
              borderRadius: '4px',
              cursor: isConnected ? 'pointer' : 'not-allowed',
              fontSize: '14px',
            }}
          >
            发送
          </button>
          
          <button
            type="button"
            onClick={clearMessages}
            style={{
              padding: '8px 16px',
              backgroundColor: '#f44336',
              color: '#fff',
              border: 'none',
              borderRadius: '4px',
              cursor: 'pointer',
              fontSize: '14px',
            }}
          >
            清空
          </button>
        </div>
      </form>
    </div>
  );
}
