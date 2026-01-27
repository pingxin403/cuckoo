/**
 * IM Client SDK
 * 
 * A TypeScript/JavaScript SDK for connecting to the IM Gateway Service.
 * 
 * Features:
 * - WebSocket-based real-time messaging
 * - Automatic reconnection with exponential backoff
 * - Heartbeat handling
 * - Client-side message deduplication
 * - Multiple storage backends (Memory, LocalStorage, IndexedDB)
 * - TypeScript type safety
 * - Event-driven architecture
 * 
 * @example
 * ```typescript
 * import { IMClient } from '@/sdk/im-client';
 * 
 * const client = new IMClient({
 *   gatewayUrl: 'wss://gateway.example.com:8080/ws',
 *   token: 'your-jwt-token',
 *   debug: true
 * });
 * 
 * client.on('onConnected', (userId, deviceId) => {
 *   console.log('Connected:', userId, deviceId);
 * });
 * 
 * client.on('onMessage', (message) => {
 *   console.log('New message:', message);
 * });
 * 
 * await client.connect();
 * 
 * const msgId = await client.sendMessage('user_123', 'Hello!');
 * ```
 */

export { IMClient } from './IMClient';

export type {
  // Configuration
  IMClientConfig,
  
  // Events
  IMClientEvents,
  
  // State
  ConnectionState,
  ClientState,
  
  // Messages
  MessageType,
  RecipientType,
  MessageStatus,
  BaseMessage,
  AuthResponseMessage,
  SendMessageRequest,
  IncomingMessage,
  AckMessage,
  HeartbeatMessage,
  HeartbeatResponseMessage,
  ReadReceiptMessage,
  ErrorMessage,
  WebSocketMessage,
  
  // Storage
  StoredMessage,
  DeduplicationEntry,
} from './types';

export type {
  DeduplicationStorage,
} from './deduplication';

export {
  MemoryDeduplicationStorage,
  LocalStorageDeduplicationStorage,
  IndexedDBDeduplicationStorage,
  createDeduplicationStorage,
} from './deduplication';
