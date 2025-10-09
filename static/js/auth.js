/**
 * JWT Authentication Client Helper
 * Handles JWT token storage, refresh, and API requests
 */

class AuthClient {
    constructor(config = {}) {
        this.baseURL = config.baseURL || '';
        this.tokenRefreshThreshold = config.tokenRefreshThreshold || 60; // seconds before expiry to refresh
        this.onTokenRefreshed = config.onTokenRefreshed || null;
        this.onUnauthorized = config.onUnauthorized || null;
        this.refreshPromise = null; // Prevent multiple simultaneous refreshes
    }

    /**
     * Storage keys for tokens
     */
    static STORAGE_KEYS = {
        ACCESS_TOKEN: 'jwt_access_token',
        REFRESH_TOKEN: 'jwt_refresh_token',
        TOKEN_EXPIRY: 'jwt_token_expiry',
        USER_INFO: 'jwt_user_info'
    };

    /**
     * Login with username and password
     */
    async login(username, password) {
        try {
            const response = await fetch(`${this.baseURL}/api/auth/login`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ username, password })
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.message || 'Login failed');
            }

            const data = await response.json();
            this.storeTokens(data);

            return {
                success: true,
                user: data.user,
                expiresAt: data.expires_at
            };
        } catch (error) {
            console.error('Login error:', error);
            throw error;
        }
    }

    /**
     * Logout (revoke tokens)
     */
    async logout(logoutAll = false) {
        try {
            const refreshToken = this.getRefreshToken();

            const response = await this.fetch('/api/auth/logout', {
                method: 'POST',
                body: JSON.stringify({
                    refresh_token: refreshToken,
                    all: logoutAll
                })
            });

            // Clear local storage regardless of response
            this.clearTokens();

            return response;
        } catch (error) {
            // Clear tokens even if request fails
            this.clearTokens();
            throw error;
        }
    }

    /**
     * Store JWT tokens in localStorage
     */
    storeTokens(data) {
        localStorage.setItem(AuthClient.STORAGE_KEYS.ACCESS_TOKEN, data.access_token);
        localStorage.setItem(AuthClient.STORAGE_KEYS.REFRESH_TOKEN, data.refresh_token);
        localStorage.setItem(AuthClient.STORAGE_KEYS.TOKEN_EXPIRY, data.expires_at);

        if (data.user) {
            localStorage.setItem(AuthClient.STORAGE_KEYS.USER_INFO, JSON.stringify(data.user));
        }
    }

    /**
     * Clear all tokens from localStorage
     */
    clearTokens() {
        Object.values(AuthClient.STORAGE_KEYS).forEach(key => {
            localStorage.removeItem(key);
        });
    }

    /**
     * Get access token from localStorage
     */
    getAccessToken() {
        return localStorage.getItem(AuthClient.STORAGE_KEYS.ACCESS_TOKEN);
    }

    /**
     * Get refresh token from localStorage
     */
    getRefreshToken() {
        return localStorage.getItem(AuthClient.STORAGE_KEYS.REFRESH_TOKEN);
    }

    /**
     * Get stored user info
     */
    getUserInfo() {
        const userJson = localStorage.getItem(AuthClient.STORAGE_KEYS.USER_INFO);
        return userJson ? JSON.parse(userJson) : null;
    }

    /**
     * Check if user is authenticated
     */
    isAuthenticated() {
        return !!this.getAccessToken() && !!this.getRefreshToken();
    }

    /**
     * Check if token is expired or about to expire
     */
    isTokenExpired() {
        const expiryStr = localStorage.getItem(AuthClient.STORAGE_KEYS.TOKEN_EXPIRY);
        if (!expiryStr) return true;

        const expiry = new Date(expiryStr);
        const now = new Date();

        // Check if token is expired or will expire within threshold
        const secondsUntilExpiry = (expiry - now) / 1000;
        return secondsUntilExpiry <= this.tokenRefreshThreshold;
    }

    /**
     * Refresh access token using refresh token
     */
    async refreshToken() {
        // If already refreshing, return existing promise
        if (this.refreshPromise) {
            return this.refreshPromise;
        }

        const refreshToken = this.getRefreshToken();
        if (!refreshToken) {
            throw new Error('No refresh token available');
        }

        this.refreshPromise = (async () => {
            try {
                const response = await fetch(`${this.baseURL}/api/auth/refresh`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({ refresh_token: refreshToken })
                });

                if (!response.ok) {
                    this.clearTokens();
                    throw new Error('Token refresh failed');
                }

                const data = await response.json();
                this.storeTokens(data);

                if (this.onTokenRefreshed) {
                    this.onTokenRefreshed(data);
                }

                return data;
            } finally {
                this.refreshPromise = null;
            }
        })();

        return this.refreshPromise;
    }

    /**
     * Make authenticated API request
     */
    async fetch(url, options = {}) {
        // Auto-refresh token if needed
        if (this.isAuthenticated() && this.isTokenExpired()) {
            try {
                await this.refreshToken();
            } catch (error) {
                console.error('Token refresh failed:', error);
                if (this.onUnauthorized) {
                    this.onUnauthorized();
                }
                throw error;
            }
        }

        const token = this.getAccessToken();
        const headers = {
            ...options.headers,
            'Content-Type': 'application/json'
        };

        if (token) {
            headers['Authorization'] = `Bearer ${token}`;
        }

        const response = await fetch(`${this.baseURL}${url}`, {
            ...options,
            headers
        });

        // Handle 401 Unauthorized
        if (response.status === 401) {
            // Try to refresh token once
            if (this.getRefreshToken()) {
                try {
                    await this.refreshToken();
                    // Retry request with new token
                    headers['Authorization'] = `Bearer ${this.getAccessToken()}`;
                    return await fetch(`${this.baseURL}${url}`, {
                        ...options,
                        headers
                    });
                } catch (error) {
                    console.error('Token refresh failed on 401:', error);
                    this.clearTokens();
                    if (this.onUnauthorized) {
                        this.onUnauthorized();
                    }
                    throw new Error('Unauthorized');
                }
            } else {
                this.clearTokens();
                if (this.onUnauthorized) {
                    this.onUnauthorized();
                }
                throw new Error('Unauthorized');
            }
        }

        return response;
    }

    /**
     * Get current user info from server
     */
    async getCurrentUser() {
        try {
            const response = await this.fetch('/api/auth/me');
            if (!response.ok) {
                throw new Error('Failed to fetch user info');
            }
            const user = await response.json();
            localStorage.setItem(AuthClient.STORAGE_KEYS.USER_INFO, JSON.stringify(user));
            return user;
        } catch (error) {
            console.error('Failed to get current user:', error);
            throw error;
        }
    }

    /**
     * Start auto-refresh interval
     */
    startAutoRefresh(intervalSeconds = 300) {
        return setInterval(async () => {
            if (this.isAuthenticated() && this.isTokenExpired()) {
                try {
                    await this.refreshToken();
                    console.log('Token auto-refreshed');
                } catch (error) {
                    console.error('Auto-refresh failed:', error);
                }
            }
        }, intervalSeconds * 1000);
    }

    /**
     * Stop auto-refresh interval
     */
    stopAutoRefresh(intervalId) {
        if (intervalId) {
            clearInterval(intervalId);
        }
    }
}

// Export for use in other scripts
if (typeof window !== 'undefined') {
    window.AuthClient = AuthClient;
}

// Create global auth client instance
const authClient = new AuthClient({
    onUnauthorized: () => {
        // Redirect to login page on unauthorized
        if (window.location.pathname !== '/login') {
            window.location.href = '/login';
        }
    }
});

if (typeof window !== 'undefined') {
    window.authClient = authClient;
}
