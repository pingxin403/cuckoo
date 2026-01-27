/**
 * Auth Page
 * 
 * Login and registration interface.
 */

import { useState } from 'react';
import { authService, type LoginRequest, type RegisterRequest } from '@/services/authService';

interface AuthPageProps {
  onAuthSuccess?: (userId: string, token: string) => void;
}

export function AuthPage({ onAuthSuccess }: AuthPageProps) {
  const [mode, setMode] = useState<'login' | 'register'>('login');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  // Login form state
  const [loginForm, setLoginForm] = useState<LoginRequest>({
    username: '',
    password: '',
  });

  // Register form state
  const [registerForm, setRegisterForm] = useState<RegisterRequest>({
    username: '',
    email: '',
    password: '',
  });

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);
    setLoading(true);

    try {
      const response = await authService.login(loginForm);
      
      if (response.success && response.accessToken && response.userId) {
        setSuccess('Login successful!');
        onAuthSuccess?.(response.userId, response.accessToken);
      } else {
        setError(response.errorMessage || 'Login failed');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);
    setLoading(true);

    try {
      const response = await authService.register(registerForm);
      
      if (response.success) {
        setSuccess(response.errorMessage || 'Registration successful! Please login.');
        setMode('login');
        setRegisterForm({ username: '', email: '', password: '' });
      } else {
        setError(response.errorMessage || 'Registration failed');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Registration failed');
    } finally {
      setLoading(false);
    }
  };

  const handleLogout = () => {
    authService.logout();
    setSuccess('Logged out successfully');
    setLoginForm({ username: '', password: '' });
  };

  const isAuthenticated = authService.isAuthenticated();
  const currentUserId = authService.getUserId();

  return (
    <div style={{ maxWidth: '500px', margin: '0 auto' }}>
      <h2 style={{ textAlign: 'center', marginBottom: '24px' }}>
        Authentication
      </h2>

      {/* Demo Notice */}
      <div style={{
        padding: '12px',
        backgroundColor: '#fff3cd',
        border: '1px solid #ffc107',
        borderRadius: '4px',
        marginBottom: '24px',
        fontSize: '14px',
      }}>
        <strong>Demo Mode:</strong> This is a mock authentication system.
        Enter any username/password to login.
      </div>

      {/* Current Auth Status */}
      {isAuthenticated && (
        <div style={{
          padding: '16px',
          backgroundColor: '#d4edda',
          border: '1px solid #c3e6cb',
          borderRadius: '4px',
          marginBottom: '24px',
        }}>
          <div style={{ marginBottom: '8px' }}>
            <strong>Logged in as:</strong> {currentUserId}
          </div>
          <button
            onClick={handleLogout}
            style={{
              padding: '8px 16px',
              backgroundColor: '#dc3545',
              color: '#fff',
              border: 'none',
              borderRadius: '4px',
              cursor: 'pointer',
              fontSize: '14px',
            }}
          >
            Logout
          </button>
        </div>
      )}

      {/* Success/Error Messages */}
      {success && (
        <div style={{
          padding: '12px',
          backgroundColor: '#d4edda',
          color: '#155724',
          border: '1px solid #c3e6cb',
          borderRadius: '4px',
          marginBottom: '16px',
        }}>
          {success}
        </div>
      )}

      {error && (
        <div style={{
          padding: '12px',
          backgroundColor: '#f8d7da',
          color: '#721c24',
          border: '1px solid #f5c6cb',
          borderRadius: '4px',
          marginBottom: '16px',
        }}>
          {error}
        </div>
      )}

      {/* Mode Toggle */}
      {!isAuthenticated && (
        <>
          <div style={{
            display: 'flex',
            gap: '8px',
            marginBottom: '24px',
            justifyContent: 'center',
          }}>
            <button
              onClick={() => setMode('login')}
              style={{
                padding: '10px 20px',
                backgroundColor: mode === 'login' ? '#2196F3' : '#f5f5f5',
                color: mode === 'login' ? '#fff' : '#000',
                border: 'none',
                borderRadius: '4px',
                cursor: 'pointer',
                fontSize: '14px',
                fontWeight: mode === 'login' ? 'bold' : 'normal',
              }}
            >
              Login
            </button>
            <button
              onClick={() => setMode('register')}
              style={{
                padding: '10px 20px',
                backgroundColor: mode === 'register' ? '#2196F3' : '#f5f5f5',
                color: mode === 'register' ? '#fff' : '#000',
                border: 'none',
                borderRadius: '4px',
                cursor: 'pointer',
                fontSize: '14px',
                fontWeight: mode === 'register' ? 'bold' : 'normal',
              }}
            >
              Register
            </button>
          </div>

          {/* Login Form */}
          {mode === 'login' && (
            <form onSubmit={handleLogin} style={{
              border: '1px solid #ddd',
              borderRadius: '8px',
              padding: '24px',
              backgroundColor: '#fff',
            }}>
              <div style={{ marginBottom: '16px' }}>
                <label style={{ display: 'block', marginBottom: '8px', fontWeight: 'bold' }}>
                  Username
                </label>
                <input
                  type="text"
                  value={loginForm.username}
                  onChange={(e) => setLoginForm({ ...loginForm, username: e.target.value })}
                  required
                  disabled={loading}
                  style={{
                    width: '100%',
                    padding: '10px',
                    border: '1px solid #ddd',
                    borderRadius: '4px',
                    fontSize: '14px',
                  }}
                />
              </div>

              <div style={{ marginBottom: '24px' }}>
                <label style={{ display: 'block', marginBottom: '8px', fontWeight: 'bold' }}>
                  Password
                </label>
                <input
                  type="password"
                  value={loginForm.password}
                  onChange={(e) => setLoginForm({ ...loginForm, password: e.target.value })}
                  required
                  disabled={loading}
                  style={{
                    width: '100%',
                    padding: '10px',
                    border: '1px solid #ddd',
                    borderRadius: '4px',
                    fontSize: '14px',
                  }}
                />
              </div>

              <button
                type="submit"
                disabled={loading}
                style={{
                  width: '100%',
                  padding: '12px',
                  backgroundColor: loading ? '#ccc' : '#2196F3',
                  color: '#fff',
                  border: 'none',
                  borderRadius: '4px',
                  cursor: loading ? 'not-allowed' : 'pointer',
                  fontSize: '16px',
                  fontWeight: 'bold',
                }}
              >
                {loading ? 'Logging in...' : 'Login'}
              </button>
            </form>
          )}

          {/* Register Form */}
          {mode === 'register' && (
            <form onSubmit={handleRegister} style={{
              border: '1px solid #ddd',
              borderRadius: '8px',
              padding: '24px',
              backgroundColor: '#fff',
            }}>
              <div style={{ marginBottom: '16px' }}>
                <label style={{ display: 'block', marginBottom: '8px', fontWeight: 'bold' }}>
                  Username
                </label>
                <input
                  type="text"
                  value={registerForm.username}
                  onChange={(e) => setRegisterForm({ ...registerForm, username: e.target.value })}
                  required
                  disabled={loading}
                  style={{
                    width: '100%',
                    padding: '10px',
                    border: '1px solid #ddd',
                    borderRadius: '4px',
                    fontSize: '14px',
                  }}
                />
              </div>

              <div style={{ marginBottom: '16px' }}>
                <label style={{ display: 'block', marginBottom: '8px', fontWeight: 'bold' }}>
                  Email
                </label>
                <input
                  type="email"
                  value={registerForm.email}
                  onChange={(e) => setRegisterForm({ ...registerForm, email: e.target.value })}
                  required
                  disabled={loading}
                  style={{
                    width: '100%',
                    padding: '10px',
                    border: '1px solid #ddd',
                    borderRadius: '4px',
                    fontSize: '14px',
                  }}
                />
              </div>

              <div style={{ marginBottom: '24px' }}>
                <label style={{ display: 'block', marginBottom: '8px', fontWeight: 'bold' }}>
                  Password
                </label>
                <input
                  type="password"
                  value={registerForm.password}
                  onChange={(e) => setRegisterForm({ ...registerForm, password: e.target.value })}
                  required
                  disabled={loading}
                  style={{
                    width: '100%',
                    padding: '10px',
                    border: '1px solid #ddd',
                    borderRadius: '4px',
                    fontSize: '14px',
                  }}
                />
              </div>

              <button
                type="submit"
                disabled={loading}
                style={{
                  width: '100%',
                  padding: '12px',
                  backgroundColor: loading ? '#ccc' : '#4CAF50',
                  color: '#fff',
                  border: 'none',
                  borderRadius: '4px',
                  cursor: loading ? 'not-allowed' : 'pointer',
                  fontSize: '16px',
                  fontWeight: 'bold',
                }}
              >
                {loading ? 'Registering...' : 'Register'}
              </button>
            </form>
          )}
        </>
      )}
    </div>
  );
}
