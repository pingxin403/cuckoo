/**
 * Auth Service
 * 
 * Client for authentication operations.
 * Note: This is a mock implementation for demo purposes.
 * In production, integrate with real auth-service gRPC API.
 */

export interface LoginRequest {
  username: string;
  password: string;
}

export interface RegisterRequest {
  username: string;
  email: string;
  password: string;
}

export interface AuthResponse {
  success: boolean;
  accessToken?: string;
  refreshToken?: string;
  userId?: string;
  deviceId?: string;
  expiresAt?: number;
  errorMessage?: string;
}

class AuthService {
  private accessToken: string | null = null;
  private refreshToken: string | null = null;
  private userId: string | null = null;
  private deviceId: string | null = null;

  /**
   * Login with username and password
   */
  async login(request: LoginRequest): Promise<AuthResponse> {
    try {
      // Mock implementation - replace with real API call
      // In production: call auth-service gRPC ValidateToken endpoint
      
      // Simulate API delay
      await new Promise(resolve => setTimeout(resolve, 500));

      // Mock successful login
      const mockToken = this.generateMockToken(request.username);
      const mockRefreshToken = this.generateMockToken(request.username, true);
      const mockUserId = `user_${request.username}`;
      const mockDeviceId = `device_${Date.now()}`;

      this.accessToken = mockToken;
      this.refreshToken = mockRefreshToken;
      this.userId = mockUserId;
      this.deviceId = mockDeviceId;

      // Store in localStorage
      localStorage.setItem('auth_token', mockToken);
      localStorage.setItem('refresh_token', mockRefreshToken);
      localStorage.setItem('user_id', mockUserId);
      localStorage.setItem('device_id', mockDeviceId);

      return {
        success: true,
        accessToken: mockToken,
        refreshToken: mockRefreshToken,
        userId: mockUserId,
        deviceId: mockDeviceId,
        expiresAt: Date.now() + 3600000, // 1 hour
      };
    } catch (error) {
      return {
        success: false,
        errorMessage: error instanceof Error ? error.message : 'Login failed',
      };
    }
  }

  /**
   * Register new user
   */
  async register(_request: RegisterRequest): Promise<AuthResponse> {
    try {
      // Mock implementation - replace with real API call
      
      // Simulate API delay
      await new Promise(resolve => setTimeout(resolve, 500));

      // Mock successful registration
      return {
        success: true,
        errorMessage: 'Registration successful. Please login.',
      };
    } catch (error) {
      return {
        success: false,
        errorMessage: error instanceof Error ? error.message : 'Registration failed',
      };
    }
  }

  /**
   * Logout current user
   */
  logout(): void {
    this.accessToken = null;
    this.refreshToken = null;
    this.userId = null;
    this.deviceId = null;

    localStorage.removeItem('auth_token');
    localStorage.removeItem('refresh_token');
    localStorage.removeItem('user_id');
    localStorage.removeItem('device_id');
  }

  /**
   * Get current access token
   */
  getToken(): string | null {
    if (!this.accessToken) {
      this.accessToken = localStorage.getItem('auth_token');
    }
    return this.accessToken;
  }

  /**
   * Get current user ID
   */
  getUserId(): string | null {
    if (!this.userId) {
      this.userId = localStorage.getItem('user_id');
    }
    return this.userId;
  }

  /**
   * Get current device ID
   */
  getDeviceId(): string | null {
    if (!this.deviceId) {
      this.deviceId = localStorage.getItem('device_id');
    }
    return this.deviceId;
  }

  /**
   * Check if user is authenticated
   */
  isAuthenticated(): boolean {
    return this.getToken() !== null;
  }

  /**
   * Refresh access token
   */
  async refreshAccessToken(): Promise<AuthResponse> {
    try {
      const refreshToken = this.refreshToken || localStorage.getItem('refresh_token');
      
      if (!refreshToken) {
        return {
          success: false,
          errorMessage: 'No refresh token available',
        };
      }

      // Mock implementation - replace with real API call
      // In production: call auth-service gRPC RefreshToken endpoint
      
      await new Promise(resolve => setTimeout(resolve, 300));

      const newToken = this.generateMockToken(this.userId || 'user');
      this.accessToken = newToken;
      localStorage.setItem('auth_token', newToken);

      return {
        success: true,
        accessToken: newToken,
        expiresAt: Date.now() + 3600000,
      };
    } catch (error) {
      return {
        success: false,
        errorMessage: error instanceof Error ? error.message : 'Token refresh failed',
      };
    }
  }

  /**
   * Generate mock JWT token (for demo only)
   */
  private generateMockToken(userId: string, isRefresh = false): string {
    const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
    const payload = btoa(JSON.stringify({
      user_id: userId,
      device_id: `device_${Date.now()}`,
      exp: Date.now() + (isRefresh ? 7 * 24 * 3600000 : 3600000),
      type: isRefresh ? 'refresh' : 'access',
    }));
    return `${header}.${payload}.mock_signature`;
  }
}

// Export singleton instance
export const authService = new AuthService();
