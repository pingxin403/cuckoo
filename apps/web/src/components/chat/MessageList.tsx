/**
 * MessageList Component
 * 
 * Displays a scrollable list of messages with auto-scroll to bottom.
 */

import { useEffect, useRef } from 'react';
import { MessageBubble } from './MessageBubble';
import type { ChatMessage } from '@/hooks/useChat';

interface MessageListProps {
  messages: ChatMessage[];
  emptyText?: string;
  style?: React.CSSProperties;
}

export function MessageList({ 
  messages, 
  emptyText = 'No messages yet',
  style, 
}: MessageListProps) {
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  return (
    <div style={{ 
      flex: 1, 
      overflowY: 'auto', 
      padding: '16px',
      backgroundColor: '#fafafa',
      ...style,
    }}>
      {messages.length === 0 ? (
        <div style={{ 
          textAlign: 'center', 
          color: '#999', 
          marginTop: '40px', 
        }}>
          {emptyText}
        </div>
      ) : (
        messages.map((msg) => (
          <MessageBubble
            key={msg.msg_id}
            message={msg}
            isOwn={msg.isOwn || false}
          />
        ))
      )}
      <div ref={messagesEndRef} />
    </div>
  );
}
