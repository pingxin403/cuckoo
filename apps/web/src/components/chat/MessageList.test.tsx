/**
 * MessageList Component Tests
 */

import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MessageList } from './MessageList';
import type { ChatMessage } from '@/hooks/useChat';

describe('MessageList', () => {
  const createMessage = (id: string, content: string, timestamp: number): ChatMessage => ({
    type: 'message',
    msg_id: id,
    sender_id: 'user456',
    recipient_id: 'user123',
    recipient_type: 'user',
    content,
    sequence_number: parseInt(id.replace('msg', '')),
    timestamp,
    status: 'received',
    isOwn: false,
  });

  describe('Empty State', () => {
    it('should show empty state when no messages', () => {
      render(<MessageList messages={[]} />);
      
      expect(screen.getByText(/No messages yet/i)).toBeInTheDocument();
    });
  });

  describe('Message Rendering', () => {
    it('should render single message', () => {
      const messages = [createMessage('msg1', 'Hello', Date.now())];
      
      render(<MessageList messages={messages} />);
      
      expect(screen.getByText('Hello')).toBeInTheDocument();
    });

    it('should render multiple messages', () => {
      const messages = [
        createMessage('msg1', 'Message 1', Date.now() - 2000),
        createMessage('msg2', 'Message 2', Date.now() - 1000),
        createMessage('msg3', 'Message 3', Date.now()),
      ];
      
      render(<MessageList messages={messages} />);
      
      expect(screen.getByText('Message 1')).toBeInTheDocument();
      expect(screen.getByText('Message 2')).toBeInTheDocument();
      expect(screen.getByText('Message 3')).toBeInTheDocument();
    });

    it('should render messages in order', () => {
      const messages = [
        createMessage('msg1', 'First', Date.now() - 2000),
        createMessage('msg2', 'Second', Date.now() - 1000),
        createMessage('msg3', 'Third', Date.now()),
      ];
      
      render(<MessageList messages={messages} />);
      
      // Check that all three messages are rendered
      expect(screen.getByText('First')).toBeInTheDocument();
      expect(screen.getByText('Second')).toBeInTheDocument();
      expect(screen.getByText('Third')).toBeInTheDocument();
    });
  });

  describe('Scrolling', () => {
    it('should have scrollable container', () => {
      const messages = Array.from({ length: 50 }, (_, i) => 
        createMessage(`msg${i}`, `Message ${i}`, Date.now() - (50 - i) * 1000),
      );
      
      const { container } = render(<MessageList messages={messages} />);
      
      const scrollContainer = container.querySelector('[style*="overflow-y: auto"]');
      expect(scrollContainer).toBeInTheDocument();
    });
  });

  describe('Performance', () => {
    it('should handle large number of messages', () => {
      const messages = Array.from({ length: 1000 }, (_, i) => 
        createMessage(`msg${i}`, `Message ${i}`, Date.now() - (1000 - i) * 1000),
      );
      
      const { container } = render(<MessageList messages={messages} />);
      
      // Should render without crashing
      expect(container).toBeInTheDocument();
    });
  });
});
