# IM Client SDK

A TypeScript/JavaScript SDK for connecting to the IM Gateway Service with WebSocket support, automatic reconnection, and message deduplication.

## Features

- ✅ **WebSocket-based real-time messaging**
- ✅ **Automatic reconnection** with exponential backoff
- ✅ **Heartbeat handling** (30s interval)
- ✅ **Client-side message deduplication** (7-day TTL)
- ✅ **Multiple storage backends** (Memory, LocalStorage, IndexedDB)
- ✅ **TypeScript type safety**
- ✅ **Event-driven architecture**
- ✅ **Read receipts support**
- ✅ **Multi-device support**

## Installation

The SDK is included in the `apps/web` project. Import it directly:

```typescript
import { IMClient } from '@/sdk/im-client';
```

## Quick Start

### 1. Create a Client Instance

```typescript
import { IMClient } from '@/sdk/im-client';

const client = new IMClient({
  gatewayUrl: 'wss://gateway.example.com:8080/ws',
  token: 'your-jwt-token-here',
  debug: true // Enable debug logging
});
```

### 2. Register Event Handlers

```typescript
// Connection events
client.on('onConnected', (userId, deviceId) => {
  console.log('Connected:', userId, deviceId);
});

client.on('onDisconnected', (code, reason) => {
  console.log('Disconnected:', code, reason);
});

client.on('onError', (error) => {
  console.error('Error:', error);
});

// Message events
client.on('onMessage', (message) => {
  console.log('New message:', message);
  // Display message in UI
});

client.on('onAck', (ack) => {
  console.log('Message ACK:', ack.msg_id, ack.status);
  // Update message status in UI
});

client.on('onReadReceipt', (receipt) => {
  console.log('Read receipt:', receipt.msg_id, receipt.reader_id);
  // Update message read status in UI
});

// State change events
client.on('onStateChange', (state) => {
  console.log('State changed:', state);
  // Update connection indicator in UI
});

client.on('onReconnecting', (attempt, maxAttempts) => {
  console.log(`Reconnecting... (${attempt}/${maxAttempts})`);
  // Show reconnection indicator in UI
});
```

### 3. Connect to Gateway

```typescript
try {
  await client.connect();
  console.log('Successfully connected!');
} catch (error) {
  console.error('Connection failed:', error);
}
```

### 4. Send Messages

```typescript
// Send a private message
const msgId = await client.sendMessage('user_123', 'Hello!', 'user');

// Send a group message
const groupMsgId = await client.sendMessage('group_456', 'Hello everyone!', 'group');
```

### 5. Send Read Receipts

```typescript
client.on('onMessage', (message) => {
  // Display message
  console.log('New message:', message.content);
  
  // Send read receipt
  client.sendReadReceipt(message.msg_id);
});
```

### 6. Disconnect

```typescript
client.disconnect();
```

## Configuration

### Full Configuration Options

```typescript
const client = new IMClient({
  // Required
  gatewayUrl: 'wss://gateway.example.com:8080/ws',
  token: 'your-jwt-token',
  
  // Optional
  heartbeatInterval: 30000, // 30 seconds (default)
  connectionTimeout: 10000, // 10 seconds (default)
  
  // Reconnection settings
  reconnect: {
    enabled: true, // Enable automatic reconnection (default)
    maxAttempts: 5, // Maximum reconnection attempts (default)
    initialDelay: 1000, // Initial delay in ms (default)
    maxDelay: 30000, // Maximum delay in ms (default)
    backoffMultiplier: 2 // Exponential backoff multiplier (default)
  },
  
  // Deduplication settings
  deduplication: {
    enabled: true, // Enable client-side deduplication (default)
    storageType: 'memory', // 'memory' | 'localstorage' | 'indexeddb' (default: 'memory')
    ttl: 7 * 24 * 60 * 60 * 1000 // 7 days (default)
  },
  
  // Debug logging
  debug: false // Disable debug logging (default)
});
```

### Storage Types

#### Memory Storage (Default)
- Fast, no persistence
- Data lost on page reload
- Best for: Testing, temporary sessions

```typescript
deduplication: {
  storageType: 'memory'
}
```

#### LocalStorage
- Persists across page reloads
- Limited to ~5-10MB
- Best for: Small to medium message volumes

```typescript
deduplication: {
  storageType: 'localstorage'
}
```

#### IndexedDB
- Persists across page reloads
- Large storage capacity (50MB+)
- Best for: High message volumes, production use

```typescript
deduplication: {
  storageType: 'indexeddb'
}
```

## API Reference

### IMClient Class

#### Constructor

```typescript
new IMClient(config: IMClientConfig)
```

#### Methods

##### `connect(): Promise<void>`
Connect to the IM Gateway. Returns a promise that resolves when authentication is successful.

```typescript
await client.connect();
```

##### `disconnect(): void`
Disconnect from the IM Gateway and stop all timers.

```typescript
client.disconnect();
```

##### `sendMessage(recipientId: string, content: string, recipientType?: RecipientType): Promise<string>`
Send a message to a user or group. Returns the message ID.

```typescript
const msgId = await client.sendMessage('user_123', 'Hello!', 'user');
```

##### `sendReadReceipt(msgId: string): void`
Send a read receipt for a message.

```typescript
client.sendReadReceipt('msg_user123_1706180400000_abc');
```

##### `getState(): ConnectionState`
Get the current connection state.

```typescript
const state = client.getState(); // 'connected' | 'disconnected' | 'connecting' | etc.
```

##### `getUserId(): string | undefined`
Get the authenticated user ID (available after connection).

```typescript
const userId = client.getUserId();
```

##### `getDeviceId(): string | undefined`
Get the device ID (available after connection).

```typescript
const deviceId = client.getDeviceId();
```

##### `on<K extends keyof IMClientEvents>(event: K, handler: IMClientEvents[K]): void`
Register an event handler.

```typescript
client.on('onMessage', (message) => {
  console.log('New message:', message);
});
```

##### `off<K extends keyof IMClientEvents>(event: K): void`
Unregister an event handler.

```typescript
client.off('onMessage');
```

##### `clearDeduplication(): Promise<void>`
Clear the deduplication storage.

```typescript
await client.clearDeduplication();
```

##### `destroy(): void`
Destroy the client and clean up all resources.

```typescript
client.destroy();
```

### Events

#### `onConnected(userId: string, deviceId: string): void`
Fired when connection is established and authenticated.

#### `onDisconnected(code: number, reason: string): void`
Fired when connection is closed.

#### `onError(error: Error): void`
Fired when an error occurs.

#### `onMessage(message: IncomingMessage): void`
Fired when a new message is received.

#### `onAck(ack: AckMessage): void`
Fired when a message ACK is received.

#### `onReadReceipt(receipt: ReadReceiptMessage): void`
Fired when a read receipt is received.

#### `onStateChange(state: ConnectionState): void`
Fired when connection state changes.

#### `onReconnecting(attempt: number, maxAttempts: number): void`
Fired when reconnection is attempted.

## Message Types

### IncomingMessage

```typescript
interface IncomingMessage {
  type: 'message';
  msg_id: string;
  sender_id: string;
  recipient_id: string;
  recipient_type: 'user' | 'group';
  content: string;
  sequence_number: number;
  timestamp: number;
}
```

### AckMessage

```typescript
interface AckMessage {
  type: 'ack';
  msg_id: string;
  status: 'pending' | 'sent' | 'delivered' | 'received' | 'read' | 'failed';
  sequence_number?: number;
  timestamp: number;
}
```

### ReadReceiptMessage

```typescript
interface ReadReceiptMessage {
  type: 'read_receipt';
  msg_id: string;
  reader_id: string;
  read_at: number;
  timestamp: number;
}
```

## Connection States

- `disconnected` - Not connected
- `connecting` - Establishing connection
- `connected` - Connected and authenticated
- `reconnecting` - Attempting to reconnect
- `disconnecting` - Closing connection
- `failed` - Connection failed (max reconnection attempts reached)

## Error Handling

### Connection Errors

```typescript
try {
  await client.connect();
} catch (error) {
  if (error.message === 'Connection timeout') {
    // Handle timeout
  } else if (error.message === 'Authentication failed') {
    // Handle auth failure
  } else {
    // Handle other errors
  }
}
```

### Message Send Errors

```typescript
try {
  await client.sendMessage('user_123', 'Hello!');
} catch (error) {
  if (error.message === 'Not connected to IM Gateway') {
    // Reconnect first
  } else if (error.message === 'Duplicate message') {
    // Message already sent
  }
}
```

### Error Events

```typescript
client.on('onError', (error) => {
  console.error('Client error:', error);
  // Show error notification to user
});
```

## Best Practices

### 1. Handle Reconnection

```typescript
client.on('onReconnecting', (attempt, maxAttempts) => {
  // Show reconnection indicator
  showNotification(`Reconnecting... (${attempt}/${maxAttempts})`);
});

client.on('onConnected', () => {
  // Hide reconnection indicator
  hideNotification();
  
  // Fetch missed messages
  fetchOfflineMessages();
});
```

### 2. Persist Messages Locally

```typescript
client.on('onMessage', async (message) => {
  // Save to local database
  await saveMessageToLocalDB(message);
  
  // Display in UI
  displayMessage(message);
  
  // Send read receipt
  client.sendReadReceipt(message.msg_id);
});
```

### 3. Handle ACKs

```typescript
const pendingMessages = new Map();

async function sendMessageWithRetry(recipientId: string, content: string) {
  const msgId = await client.sendMessage(recipientId, content);
  
  // Store pending message
  pendingMessages.set(msgId, {
    recipientId,
    content,
    timestamp: Date.now()
  });
  
  // Set timeout for retry
  setTimeout(() => {
    if (pendingMessages.has(msgId)) {
      // Message not ACKed, retry
      console.log('Message not ACKed, retrying:', msgId);
      sendMessageWithRetry(recipientId, content);
    }
  }, 5000);
}

client.on('onAck', (ack) => {
  // Remove from pending
  pendingMessages.delete(ack.msg_id);
  
  // Update UI
  updateMessageStatus(ack.msg_id, ack.status);
});
```

### 4. Use IndexedDB for Production

```typescript
const client = new IMClient({
  gatewayUrl: 'wss://gateway.example.com:8080/ws',
  token: getAuthToken(),
  deduplication: {
    enabled: true,
    storageType: 'indexeddb', // Best for production
    ttl: 7 * 24 * 60 * 60 * 1000
  }
});
```

### 5. Clean Up on Unmount

```typescript
// React example
useEffect(() => {
  const client = new IMClient(config);
  client.connect();
  
  return () => {
    client.destroy(); // Clean up resources
  };
}, []);
```

## React Integration Example

```typescript
import { useEffect, useState } from 'react';
import { IMClient, IncomingMessage } from '@/sdk/im-client';

function useChatClient(token: string) {
  const [client, setClient] = useState<IMClient | null>(null);
  const [messages, setMessages] = useState<IncomingMessage[]>([]);
  const [connectionState, setConnectionState] = useState<string>('disconnected');

  useEffect(() => {
    const imClient = new IMClient({
      gatewayUrl: 'wss://gateway.example.com:8080/ws',
      token,
      debug: true
    });

    // Register event handlers
    imClient.on('onConnected', (userId, deviceId) => {
      console.log('Connected:', userId, deviceId);
    });

    imClient.on('onMessage', (message) => {
      setMessages(prev => [...prev, message]);
      imClient.sendReadReceipt(message.msg_id);
    });

    imClient.on('onStateChange', (state) => {
      setConnectionState(state);
    });

    // Connect
    imClient.connect().catch(console.error);

    setClient(imClient);

    // Cleanup
    return () => {
      imClient.destroy();
    };
  }, [token]);

  const sendMessage = async (recipientId: string, content: string) => {
    if (!client) throw new Error('Client not initialized');
    return client.sendMessage(recipientId, content);
  };

  return {
    client,
    messages,
    connectionState,
    sendMessage
  };
}

// Usage in component
function ChatComponent() {
  const { messages, connectionState, sendMessage } = useChatClient('your-token');

  return (
    <div>
      <div>Status: {connectionState}</div>
      <div>
        {messages.map(msg => (
          <div key={msg.msg_id}>{msg.content}</div>
        ))}
      </div>
      <button onClick={() => sendMessage('user_123', 'Hello!')}>
        Send Message
      </button>
    </div>
  );
}
```

## Troubleshooting

### Connection Issues

**Problem**: Connection timeout

**Solution**: Check network connectivity and gateway URL

```typescript
client.on('onError', (error) => {
  if (error.message.includes('timeout')) {
    // Increase timeout
    const newClient = new IMClient({
      ...config,
      connectionTimeout: 20000 // 20 seconds
    });
  }
});
```

### Authentication Issues

**Problem**: Authentication failed

**Solution**: Verify JWT token is valid and not expired

```typescript
// Refresh token before connecting
const freshToken = await refreshAuthToken();
const client = new IMClient({
  gatewayUrl: 'wss://gateway.example.com:8080/ws',
  token: freshToken
});
```

### Message Duplication

**Problem**: Receiving duplicate messages

**Solution**: Ensure deduplication is enabled

```typescript
const client = new IMClient({
  ...config,
  deduplication: {
    enabled: true,
    storageType: 'indexeddb', // Persists across reloads
    ttl: 7 * 24 * 60 * 60 * 1000
  }
});
```

## License

This SDK is part of the IM Chat System project.

## Support

For issues and questions, please refer to the main project documentation or contact the development team.
