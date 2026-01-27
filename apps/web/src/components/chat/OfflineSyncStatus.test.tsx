/**
 * OfflineSyncStatus Component Tests
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { OfflineSyncStatus } from './OfflineSyncStatus';

describe('OfflineSyncStatus', () => {
  const mockOnSync = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Visibility', () => {
    it('should not render when no offline messages', () => {
      const { container } = render(
        <OfflineSyncStatus
          offlineCount={0}
          isSyncing={false}
          onSync={mockOnSync}
        />,
      );
      
      expect(container.firstChild).toBeNull();
    });

    it('should render when offline messages exist', () => {
      render(
        <OfflineSyncStatus
          offlineCount={5}
          isSyncing={false}
          onSync={mockOnSync}
        />,
      );
      
      expect(screen.getByText('5')).toBeInTheDocument();
      expect(screen.getByText(/offline messages/i)).toBeInTheDocument();
    });

    it('should render when syncing', () => {
      render(
        <OfflineSyncStatus
          offlineCount={0}
          isSyncing={true}
          onSync={mockOnSync}
        />,
      );
      
      expect(screen.getByText(/Syncing/i)).toBeInTheDocument();
    });
  });

  describe('Offline Count Display', () => {
    it('should show singular form for 1 message', () => {
      render(
        <OfflineSyncStatus
          offlineCount={1}
          isSyncing={false}
          onSync={mockOnSync}
        />,
      );
      
      expect(screen.getByText('1')).toBeInTheDocument();
      expect(screen.getByText('offline message')).toBeInTheDocument();
    });

    it('should show plural form for multiple messages', () => {
      render(
        <OfflineSyncStatus
          offlineCount={5}
          isSyncing={false}
          onSync={mockOnSync}
        />,
      );
      
      expect(screen.getByText('5')).toBeInTheDocument();
      expect(screen.getByText('offline messages')).toBeInTheDocument();
    });

    it('should show large numbers correctly', () => {
      render(
        <OfflineSyncStatus
          offlineCount={999}
          isSyncing={false}
          onSync={mockOnSync}
        />,
      );
      
      expect(screen.getByText('999')).toBeInTheDocument();
      expect(screen.getByText('offline messages')).toBeInTheDocument();
    });
  });

  describe('Syncing State', () => {
    it('should show syncing indicator', () => {
      render(
        <OfflineSyncStatus
          offlineCount={5}
          isSyncing={true}
          onSync={mockOnSync}
        />,
      );
      
      expect(screen.getByText(/Syncing/i)).toBeInTheDocument();
    });

    it('should not show sync button when syncing', () => {
      render(
        <OfflineSyncStatus
          offlineCount={5}
          isSyncing={true}
          onSync={mockOnSync}
        />,
      );
      
      const syncButton = screen.queryByRole('button');
      expect(syncButton).not.toBeInTheDocument();
    });

    it('should enable sync button when not syncing', () => {
      render(
        <OfflineSyncStatus
          offlineCount={5}
          isSyncing={false}
          onSync={mockOnSync}
        />,
      );
      
      const syncButton = screen.getByRole('button');
      expect(syncButton).not.toBeDisabled();
    });
  });

  describe('Sync Button', () => {
    it('should render sync button', () => {
      render(
        <OfflineSyncStatus
          offlineCount={5}
          isSyncing={false}
          onSync={mockOnSync}
        />,
      );
      
      expect(screen.getByRole('button')).toBeInTheDocument();
    });

    it('should call onSync when clicked', () => {
      render(
        <OfflineSyncStatus
          offlineCount={5}
          isSyncing={false}
          onSync={mockOnSync}
        />,
      );
      
      const syncButton = screen.getByRole('button');
      fireEvent.click(syncButton);
      
      expect(mockOnSync).toHaveBeenCalledTimes(1);
    });

    it('should not show sync button when syncing', () => {
      render(
        <OfflineSyncStatus
          offlineCount={5}
          isSyncing={true}
          onSync={mockOnSync}
        />,
      );
      
      const syncButton = screen.queryByRole('button');
      expect(syncButton).not.toBeInTheDocument();
    });
  });

  describe('Visual Styling', () => {
    it('should have info background color', () => {
      const { container } = render(
        <OfflineSyncStatus
          offlineCount={5}
          isSyncing={false}
          onSync={mockOnSync}
        />,
      );
      
      // Check for the main container with background color
      const statusBar = container.firstChild as HTMLElement;
      expect(statusBar).toHaveStyle({ backgroundColor: '#e3f2fd' });
    });

    it('should show rotating icon when syncing', () => {
      const { container } = render(
        <OfflineSyncStatus
          offlineCount={5}
          isSyncing={true}
          onSync={mockOnSync}
        />,
      );
      
      const rotatingIcon = container.querySelector('[style*="animation"]');
      expect(rotatingIcon).toBeInTheDocument();
    });
  });
});
