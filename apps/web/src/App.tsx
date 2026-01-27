import { useState } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { HelloForm } from './components/HelloForm';
import { TodoForm } from './components/TodoForm';
import { TodoList } from './components/TodoList';
import { Chat } from './components/chat/Chat';
import { AuthPage } from './pages/Auth';
import { UserPage } from './pages/User';
import { authService } from './services/authService';
import './App.css';

// Create a client
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
});

type TabType = 'services' | 'chat' | 'auth' | 'users';

function App() {
  const [activeTab, setActiveTab] = useState<TabType>('services');
  const [authToken, setAuthToken] = useState<string | null>(authService.getToken());
  const [authUserId, setAuthUserId] = useState<string | null>(authService.getUserId());

  const handleAuthSuccess = (userId: string, token: string) => {
    setAuthToken(token);
    setAuthUserId(userId);
    setActiveTab('chat'); // Redirect to chat after successful login
  };

  return (
    <QueryClientProvider client={queryClient}>
      <div style={{ maxWidth: '1200px', margin: '0 auto', padding: '20px' }}>
        <h1 style={{ textAlign: 'center', marginBottom: '20px' }}>
          Monorepo Services Demo
        </h1>

        {/* Tab Navigation */}
        <div style={{ 
          display: 'flex', 
          gap: '8px', 
          marginBottom: '40px',
          justifyContent: 'center',
          flexWrap: 'wrap',
        }}>
          <button
            onClick={() => setActiveTab('services')}
            style={{
              padding: '10px 20px',
              backgroundColor: activeTab === 'services' ? '#2196F3' : '#f5f5f5',
              color: activeTab === 'services' ? '#fff' : '#000',
              border: 'none',
              borderRadius: '4px',
              cursor: 'pointer',
              fontSize: '14px',
              fontWeight: activeTab === 'services' ? 'bold' : 'normal',
            }}
          >
            Services
          </button>
          <button
            onClick={() => setActiveTab('auth')}
            style={{
              padding: '10px 20px',
              backgroundColor: activeTab === 'auth' ? '#2196F3' : '#f5f5f5',
              color: activeTab === 'auth' ? '#fff' : '#000',
              border: 'none',
              borderRadius: '4px',
              cursor: 'pointer',
              fontSize: '14px',
              fontWeight: activeTab === 'auth' ? 'bold' : 'normal',
            }}
          >
            Auth
          </button>
          <button
            onClick={() => setActiveTab('users')}
            style={{
              padding: '10px 20px',
              backgroundColor: activeTab === 'users' ? '#2196F3' : '#f5f5f5',
              color: activeTab === 'users' ? '#fff' : '#000',
              border: 'none',
              borderRadius: '4px',
              cursor: 'pointer',
              fontSize: '14px',
              fontWeight: activeTab === 'users' ? 'bold' : 'normal',
            }}
          >
            Users
          </button>
          <button
            onClick={() => setActiveTab('chat')}
            style={{
              padding: '10px 20px',
              backgroundColor: activeTab === 'chat' ? '#2196F3' : '#f5f5f5',
              color: activeTab === 'chat' ? '#fff' : '#000',
              border: 'none',
              borderRadius: '4px',
              cursor: 'pointer',
              fontSize: '14px',
              fontWeight: activeTab === 'chat' ? 'bold' : 'normal',
            }}
          >
            IM Chat
          </button>
        </div>

        {/* Services Tab */}
        {activeTab === 'services' && (
          <>
            <div
              style={{
                display: 'grid',
                gridTemplateColumns: '1fr 1fr',
                gap: '40px',
                marginBottom: '40px',
              }}
            >
              <div
                style={{
                  border: '1px solid #ddd',
                  borderRadius: '8px',
                  padding: '20px',
                }}
              >
                <HelloForm />
              </div>

              <div
                style={{
                  border: '1px solid #ddd',
                  borderRadius: '8px',
                  padding: '20px',
                }}
              >
                <TodoForm />
              </div>
            </div>

            <div
              style={{
                border: '1px solid #ddd',
                borderRadius: '8px',
                padding: '20px',
              }}
            >
              <TodoList />
            </div>
          </>
        )}

        {/* Auth Tab */}
        {activeTab === 'auth' && (
          <AuthPage onAuthSuccess={handleAuthSuccess} />
        )}

        {/* Users Tab */}
        {activeTab === 'users' && (
          <UserPage />
        )}

        {/* Chat Tab */}
        {activeTab === 'chat' && (
          <div style={{ maxWidth: '800px', margin: '0 auto' }}>
            {!authToken || !authUserId ? (
              <div style={{
                padding: '24px',
                backgroundColor: '#fff3cd',
                border: '1px solid #ffc107',
                borderRadius: '4px',
                textAlign: 'center',
              }}>
                <p style={{ marginBottom: '16px' }}>
                  Please login first to use the chat feature.
                </p>
                <button
                  onClick={() => setActiveTab('auth')}
                  style={{
                    padding: '10px 20px',
                    backgroundColor: '#2196F3',
                    color: '#fff',
                    border: 'none',
                    borderRadius: '4px',
                    cursor: 'pointer',
                    fontSize: '14px',
                    fontWeight: 'bold',
                  }}
                >
                  Go to Login
                </button>
              </div>
            ) : (
              <>
                <div style={{
                  marginBottom: '16px',
                  padding: '12px',
                  backgroundColor: '#d4edda',
                  border: '1px solid #c3e6cb',
                  borderRadius: '4px',
                  fontSize: '14px',
                }}>
                  <strong>Demo Instructions:</strong>
                  <ul style={{ margin: '8px 0 0 0', paddingLeft: '20px' }}>
                    <li>Using authentication token from login</li>
                    <li>Ensure IM Gateway service is running at <code>ws://localhost:8080/ws</code></li>
                    <li>Open this page in multiple browser tabs to test messaging</li>
                  </ul>
                </div>
                
                <Chat token={authToken} userId={authUserId} />
              </>
            )}
          </div>
        )}
      </div>
    </QueryClientProvider>
  );
}

export default App;
