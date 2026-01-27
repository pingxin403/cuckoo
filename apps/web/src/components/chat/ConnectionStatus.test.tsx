/**
 * ConnectionStatus Component Tests
 */

import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ConnectionStatus } from './ConnectionStatus';

describe('ConnectionStatus', () => {
  describe('Connection States', () => {
    it('should show connected state', () => {
      render(
        <ConnectionStatus
          isConnected={true}
          isConnecting={false}
          userId="user123"
          deviceId="device456"
          error={null}
          reconnectInfo={null}
        />,
      );
      
      expect(screen.getByText(/Connected/i)).toBeInTheDocument();
      expect(screen.getByText(/user123/)).toBeInTheDocument();
      expect(screen.getByText(/device456/)).toBeInTheDocument();
    });

    it('should show connecting state', () => {
      render(
        <ConnectionStatus
          isConnected={false}
          isConnecting={true}
          userId={undefined}
          deviceId={undefined}
          error={null}
          reconnectInfo={null}
        />,
      );
      
      expect(screen.getByText(/Connecting/i)).toBeInTheDocument();
    });

    it('should show disconnected state', () => {
      render(
        <ConnectionStatus
          isConnected={false}
          isConnecting={false}
          userId={undefined}
          deviceId={undefined}
          error={null}
          reconnectInfo={null}
        />,
      );
      
      expect(screen.getByText(/Disconnected/i)).toBeInTheDocument();
    });
  });

  describe('Error Display', () => {
    it('should show error message', () => {
      const error = new Error('Connection failed');
      
      render(
        <ConnectionStatus
          isConnected={false}
          isConnecting={false}
          userId={undefined}
          deviceId={undefined}
          error={error}
          reconnectInfo={null}
        />,
      );
      
      expect(screen.getByText(/Connection failed/i)).toBeInTheDocument();
    });

    it('should not show error when connected', () => {
      const error = new Error('Previous error');
      
      render(
        <ConnectionStatus
          isConnected={true}
          isConnecting={false}
          userId="user123"
          deviceId="device456"
          error={error}
          reconnectInfo={null}
        />,
      );
      
      expect(screen.queryByText(/Previous error/i)).not.toBeInTheDocument();
    });
  });

  describe('Reconnection Info', () => {
    it('should show reconnection attempt', () => {
      render(
        <ConnectionStatus
          isConnected={false}
          isConnecting={true}
          userId={undefined}
          deviceId={undefined}
          error={null}
          reconnectInfo={{ attempt: 2, maxAttempts: 5 }}
        />,
      );
      
      expect(screen.getByText(/Reconnecting/i)).toBeInTheDocument();
      expect(screen.getByText(/2.*5/)).toBeInTheDocument();
    });

    it('should not show reconnection info when connected', () => {
      render(
        <ConnectionStatus
          isConnected={true}
          isConnecting={false}
          userId="user123"
          deviceId="device456"
          error={null}
          reconnectInfo={{ attempt: 2, maxAttempts: 5 }}
        />,
      );
      
      expect(screen.queryByText(/Reconnecting/i)).not.toBeInTheDocument();
    });
  });

  describe('User Info Display', () => {
    it('should show user ID when connected', () => {
      render(
        <ConnectionStatus
          isConnected={true}
          isConnecting={false}
          userId="user123"
          deviceId="device456"
          error={null}
          reconnectInfo={null}
        />,
      );
      
      expect(screen.getByText(/user123/)).toBeInTheDocument();
    });

    it('should show device ID when connected', () => {
      render(
        <ConnectionStatus
          isConnected={true}
          isConnecting={false}
          userId="user123"
          deviceId="device456"
          error={null}
          reconnectInfo={null}
        />,
      );
      
      expect(screen.getByText(/device456/)).toBeInTheDocument();
    });

    it('should not show user info when disconnected', () => {
      render(
        <ConnectionStatus
          isConnected={false}
          isConnecting={false}
          userId={undefined}
          deviceId={undefined}
          error={null}
          reconnectInfo={null}
        />,
      );
      
      expect(screen.queryByText(/user123/)).not.toBeInTheDocument();
      expect(screen.queryByText(/device456/)).not.toBeInTheDocument();
    });
  });

  describe('Visual Indicators', () => {
    it('should show green indicator when connected', () => {
      const { container } = render(
        <ConnectionStatus
          isConnected={true}
          isConnecting={false}
          userId="user123"
          deviceId="device456"
          error={null}
          reconnectInfo={null}
        />,
      );
      
      const indicator = container.querySelector('[style*="background-color: rgb(76, 175, 80)"]');
      expect(indicator).toBeInTheDocument();
    });

    it('should show yellow indicator when connecting', () => {
      const { container } = render(
        <ConnectionStatus
          isConnected={false}
          isConnecting={true}
          userId={undefined}
          deviceId={undefined}
          error={null}
          reconnectInfo={null}
        />,
      );
      
      const indicator = container.querySelector('[style*="background-color: rgb(255, 193, 7)"]');
      expect(indicator).toBeInTheDocument();
    });

    it('should show red indicator when disconnected', () => {
      const { container } = render(
        <ConnectionStatus
          isConnected={false}
          isConnecting={false}
          userId={undefined}
          deviceId={undefined}
          error={null}
          reconnectInfo={null}
        />,
      );
      
      const indicator = container.querySelector('[style*="background-color: rgb(244, 67, 54)"]');
      expect(indicator).toBeInTheDocument();
    });
  });
});
