/**
 * MessageInput Component Tests
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MessageInput } from './MessageInput';

describe('MessageInput', () => {
  const mockOnSend = vi.fn();
  const mockOnClear = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Rendering', () => {
    it('should render input fields', () => {
      render(<MessageInput onSend={mockOnSend} onClear={mockOnClear} />);
      
      expect(screen.getByPlaceholderText(/Enter user ID/i)).toBeInTheDocument();
      expect(screen.getByPlaceholderText(/Type a message/i)).toBeInTheDocument();
    });

    it('should render send button', () => {
      render(<MessageInput onSend={mockOnSend} onClear={mockOnClear} />);
      
      expect(screen.getByText(/Send/i)).toBeInTheDocument();
    });

    it('should render clear button', () => {
      render(<MessageInput onSend={mockOnSend} onClear={mockOnClear} />);
      
      expect(screen.getByText(/Clear/i)).toBeInTheDocument();
    });

    it('should render recipient type selector', () => {
      render(<MessageInput onSend={mockOnSend} onClear={mockOnClear} />);
      
      expect(screen.getByRole('combobox')).toBeInTheDocument();
    });
  });

  describe('Input Handling', () => {
    it('should update recipient ID input', () => {
      render(<MessageInput onSend={mockOnSend} onClear={mockOnClear} />);
      
      const recipientInput = screen.getByPlaceholderText(/Enter user ID/i) as HTMLInputElement;
      fireEvent.change(recipientInput, { target: { value: 'user456' } });
      
      expect(recipientInput.value).toBe('user456');
    });

    it('should update message input', () => {
      render(<MessageInput onSend={mockOnSend} onClear={mockOnClear} />);
      
      const messageInput = screen.getByPlaceholderText(/Type a message/i) as HTMLInputElement;
      fireEvent.change(messageInput, { target: { value: 'Hello, world!' } });
      
      expect(messageInput.value).toBe('Hello, world!');
    });

    it('should update recipient type', () => {
      render(<MessageInput onSend={mockOnSend} onClear={mockOnClear} />);
      
      const typeSelector = screen.getByRole('combobox') as HTMLSelectElement;
      fireEvent.change(typeSelector, { target: { value: 'group' } });
      
      expect(typeSelector.value).toBe('group');
    });
  });

  describe('Message Sending', () => {
    it('should call onSend with correct parameters', async () => {
      render(<MessageInput onSend={mockOnSend} onClear={mockOnClear} />);
      
      const recipientInput = screen.getByPlaceholderText(/Enter user ID/i);
      const messageInput = screen.getByPlaceholderText(/Type a message/i);
      const sendButton = screen.getByText(/Send/i);
      
      fireEvent.change(recipientInput, { target: { value: 'user456' } });
      fireEvent.change(messageInput, { target: { value: 'Hello!' } });
      fireEvent.click(sendButton);
      
      await waitFor(() => {
        expect(mockOnSend).toHaveBeenCalledWith('user456', 'Hello!', 'user');
      });
    });

    it('should clear message input after sending', async () => {
      render(<MessageInput onSend={mockOnSend} onClear={mockOnClear} />);
      
      const recipientInput = screen.getByPlaceholderText(/Enter user ID/i);
      const messageInput = screen.getByPlaceholderText(/Type a message/i) as HTMLInputElement;
      const sendButton = screen.getByText(/Send/i);
      
      fireEvent.change(recipientInput, { target: { value: 'user456' } });
      fireEvent.change(messageInput, { target: { value: 'Hello!' } });
      fireEvent.click(sendButton);
      
      await waitFor(() => {
        expect(messageInput.value).toBe('');
      });
    });

    it('should not send empty message', async () => {
      render(<MessageInput onSend={mockOnSend} onClear={mockOnClear} />);
      
      const recipientInput = screen.getByPlaceholderText(/Enter user ID/i);
      const sendButton = screen.getByText(/Send/i);
      
      fireEvent.change(recipientInput, { target: { value: 'user456' } });
      fireEvent.click(sendButton);
      
      expect(mockOnSend).not.toHaveBeenCalled();
    });

    it('should not send without recipient', async () => {
      render(<MessageInput onSend={mockOnSend} onClear={mockOnClear} />);
      
      const messageInput = screen.getByPlaceholderText(/Type a message/i);
      const sendButton = screen.getByText(/Send/i);
      
      fireEvent.change(messageInput, { target: { value: 'Hello!' } });
      fireEvent.click(sendButton);
      
      expect(mockOnSend).not.toHaveBeenCalled();
    });

    it('should send group message', async () => {
      render(<MessageInput onSend={mockOnSend} onClear={mockOnClear} />);
      
      const recipientInput = screen.getByPlaceholderText(/Enter user ID/i);
      const messageInput = screen.getByPlaceholderText(/Type a message/i);
      const typeSelector = screen.getByRole('combobox');
      const sendButton = screen.getByText(/Send/i);
      
      fireEvent.change(recipientInput, { target: { value: 'group789' } });
      fireEvent.change(messageInput, { target: { value: 'Hello group!' } });
      fireEvent.change(typeSelector, { target: { value: 'group' } });
      fireEvent.click(sendButton);
      
      await waitFor(() => {
        expect(mockOnSend).toHaveBeenCalledWith('group789', 'Hello group!', 'group');
      });
    });
  });

  describe('Keyboard Shortcuts', () => {
    it('should send message on Enter key', async () => {
      render(<MessageInput onSend={mockOnSend} onClear={mockOnClear} />);
      
      const recipientInput = screen.getByPlaceholderText(/Enter user ID/i);
      const messageInput = screen.getByPlaceholderText(/Type a message/i);
      
      fireEvent.change(recipientInput, { target: { value: 'user456' } });
      fireEvent.change(messageInput, { target: { value: 'Hello!' } });
      fireEvent.keyDown(messageInput, { key: 'Enter', code: 'Enter', keyCode: 13 });
      
      await waitFor(() => {
        expect(mockOnSend).toHaveBeenCalledWith('user456', 'Hello!', 'user');
      });
    });

    it('should not send on Shift+Enter', async () => {
      render(<MessageInput onSend={mockOnSend} onClear={mockOnClear} />);
      
      const recipientInput = screen.getByPlaceholderText(/Enter user ID/i);
      const messageInput = screen.getByPlaceholderText(/Type a message/i);
      
      fireEvent.change(recipientInput, { target: { value: 'user456' } });
      fireEvent.change(messageInput, { target: { value: 'Hello!' } });
      fireEvent.keyDown(messageInput, { key: 'Enter', code: 'Enter', keyCode: 13, shiftKey: true });
      
      // Wait a bit to ensure no call was made
      await new Promise(resolve => setTimeout(resolve, 100));
      
      expect(mockOnSend).not.toHaveBeenCalled();
    });
  });

  describe('Clear Functionality', () => {
    it('should call onClear when clear button clicked', () => {
      render(<MessageInput onSend={mockOnSend} onClear={mockOnClear} />);
      
      const clearButton = screen.getByText(/Clear/i);
      fireEvent.click(clearButton);
      
      expect(mockOnClear).toHaveBeenCalled();
    });
  });

  describe('Disabled State', () => {
    it('should disable inputs when disabled prop is true', () => {
      render(<MessageInput onSend={mockOnSend} onClear={mockOnClear} disabled={true} />);
      
      const recipientInput = screen.getByPlaceholderText(/Enter user ID/i) as HTMLInputElement;
      const messageInput = screen.getByPlaceholderText(/Type a message/i) as HTMLInputElement;
      const sendButton = screen.getByText(/Send/i) as HTMLButtonElement;
      
      expect(recipientInput.disabled).toBe(true);
      expect(messageInput.disabled).toBe(true);
      expect(sendButton.disabled).toBe(true);
    });

    it('should enable inputs when disabled prop is false', () => {
      render(<MessageInput onSend={mockOnSend} onClear={mockOnClear} disabled={false} />);
      
      const recipientInput = screen.getByPlaceholderText(/Enter user ID/i) as HTMLInputElement;
      const messageInput = screen.getByPlaceholderText(/Type a message/i) as HTMLInputElement;
      
      expect(recipientInput.disabled).toBe(false);
      expect(messageInput.disabled).toBe(false);
      
      // Send button is disabled when no content, but not because of disabled prop
      // Let's add content to verify it's not disabled by the prop
      fireEvent.change(recipientInput, { target: { value: 'user123' } });
      fireEvent.change(messageInput, { target: { value: 'Hello' } });
      
      const sendButton = screen.getByText(/Send/i) as HTMLButtonElement;
      expect(sendButton.disabled).toBe(false);
    });
  });

  describe('Recipient Input Visibility', () => {
    it('should show recipient input by default', () => {
      render(<MessageInput onSend={mockOnSend} onClear={mockOnClear} />);
      
      expect(screen.getByPlaceholderText(/Enter user ID/i)).toBeInTheDocument();
    });

    it('should hide recipient input when showRecipientInput is false', () => {
      render(<MessageInput onSend={mockOnSend} onClear={mockOnClear} showRecipientInput={false} />);
      
      expect(screen.queryByPlaceholderText(/Enter user ID/i)).not.toBeInTheDocument();
    });
  });
});
