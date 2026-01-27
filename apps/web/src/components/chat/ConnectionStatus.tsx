/**
 * ConnectionStatus Component
 * 
 * Displays connection status with visual indicator and reconnection info.
 */

interface ConnectionStatusProps {
  isConnected: boolean;
  isConnecting: boolean;
  userId?: string;
  deviceId?: string;
  error?: Error | null;
  reconnectInfo?: { attempt: number; maxAttempts: number } | null;
  style?: React.CSSProperties;
}

export function ConnectionStatus({
  isConnected,
  isConnecting,
  userId,
  deviceId,
  error,
  reconnectInfo,
  style,
}: ConnectionStatusProps) {
  const getStatusColor = () => {
    if (isConnected) {
return '#4CAF50';
}
    if (isConnecting) {
return '#FFC107';
}
    return '#f44336';
  };

  const getStatusText = () => {
    if (isConnected) {
return 'Connected';
}
    if (isConnecting) {
return 'Connecting...';
}
    return 'Disconnected';
  };

  return (
    <div style={{ 
      padding: '16px', 
      borderBottom: '1px solid #ddd',
      backgroundColor: '#f5f5f5',
      ...style,
    }}>
      <h3 style={{ margin: '0 0 8px 0' }}>IM Chat</h3>
      
      {/* Connection Status */}
      <div style={{ 
        display: 'flex', 
        alignItems: 'center', 
        gap: '8px',
        fontSize: '14px',
      }}>
        <div style={{
          width: '8px',
          height: '8px',
          borderRadius: '50%',
          backgroundColor: getStatusColor(),
        }} />
        <span>{getStatusText()}</span>
        
        {reconnectInfo && !isConnected && (
          <span style={{ color: '#FFC107', marginLeft: '8px' }}>
            Reconnecting ({reconnectInfo.attempt}/{reconnectInfo.maxAttempts})
          </span>
        )}
      </div>

      {/* User Info */}
      {userId && (
        <div style={{ fontSize: '12px', color: '#666', marginTop: '4px' }}>
          User ID: {userId} | Device ID: {deviceId}
        </div>
      )}

      {/* Error Display */}
      {error && !isConnected && (
        <div style={{ 
          marginTop: '8px',
          padding: '8px',
          backgroundColor: '#ffebee',
          color: '#c62828',
          borderRadius: '4px',
          fontSize: '12px',
        }}>
          Error: {error.message}
        </div>
      )}
    </div>
  );
}
