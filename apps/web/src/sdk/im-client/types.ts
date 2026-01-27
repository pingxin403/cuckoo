/**
 * IM Client SDK - Type Definitions
 * 
 * This file contains all TypeScript type definitions for the IM Client SDK.
 */

// ============================================================================
// Message Types
// ============================================================================

export type MessageType = 
  | 'auth_response'
  | 'send_msg'
  | 'message'
  | 'ack'
  | 'heartbeat'
  | 'heartbeat_response'
  | 'read_receipt'
  | 'error';

export type RecipientType = 'user' | 'group';

export type MessageStatus = 
  | 'pending'
  | 'sent'
  | 'delivered'
  | 'received'
  | 'read'
  | 'failed';

// ============================================================================
// WebSocket Messages
// ============================================================================

export interface BaseMessage {
  type: MessageType;
  timestamp: number;
}

export interface AuthResponseMessage extends BaseMessage {
  type: 'auth_response';
  success: boolean;
  user_id: string;
  device_id: string;
}

export interface SendMessageRequest extends BaseMessage {
  type: 'send_msg';
  msg_id: string;
  recipient_id: string;
  recipient_type: RecipientType;
  content: string;
}

export interface IncomingMessage extends BaseMessage {
  type: 'message';
  msg_id: string;
  sender_id: string;
  recipient_id: string;
  recipient_type: RecipientType;
  content: string;
  sequence_number: number;
}

export interface AckMessage extends BaseMessage {
  type: 'ack';
  msg_id: string;
  status: MessageStatus;
  sequence_number?: number;
}

export interface HeartbeatMessage extends BaseMessage {
  type: 'heartbeat';
}

export interface HeartbeatResponseMessage extends BaseMessage {
  type: 'heartbeat_response';
}

export interface ReadReceiptMessage extends BaseMessage {
  type: 'read_receipt';
  msg_id: string;
  reader_id: string;
  read_at: number;
}

export interface ErrorMessage extends BaseMessage {
  type: 'error';
  code: string;
  message: string;
  details?: unknown;
}

export type WebSocketMessage = 
  | AuthResponseMessage
  | SendMessageRequest
  | IncomingMessage
  | AckMessage
  | HeartbeatMessage
  | HeartbeatResponseMessage
  | ReadReceiptMessage
  | ErrorMessage;

// ============================================================================
// Client Configuration
// ============================================================================

export interface IMClientConfig {
  /** WebSocket gateway URL (e.g., 'wss://gateway.example.com:8080/ws') */
  gatewayUrl: string;
  
  /** JWT token for authentication */
  token: string;
  
  /** Heartbeat interval in milliseconds (default: 30000) */
  heartbeatInterval?: number;
  
  /** Connection timeout in milliseconds (default: 10000) */
  connectionTimeout?: number;
  
  /** Reconnection settings */
  reconnect?: {
    /** Enable automatic reconnection (default: true) */
    enabled: boolean;
    
    /** Maximum number of reconnection attempts (default: 5) */
    maxAttempts: number;
    
    /** Initial delay between reconnection attempts in ms (default: 1000) */
    initialDelay: number;
    
    /** Maximum delay between reconnection attempts in ms (default: 30000) */
    maxDelay: number;
    
    /** Backoff multiplier (default: 2) */
    backoffMultiplier: number;
  };
  
  /** Message deduplication settings */
  deduplication?: {
    /** Enable client-side deduplication (default: true) */
    enabled: boolean;
    
    /** Storage type for deduplication (default: 'memory') */
    storageType: 'memory' | 'indexeddb' | 'localstorage';
    
    /** TTL for deduplication entries in milliseconds (default: 7 days) */
    ttl: number;
  };
  
  /** Enable debug logging (default: false) */
  debug?: boolean;
}

// ============================================================================
// Client State
// ============================================================================

export type ConnectionState = 
  | 'disconnected'
  | 'connecting'
  | 'connected'
  | 'reconnecting'
  | 'disconnecting'
  | 'failed';

export interface ClientState {
  connectionState: ConnectionState;
  userId?: string;
  deviceId?: string;
  lastHeartbeat?: number;
  reconnectAttempts: number;
}

// ============================================================================
// Event Handlers
// ============================================================================

export interface IMClientEvents {
  /** Fired when connection is established and authenticated */
  onConnected: (userId: string, deviceId: string) => void;
  
  /** Fired when connection is closed */
  onDisconnected: (code: number, reason: string) => void;
  
  /** Fired when an error occurs */
  onError: (error: Error) => void;
  
  /** Fired when a new message is received */
  onMessage: (message: IncomingMessage) => void;
  
  /** Fired when a message ACK is received */
  onAck: (ack: AckMessage) => void;
  
  /** Fired when a read receipt is received */
  onReadReceipt: (receipt: ReadReceiptMessage) => void;
  
  /** Fired when connection state changes */
  onStateChange: (state: ConnectionState) => void;
  
  /** Fired when reconnection is attempted */
  onReconnecting: (attempt: number, maxAttempts: number) => void;
}

// ============================================================================
// Message Storage (for offline support)
// ============================================================================

export interface StoredMessage {
  msg_id: string;
  recipient_id: string;
  recipient_type: RecipientType;
  content: string;
  timestamp: number;
  status: MessageStatus;
  retryCount: number;
}

// ============================================================================
// Deduplication Storage
// ============================================================================

export interface DeduplicationEntry {
  msg_id: string;
  timestamp: number;
}
