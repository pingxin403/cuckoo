/**
 * MessageInput Component
 * 
 * Input area for composing and sending messages with recipient selection.
 */

import { useState } from 'react';

export type RecipientType = 'user' | 'group';

interface MessageInputProps {
  onSend: (recipientId: string, content: string, recipientType: RecipientType) => Promise<void>;
  onClear?: () => void;
  disabled?: boolean;
  showRecipientInput?: boolean;
  placeholder?: string;
  style?: React.CSSProperties;
}

export function MessageInput({ 
  onSend, 
  onClear,
  disabled = false,
  showRecipientInput = true,
  placeholder = 'Type a message...',
  style, 
}: MessageInputProps) {
  const [recipientId, setRecipientId] = useState('');
  const [messageContent, setMessageContent] = useState('');
  const [recipientType, setRecipientType] = useState<RecipientType>('user');
  const [sending, setSending] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!messageContent.trim() || (showRecipientInput && !recipientId.trim()) || sending) {
      return;
    }

    setSending(true);
    try {
      await onSend(recipientId, messageContent, recipientType);
      setMessageContent('');
    } catch (err) {
      console.error('Failed to send message:', err);
    } finally {
      setSending(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit(e as any);
    }
  };

  return (
    <form 
      onSubmit={handleSubmit}
      style={{ 
        padding: '16px', 
        borderTop: '1px solid #ddd',
        backgroundColor: '#fff',
        ...style,
      }}
    >
      {/* Recipient Input */}
      {showRecipientInput && (
        <div style={{ 
          display: 'flex', 
          gap: '8px', 
          marginBottom: '8px', 
        }}>
          <select
            value={recipientType}
            onChange={(e) => setRecipientType(e.target.value as RecipientType)}
            disabled={disabled || sending}
            style={{
              padding: '8px',
              border: '1px solid #ddd',
              borderRadius: '4px',
              fontSize: '14px',
            }}
          >
            <option value="user">Private</option>
            <option value="group">Group</option>
          </select>
          
          <input
            type="text"
            value={recipientId}
            onChange={(e) => setRecipientId(e.target.value)}
            placeholder={recipientType === 'user' ? 'Enter user ID' : 'Enter group ID'}
            disabled={disabled || sending}
            style={{
              flex: 1,
              padding: '8px',
              border: '1px solid #ddd',
              borderRadius: '4px',
              fontSize: '14px',
            }}
          />
        </div>
      )}

      {/* Message Input */}
      <div style={{ display: 'flex', gap: '8px' }}>
        <input
          type="text"
          value={messageContent}
          onChange={(e) => setMessageContent(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          disabled={disabled || sending}
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
          disabled={disabled || sending || !messageContent.trim() || (showRecipientInput && !recipientId.trim())}
          style={{
            padding: '8px 16px',
            backgroundColor: (!disabled && !sending) ? '#2196F3' : '#ccc',
            color: '#fff',
            border: 'none',
            borderRadius: '4px',
            cursor: (!disabled && !sending) ? 'pointer' : 'not-allowed',
            fontSize: '14px',
          }}
        >
          {sending ? 'Sending...' : 'Send'}
        </button>
        
        {onClear && (
          <button
            type="button"
            onClick={onClear}
            disabled={sending}
            style={{
              padding: '8px 16px',
              backgroundColor: '#f44336',
              color: '#fff',
              border: 'none',
              borderRadius: '4px',
              cursor: sending ? 'not-allowed' : 'pointer',
              fontSize: '14px',
            }}
          >
            Clear
          </button>
        )}
      </div>
    </form>
  );
}
