/**
 * OfflineSyncStatus Component
 * 
 * Displays offline message sync status and count.
 */

interface OfflineSyncStatusProps {
  offlineCount: number;
  isSyncing: boolean;
  onSync?: () => void;
  style?: React.CSSProperties;
}

export function OfflineSyncStatus({
  offlineCount,
  isSyncing,
  onSync,
  style,
}: OfflineSyncStatusProps) {
  if (offlineCount === 0 && !isSyncing) {
    return null;
  }

  return (
    <div style={{
      padding: '12px 16px',
      backgroundColor: '#e3f2fd',
      borderBottom: '1px solid #90caf9',
      display: 'flex',
      justifyContent: 'space-between',
      alignItems: 'center',
      fontSize: '14px',
      ...style,
    }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
        {isSyncing ? (
          <>
            <div style={{
              width: '16px',
              height: '16px',
              border: '2px solid #2196F3',
              borderTopColor: 'transparent',
              borderRadius: '50%',
              animation: 'spin 1s linear infinite',
            }} />
            <span>Syncing offline messages...</span>
          </>
        ) : (
          <>
            <span style={{ 
              backgroundColor: '#2196F3',
              color: '#fff',
              borderRadius: '12px',
              padding: '2px 8px',
              fontSize: '12px',
              fontWeight: 'bold',
            }}>
              {offlineCount}
            </span>
            <span>offline message{offlineCount !== 1 ? 's' : ''}</span>
          </>
        )}
      </div>

      {onSync && !isSyncing && offlineCount > 0 && (
        <button
          onClick={onSync}
          style={{
            padding: '4px 12px',
            backgroundColor: '#2196F3',
            color: '#fff',
            border: 'none',
            borderRadius: '4px',
            cursor: 'pointer',
            fontSize: '12px',
          }}
        >
          Sync Now
        </button>
      )}

      <style>{`
        @keyframes spin {
          to { transform: rotate(360deg); }
        }
      `}</style>
    </div>
  );
}
