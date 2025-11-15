import { LoginRequest, LoginResponse, RefreshResponse, FirstUserRequest, UserRequest, UserResponse, UsersResponse, RolesResponse, RolePermissionsResponse } from '../providers/types.js';

export interface AuthState {
  isAuthenticated: boolean;
  user: {
    email: string;
    role: string;
  } | null;
  accessToken: string | null;
  refreshToken: string | null;
}

export class AuthService {
  public state: AuthState = {
    isAuthenticated: false,
    user: null,
    accessToken: null,
    refreshToken: null,
  };

  private listeners: Set<(state: AuthState) => void> = new Set();
  private refreshTimer: NodeJS.Timeout | null = null;
  private onAuthSuccessCallback: (() => void) | null = null;
  private isRefreshing: boolean = false;
  private refreshPromise: Promise<RefreshResponse> | null = null;

  constructor() {
    this.loadFromStorage();
    this.setupRefreshTimer();
    this.setupVisibilityChangeHandler();
  }

  // State management
  subscribe(listener: (state: AuthState) => void): () => void {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  // Get authentication status
  async getAuthStatus(): Promise<{ usersExist: boolean }> {
    try {
      if (this.provider) {
        const status = await this.provider.getAuthStatus();
        return status;
      }
      // If no provider available, return fallback to prevent infinite loading
      console.warn('Provider not available yet, using fallback auth status');
      return { usersExist: false };
    } catch (error) {
      console.warn('Failed to get auth status from provider:', error);
      // Return fallback on error to prevent infinite loading
      return { usersExist: false };
    }
  }


  // Set provider for auth status checks
  private provider: any = null;

  setProvider(provider: any) {
    this.provider = provider;
  }

  // Set callback to be called after successful authentication
  setOnAuthSuccess(callback: () => void) {
    this.onAuthSuccessCallback = callback;
  }

  private notifyListeners(): void {
    this.listeners.forEach(listener => listener(this.state));
  }

  private updateState(updates: Partial<AuthState>): void {
    this.state = { ...this.state, ...updates };
    this.saveToStorage();
    this.notifyListeners();
  }

  // Storage management
  private loadFromStorage(): void {
    try {
      const stored = localStorage.getItem('outway_auth');
      if (stored) {
        const parsed = JSON.parse(stored);
        this.state = {
          isAuthenticated: parsed.isAuthenticated || false,
          user: parsed.user || null,
          accessToken: parsed.accessToken || null,
          refreshToken: parsed.refreshToken || null,
        };

        // Check if access token is expired
        if (this.state.accessToken && this.isTokenExpired(this.state.accessToken)) {
          console.log('Access token expired, attempting refresh...');
          if (this.state.refreshToken) {
            this.refreshToken().catch(() => {
              console.log('Refresh failed, logging out...');
              this.logout();
            });
          } else {
            this.logout();
          }
        } else if (this.state.isAuthenticated) {
          // Set up refresh timer for valid tokens
          this.setupRefreshTimer();
        }
      }
    } catch (error) {
      console.error('Failed to load auth state from storage:', error);
      this.clearAuth();
    }
  }

  // Check if JWT token is expired
  private isTokenExpired(token: string): boolean {
    try {
      const payload = this.parseJWT(token);
      if (!payload || !payload.exp) return true;
      
      const now = Math.floor(Date.now() / 1000);
      return payload.exp <= now;
    } catch (error) {
      console.error('Failed to parse token for expiry check:', error);
      return true;
    }
  }

  private saveToStorage(): void {
    try {
      localStorage.setItem('outway_auth', JSON.stringify(this.state));
    } catch (error) {
      console.error('Failed to save auth state to storage:', error);
    }
  }

  private clearStorage(): void {
    try {
      localStorage.removeItem('outway_auth');
    } catch (error) {
      console.error('Failed to clear auth state from storage:', error);
    }
  }

  // Authentication methods
  async login(credentials: LoginRequest): Promise<LoginResponse> {
    const response = await fetch('/api/v1/auth/login', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(credentials),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Login failed');
    }

    const data: LoginResponse = await response.json();
    this.updateState({
      isAuthenticated: true,
      user: data.user,
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
    });

    this.setupRefreshTimer();
    
    // Call callback after successful authentication
    if (this.onAuthSuccessCallback) {
      this.onAuthSuccessCallback();
    }
    
    return data;
  }

  async createFirstUser(user: FirstUserRequest): Promise<LoginResponse> {
    const response = await fetch('/api/v1/auth/first-user', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(user),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to create first user');
    }

    const data: LoginResponse = await response.json();
    this.updateState({
      isAuthenticated: true,
      user: data.user,
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
    });

    this.setupRefreshTimer();
    
    // Call callback after successful authentication
    if (this.onAuthSuccessCallback) {
      this.onAuthSuccessCallback();
    }
    
    return data;
  }

  async refreshToken(): Promise<RefreshResponse> {
    // If already refreshing, return the existing promise
    if (this.isRefreshing && this.refreshPromise) {
      return this.refreshPromise;
    }

    if (!this.state.refreshToken) {
      throw new Error('No refresh token available');
    }

    // Set refreshing flag and create promise
    this.isRefreshing = true;
    this.refreshPromise = this.performRefresh();

    try {
      const result = await this.refreshPromise;
      return result;
    } finally {
      this.isRefreshing = false;
      this.refreshPromise = null;
    }
  }

  private async performRefresh(): Promise<RefreshResponse> {
    // Note: Refresh token expiration is checked on the server side
    // We don't need to check it here as it's not a JWT token

    const response = await fetch('/api/v1/auth/refresh', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ refresh_token: this.state.refreshToken }),
    });

    if (!response.ok) {
      console.log('Token refresh failed with status:', response.status);
      this.logout();
      throw new Error('Token refresh failed');
    }

    const data: RefreshResponse = await response.json();
    this.updateState({
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
    });

    // Setup new refresh timer with the new token
    this.setupRefreshTimer();
    return data;
  }

  logout(): void {
    this.clearAuth();
  }

  private clearAuth(): void {
    this.updateState({
      isAuthenticated: false,
      user: null,
      accessToken: null,
      refreshToken: null,
    });
    this.clearStorage();
    this.clearRefreshTimer();
    this.isRefreshing = false;
    this.refreshPromise = null;
  }

  // Token management
  getAccessToken(): string | null {
    return this.state.accessToken;
  }

  getRefreshToken(): string | null {
    return this.state.refreshToken;
  }

  isAuthenticated(): boolean {
    return this.state.isAuthenticated;
  }

  getUser(): { email: string; role: string } | null {
    return this.state.user;
  }

  // Auto-refresh setup
  private setupRefreshTimer(): void {
    this.clearRefreshTimer();
    
    if (!this.state.refreshToken || !this.state.accessToken) return;

    // Parse JWT token to get expiration time
    try {
      const tokenPayload = this.parseJWT(this.state.accessToken);
      if (!tokenPayload || !tokenPayload.exp) {
        console.warn('Invalid token payload, setting up fallback refresh timer');
        this.setupFallbackRefreshTimer();
        return;
      }

      const now = Math.floor(Date.now() / 1000);
      const exp = tokenPayload.exp;
      const timeUntilExpiry = (exp - now) * 1000; // Convert to milliseconds

      // Refresh token 2 minutes before expiry (more aggressive)
      const refreshTime = Math.max(timeUntilExpiry - 120000, 1000); // At least 1 second

      console.log(`Token expires in ${Math.floor(timeUntilExpiry / 1000)}s, will refresh in ${Math.floor(refreshTime / 1000)}s`);

      this.refreshTimer = setTimeout(async () => {
        try {
          console.log('Auto-refreshing token...');
          await this.refreshToken();
        } catch (error) {
          console.error('Auto-refresh failed:', error);
          this.logout();
        }
      }, refreshTime);
    } catch (error) {
      console.error('Failed to parse JWT token for refresh timer:', error);
      this.setupFallbackRefreshTimer();
    }
  }

  // Setup visibility change handler to check token when page becomes visible
  private setupVisibilityChangeHandler(): void {
    if (typeof document === 'undefined') return; // SSR safety

    document.addEventListener('visibilitychange', () => {
      if (document.visibilityState === 'visible') {
        // Page became visible, check if token needs refresh
        this.checkAndRefreshTokenIfNeeded();
      }
    });
  }

  // Check token expiration and refresh if needed
  private async checkAndRefreshTokenIfNeeded(): Promise<void> {
    if (!this.state.accessToken || !this.state.refreshToken) return;

    try {
      const tokenPayload = this.parseJWT(this.state.accessToken);
      if (!tokenPayload || !tokenPayload.exp) return;

      const now = Math.floor(Date.now() / 1000);
      const exp = tokenPayload.exp;
      const timeUntilExpiry = (exp - now) * 1000; // Convert to milliseconds

      // If token expires in less than 5 minutes, refresh it
      if (timeUntilExpiry < 5 * 60 * 1000) {
        console.log('Token expires soon, refreshing on visibility change...');
        await this.refreshToken();
      }
    } catch (error) {
      console.error('Failed to check token on visibility change:', error);
    }
  }

  // Fallback refresh timer for when JWT parsing fails
  private setupFallbackRefreshTimer(): void {
    // Set up a more frequent refresh timer as fallback
    this.refreshTimer = setTimeout(async () => {
      try {
        console.log('Fallback auto-refresh...');
        await this.refreshToken();
      } catch (error) {
        console.error('Fallback auto-refresh failed:', error);
        this.logout();
      }
    }, 3 * 60 * 1000); // Every 3 minutes
  }

  // Parse JWT token to extract payload
  private parseJWT(token: string): any {
    try {
      const parts = token.split('.');
      if (parts.length !== 3) return null;
      
      const payload = parts[1];
      if (!payload) return null;
      
      const decoded = atob(payload.replace(/-/g, '+').replace(/_/g, '/'));
      return JSON.parse(decoded);
    } catch (error) {
      console.error('Failed to parse JWT:', error);
      return null;
    }
  }

  private clearRefreshTimer(): void {
    if (this.refreshTimer) {
      clearTimeout(this.refreshTimer);
      this.refreshTimer = null;
    }
  }

  // User management methods
  async fetchUsers(): Promise<UsersResponse> {
    const response = await this.authenticatedFetch('/api/v1/users');
    return response.json();
  }

  async createUser(user: UserRequest): Promise<UserResponse> {
    const response = await this.authenticatedFetch('/api/v1/users', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(user),
    });
    return response.json();
  }

  async getUserByEmail(email: string): Promise<UserResponse> {
    const response = await this.authenticatedFetch(`/api/v1/users/${email}`);
    return response.json();
  }

  async updateUser(email: string, user: UserRequest): Promise<UserResponse> {
    const response = await this.authenticatedFetch(`/api/v1/users/${email}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(user),
    });
    return response.json();
  }

  async deleteUser(email: string): Promise<void> {
    await this.authenticatedFetch(`/api/v1/users/${email}`, {
      method: 'DELETE',
    });
  }

  async changePassword(email: string, newPassword: string): Promise<void> {
    await this.authenticatedFetch(`/api/v1/users/${email}/change-password`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ password: newPassword }),
    });
  }

  // Role and permissions
  async fetchRoles(): Promise<RolesResponse> {
    const response = await this.authenticatedFetch('/api/v1/users/roles');
    return response.json();
  }

  async fetchRolePermissions(role: string): Promise<RolePermissionsResponse> {
    const response = await this.authenticatedFetch(`/api/v1/users/roles/${role}/permissions`);
    return response.json();
  }

  // Helper method for authenticated requests
  async authenticatedFetch(url: string, options: RequestInit = {}): Promise<Response> {
    if (!this.state.accessToken) {
      throw new Error('Not authenticated');
    }

    const headers = {
      ...options.headers,
      'Authorization': `Bearer ${this.state.accessToken}`,
    };

    const response = await fetch(url, {
      ...options,
      headers,
    });

    if (response.status === 401) {
      // Token expired, try to refresh
      try {
        console.log('Received 401, attempting token refresh...');
        await this.refreshToken();
        // Retry with new token
        const newHeaders = {
          ...options.headers,
          'Authorization': `Bearer ${this.state.accessToken}`,
        };
        return fetch(url, {
          ...options,
          headers: newHeaders,
        });
      } catch (error) {
        console.error('Token refresh failed after 401:', error);
        this.logout();
        throw new Error('Authentication failed');
      }
    }

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Request failed' }));
      throw new Error(error.error || 'Request failed');
    }

    return response;
  }
}

// Singleton instance
export const authService = new AuthService();
