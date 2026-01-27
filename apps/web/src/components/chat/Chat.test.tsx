/**
 * Chat Component Integration Tests
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { Chat } from './Chat';
import { useChat } from '@/hooks/useChat';

// Mock useChat hook
vi.mock('@/hooks/useChat');

describe('Chat', () => {
  const mockToken = 'test-token';
  const mockUserId = 'user123';

  beforeEach(() => {
    vi.clearAllMocks();
    
    // Set default mock implementation
    vi.mocked(useChat).mockReturnValue({
      messages: [],
      connectionState: 'disconnected',
      error: null,
      isConnected: false,
      isConnecting: false,
      userId: undefined,
      deviceId: undefined,
      sendPrivateMessage: vi.fn(() => Promise.resolve('msg123')),
      sendGroupMessage: vi.fn(() => Promise.resolve('msg456')),
      sendReadReceipt: vi.fn(),
      clearMessages: vi.fn(),
      connect: vi.fn(() => Promise.resolve()),
      disconnect: vi.fn(),
      reconnectInfo: null,
      offlineCount: 0,
      isSyncingOffline: false,
      syncOfflineMessages: vi.fn(() => Promise.resolve()),
    });
  });

  describe('Rendering', () => {
    it('should render all sub-components', () => {
      render(<Chat token={mockToken} userId={mockUserId} />);
      
      // Should render connection status
      expect(screen.getByText(/Disconnected/i)).toBeInTheDocument();
      
      // Should render message list (empty state)
      expect(screen.getByText(/No messages yet/i)).toBeInTheDocument();
      
      // Should render message input
      expect(screen.getByPlaceholderText(/Type a message/i)).toBeInTheDocument();
    });

    it('should apply custom style', () => {
      const customStyle = { border: '2px solid red' };
      const { container } = render(
        <Chat token={mockToken} userId={mockUserId} style={customStyle} />,
      );
      
      const chatContainer = container.firstChild as HTMLElement;
      expect(chatContainer.style.border).toBe('2px solid red');
    });
  });

  describe('Connection Status', () => {
    it('should show connected status', () => {
      vi.mocked(useChat).mockReturnValue({
        messages: [],
        connectionState: 'connected',
        error: null,
        isConnected: true,
        isConnecting: false,
        userId: 'user123',
        deviceId: 'device456',
        sendPrivateMessage: vi.fn(),
        sendGroupMessage: vi.fn(),
        sendReadReceipt: vi.fn(),
        clearMessages: vi.fn(),
        connect: vi.fn(),
        disconnect: vi.fn(),
        reconnectInfo: null,
        offlineCount: 0,
        isSyncingOffline: false,
        syncOfflineMessages: vi.fn(),
      });

      render(<Chat token={mockToken} userId={mockUserId} />);
      
      expect(screen.getByText(/Connected/i)).toBeInTheDocument();
      expect(screen.getByText(/user123/)).toBeInTheDocument();
    });

    it('should show error state', () => {
      const error = new Error('Connection failed');
      vi.mocked(useChat).mockReturnValue({
        messages: [],
        connectionState: 'disconnected',
        error,
        isConnected: false,
        isConnecting: false,
        userId: undefined,
        deviceId: undefined,
        sendPrivateMessage: vi.fn(),
        sendGroupMessage: vi.fn(),
        sendReadReceipt: vi.fn(),
        clearMessages: vi.fn(),
        connect: vi.fn(),
        disconnect: vi.fn(),
        reconnectInfo: null,
        offlineCount: 0,
        isSyncingOffline: false,
        syncOfflineMessages: vi.fn(),
      });

      render(<Chat token={mockToken} userId={mockUserId} />);
      
      expect(screen.getByText(/Connection failed/i)).toBeInTheDocument();
    });
  });

  describe('Message Display', () => {
    it('should display messages', () => {
      vi.mocked(useChat).mockReturnValue({
        messages: [
          {
            type: 'message',
            msg_id: 'msg1',
            sender_id: 'user456',
            recipient_id: 'user123',
            recipient_type: 'user',
            content: 'Hello!',
            sequence_number: 1,
            timestamp: Date.now(),
            status: 'received',
            isOwn: false,
          },
        ],
        connectionState: 'connected',
        error: null,
        isConnected: true,
        isConnecting: false,
        userId: 'user123',
        deviceId: 'device456',
        sendPrivateMessage: vi.fn(),
        sendGroupMessage: vi.fn(),
        sendReadReceipt: vi.fn(),
        clearMessages: vi.fn(),
        connect: vi.fn(),
        disconnect: vi.fn(),
        reconnectInfo: null,
        offlineCount: 0,
        isSyncingOffline: false,
        syncOfflineMessages: vi.fn(),
      });

      render(<Chat token={mockToken} userId={mockUserId} />);
      
      expect(screen.getByText('Hello!')).toBeInTheDocument();
    });
  });

  describe('Message Sending', () => {
    it('should send private message', async () => {
      const mockSendPrivateMessage = vi.fn(() => Promise.resolve('msg123'));
      vi.mocked(useChat).mockReturnValue({
        messages: [],
        connectionState: 'connected',
        error: null,
        isConnected: true,
        isConnecting: false,
        userId: 'user123',
        deviceId: 'device456',
        sendPrivateMessage: mockSendPrivateMessage,
        sendGroupMessage: vi.fn(),
        sendReadReceipt: vi.fn(),
        clearMessages: vi.fn(),
        connect: vi.fn(),
        disconnect: vi.fn(),
        reconnectInfo: null,
        offlineCount: 0,
        isSyncingOffline: false,
        syncOfflineMessages: vi.fn(),
      });

      render(<Chat token={mockToken} userId={mockUserId} />);
      
      const recipientInput = screen.getByPlaceholderText(/Enter user ID/i);
      const messageInput = screen.getByPlaceholderText(/Type a message/i);
      const sendButton = screen.getByText(/Send/i);
      
      fireEvent.change(recipientInput, { target: { value: 'user456' } });
      fireEvent.change(messageInput, { target: { value: 'Hello!' } });
      fireEvent.click(sendButton);
      
      await waitFor(() => {
        expect(mockSendPrivateMessage).toHaveBeenCalledWith('user456', 'Hello!');
      });
    });

    it('should send group message', async () => {
      const mockSendGroupMessage = vi.fn(() => Promise.resolve('msg456'));
      vi.mocked(useChat).mockReturnValue({
        messages: [],
        connectionState: 'connected',
        error: null,
        isConnected: true,
        isConnecting: false,
        userId: 'user123',
        deviceId: 'device456',
        sendPrivateMessage: vi.fn(),
        sendGroupMessage: mockSendGroupMessage,
        sendReadReceipt: vi.fn(),
        clearMessages: vi.fn(),
        connect: vi.fn(),
        disconnect: vi.fn(),
        reconnectInfo: null,
        offlineCount: 0,
        isSyncingOffline: false,
        syncOfflineMessages: vi.fn(),
      });

      render(<Chat token={mockToken} userId={mockUserId} />);
      
      const recipientInput = screen.getByPlaceholderText(/Enter user ID/i);
      const messageInput = screen.getByPlaceholderText(/Type a message/i);
      const typeSelector = screen.getByRole('combobox');
      const sendButton = screen.getByText(/Send/i);
      
      fireEvent.change(recipientInput, { target: { value: 'group789' } });
      fireEvent.change(messageInput, { target: { value: 'Hello group!' } });
      fireEvent.change(typeSelector, { target: { value: 'group' } });
      fireEvent.click(sendButton);
      
      await waitFor(() => {
        expect(mockSendGroupMessage).toHaveBeenCalledWith('group789', 'Hello group!');
      });
    });

    it('should disable input when disconnected', () => {
      render(<Chat token={mockToken} userId={mockUserId} />);
      
      const messageInput = screen.getByPlaceholderText(/Type a message/i) as HTMLInputElement;
      const sendButton = screen.getByText(/Send/i) as HTMLButtonElement;
      
      expect(messageInput.disabled).toBe(true);
      expect(sendButton.disabled).toBe(true);
    });
  });

  describe('Offline Sync', () => {
    it('should show offline message count', () => {
      vi.mocked(useChat).mockReturnValue({
        messages: [],
        connectionState: 'connected',
        error: null,
        isConnected: true,
        isConnecting: false,
        userId: 'user123',
        deviceId: 'device456',
        sendPrivateMessage: vi.fn(),
        sendGroupMessage: vi.fn(),
        sendReadReceipt: vi.fn(),
        clearMessages: vi.fn(),
        connect: vi.fn(),
        disconnect: vi.fn(),
        reconnectInfo: null,
        offlineCount: 5,
        isSyncingOffline: false,
        syncOfflineMessages: vi.fn(),
      });

      render(<Chat token={mockToken} userId={mockUserId} />);
      
      expect(screen.getByText('5')).toBeInTheDocument();
      expect(screen.getByText(/offline messages/i)).toBeInTheDocument();
    });

    it('should trigger manual sync', async () => {
      const mockSyncOfflineMessages = vi.fn(() => Promise.resolve());
      vi.mocked(useChat).mockReturnValue({
        messages: [],
        connectionState: 'connected',
        error: null,
        isConnected: true,
        isConnecting: false,
        userId: 'user123',
        deviceId: 'device456',
        sendPrivateMessage: vi.fn(),
        sendGroupMessage: vi.fn(),
        sendReadReceipt: vi.fn(),
        clearMessages: vi.fn(),
        connect: vi.fn(),
        disconnect: vi.fn(),
        reconnectInfo: null,
        offlineCount: 5,
        isSyncingOffline: false,
        syncOfflineMessages: mockSyncOfflineMessages,
      });

      render(<Chat token={mockToken} userId={mockUserId} />);
      
      const syncButton = screen.getByRole('button', { name: /sync/i });
      fireEvent.click(syncButton);
      
      await waitFor(() => {
        expect(mockSyncOfflineMessages).toHaveBeenCalled();
      });
    });
  });

  describe('Props', () => {
    it('should pass autoConnect prop', () => {
      render(<Chat token={mockToken} userId={mockUserId} autoConnect={false} />);
      
      expect(useChat).toHaveBeenCalledWith(
        expect.objectContaining({ autoConnect: false }),
      );
    });

    it('should pass autoReadReceipt prop', () => {
      render(<Chat token={mockToken} userId={mockUserId} autoReadReceipt={false} />);
      
      expect(useChat).toHaveBeenCalledWith(
        expect.objectContaining({ autoReadReceipt: false }),
      );
    });

    it('should hide recipient input when showRecipientInput is false', () => {
      render(<Chat token={mockToken} userId={mockUserId} showRecipientInput={false} />);
      
      expect(screen.queryByPlaceholderText(/Enter user ID/i)).not.toBeInTheDocument();
    });
  });
});
