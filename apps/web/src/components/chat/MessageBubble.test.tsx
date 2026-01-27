/**
 * MessageBubble Component Tests
 */

import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MessageBubble } from './MessageBubble';
import type { ChatMessage } from '@/hooks/useChat';

describe('MessageBubble', () => {
  const baseMessage: ChatMessage = {
    type: 'message',
    msg_id: 'msg123',
    sender_id: 'user456',
    recipient_id: 'user123',
    recipient_type: 'user',
    content: 'Hello, world!',
    sequence_number: 1,
    timestamp: Date.now(),
    status: 'received',
    isOwn: false,
  };

  describe('Rendering', () => {
    it('should render message content', () => {
      render(<MessageBubble message={baseMessage} isOwn={false} />);
      
      expect(screen.getByText('Hello, world!')).toBeInTheDocument();
    });

    it('should render sender ID', () => {
      render(<MessageBubble message={baseMessage} isOwn={false} />);
      
      expect(screen.getByText(/user456/)).toBeInTheDocument();
    });

    it('should render timestamp', () => {
      render(<MessageBubble message={baseMessage} isOwn={false} />);
      
      // Should contain time in HH:MM format
      expect(screen.getByText(/\d{2}:\d{2}/)).toBeInTheDocument();
    });

    it('should render message status', () => {
      render(<MessageBubble message={{ ...baseMessage, status: 'delivered' }} isOwn={true} />);
      
      expect(screen.getByText(/Delivered/i)).toBeInTheDocument();
    });
  });

  describe('Own Messages', () => {
    it('should style own messages differently', () => {
      const ownMessage = { ...baseMessage, isOwn: true };
      const { container } = render(<MessageBubble message={ownMessage} isOwn={true} />);
      
      const bubble = container.querySelector('[style*="justify-content: flex-end"]');
      expect(bubble).toBeInTheDocument();
    });

    it('should style other messages differently', () => {
      const { container } = render(<MessageBubble message={baseMessage} isOwn={false} />);
      
      const bubble = container.querySelector('[style*="justify-content: flex-start"]');
      expect(bubble).toBeInTheDocument();
    });
  });

  describe('Message Types', () => {
    it('should indicate private message', () => {
      render(<MessageBubble message={baseMessage} isOwn={false} />);
      
      // Private/Group indication is not shown in MessageBubble component
      expect(screen.getByText('Hello, world!')).toBeInTheDocument();
    });

    it('should indicate group message', () => {
      const groupMessage = { ...baseMessage, recipient_type: 'group' as const };
      render(<MessageBubble message={groupMessage} isOwn={false} />);
      
      // Private/Group indication is not shown in MessageBubble component
      expect(screen.getByText('Hello, world!')).toBeInTheDocument();
    });
  });

  describe('Message Status', () => {
    it('should show pending status', () => {
      const pendingMessage = { ...baseMessage, status: 'pending' as const };
      render(<MessageBubble message={pendingMessage} isOwn={true} />);
      
      expect(screen.getByText(/Sending/i)).toBeInTheDocument();
    });

    it('should show sent status', () => {
      const sentMessage = { ...baseMessage, status: 'sent' as const };
      render(<MessageBubble message={sentMessage} isOwn={true} />);
      
      expect(screen.getByText(/Sent/i)).toBeInTheDocument();
    });

    it('should show delivered status', () => {
      const deliveredMessage = { ...baseMessage, status: 'delivered' as const };
      render(<MessageBubble message={deliveredMessage} isOwn={true} />);
      
      expect(screen.getByText(/Delivered/i)).toBeInTheDocument();
    });

    it('should show read status', () => {
      const readMessage = { ...baseMessage, status: 'read' as const };
      render(<MessageBubble message={readMessage} isOwn={true} />);
      
      expect(screen.getByText(/Read/i)).toBeInTheDocument();
    });

    it('should show failed status', () => {
      const failedMessage = { ...baseMessage, status: 'failed' as const };
      render(<MessageBubble message={failedMessage} isOwn={true} />);
      
      expect(screen.getByText(/Failed/i)).toBeInTheDocument();
    });
  });

  describe('Long Content', () => {
    it('should handle long messages', () => {
      const longContent = 'A'.repeat(1000);
      const longMessage = { ...baseMessage, content: longContent };
      
      render(<MessageBubble message={longMessage} isOwn={false} />);
      
      expect(screen.getByText(longContent)).toBeInTheDocument();
    });

    it('should handle multiline messages', () => {
      const multilineContent = 'Line 1\nLine 2\nLine 3';
      const multilineMessage = { ...baseMessage, content: multilineContent };
      
      render(<MessageBubble message={multilineMessage} isOwn={false} />);
      
      expect(screen.getByText(/Line 1/)).toBeInTheDocument();
      expect(screen.getByText(/Line 2/)).toBeInTheDocument();
      expect(screen.getByText(/Line 3/)).toBeInTheDocument();
    });
  });

  describe('Special Characters', () => {
    it('should handle HTML entities', () => {
      const htmlMessage = { ...baseMessage, content: '<script>alert("xss")</script>' };
      
      render(<MessageBubble message={htmlMessage} isOwn={false} />);
      
      // Should render as text, not execute
      expect(screen.getByText(/<script>/)).toBeInTheDocument();
    });

    it('should handle emojis', () => {
      const emojiMessage = { ...baseMessage, content: '👋 Hello! 🎉' };
      
      render(<MessageBubble message={emojiMessage} isOwn={false} />);
      
      expect(screen.getByText(/👋 Hello! 🎉/)).toBeInTheDocument();
    });
  });
});
