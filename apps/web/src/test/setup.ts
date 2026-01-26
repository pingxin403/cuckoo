/**
 * Test Setup
 * 
 * Global test configuration for Vitest
 */

import '@testing-library/jest-dom';

// Mock WebSocket
global.WebSocket = class MockWebSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;

  readyState = MockWebSocket.CONNECTING;
  url: string;
  onopen: ((event: Event) => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;

  constructor(url: string) {
    this.url = url;
    // Simulate async connection
    setTimeout(() => {
      this.readyState = MockWebSocket.OPEN;
      if (this.onopen) {
        this.onopen(new Event('open'));
      }
      
      // Simulate auth_response message for IMClient
      setTimeout(() => {
        if (this.onmessage) {
          const authResponse = {
            type: 'auth_response',
            success: true,
            user_id: 'user123',
            device_id: 'device456',
          };
          this.onmessage(new MessageEvent('message', {
            data: JSON.stringify(authResponse),
          }));
        }
      }, 10);
    }, 0);
  }

  send(_data: string | ArrayBuffer | Blob) {
    // Mock send - simulate ACK for sent messages
    if (typeof _data === 'string') {
      try {
        const message = JSON.parse(_data);
        if (message.type === 'message' && this.onmessage) {
          // Simulate ACK after a short delay
          setTimeout(() => {
            if (this.onmessage) {
              const ack = {
                type: 'ack',
                msg_id: message.msg_id,
                status: 'sent',
                timestamp: Date.now(),
              };
              this.onmessage(new MessageEvent('message', {
                data: JSON.stringify(ack),
              }));
            }
          }, 10);
        }
      } catch {
        // Ignore parse errors
      }
    }
  }

  close(code?: number, reason?: string) {
    this.readyState = MockWebSocket.CLOSED;
    if (this.onclose) {
      this.onclose(new CloseEvent('close', { code, reason }));
    }
  }
} as any;

// Mock IndexedDB
const indexedDB = {
  open: vi.fn(() => ({
    onsuccess: null,
    onerror: null,
    result: {
      createObjectStore: vi.fn(),
      transaction: vi.fn(() => ({
        objectStore: vi.fn(() => ({
          add: vi.fn(),
          get: vi.fn(),
          delete: vi.fn(),
          clear: vi.fn(),
        })),
      })),
    },
  })),
};

global.indexedDB = indexedDB as any;

// Mock scrollIntoView
Element.prototype.scrollIntoView = vi.fn();
