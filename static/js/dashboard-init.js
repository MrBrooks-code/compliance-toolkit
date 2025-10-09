/**
 * Dashboard Initialization Script
 * Handles authentication check and token refresh for dashboard pages
 */

// Global auth state
let autoRefreshInterval = null;

/**
 * Initialize dashboard authentication
 */
async function initDashboard() {
    // Check if user is authenticated (JWT or session)
    const isJWTAuth = window.authClient && window.authClient.isAuthenticated();
    const hasSessionCookie = document.cookie.split('; ').some(row => row.startsWith('session_user='));

    if (!isJWTAuth && !hasSessionCookie) {
        // Not authenticated, redirect to login
        window.location.href = '/login';
        return;
    }

    // If using JWT, start auto-refresh
    if (isJWTAuth) {
        console.log('JWT authentication detected, starting auto-refresh');

        // Start auto-refresh every 5 minutes
        autoRefreshInterval = window.authClient.startAutoRefresh(300);

        // Update user info in UI
        updateUserInfo();
    }
}

/**
 * Update user info in dashboard UI
 */
async function updateUserInfo() {
    try {
        const user = window.authClient.getUserInfo();
        if (user) {
            // Update user display elements if they exist
            const userNameElements = document.querySelectorAll('[data-user-name]');
            userNameElements.forEach(el => {
                el.textContent = user.username || 'User';
            });

            const userRoleElements = document.querySelectorAll('[data-user-role]');
            userRoleElements.forEach(el => {
                el.textContent = user.role || '';
            });
        }
    } catch (error) {
        console.error('Failed to update user info:', error);
    }
}

/**
 * Enhanced fetch function that works with both JWT and session auth
 */
async function dashboardFetch(url, options = {}) {
    // If JWT is available, use the JWT client
    if (window.authClient && window.authClient.isAuthenticated()) {
        return window.authClient.fetch(url, options);
    }

    // Otherwise, use regular fetch with credentials for session cookies
    return fetch(url, {
        ...options,
        credentials: 'same-origin'
    });
}

/**
 * Logout handler
 */
async function logout(logoutAll = false) {
    try {
        // If using JWT, call JWT logout
        if (window.authClient && window.authClient.isAuthenticated()) {
            await window.authClient.logout(logoutAll);

            // Stop auto-refresh
            if (autoRefreshInterval) {
                window.authClient.stopAutoRefresh(autoRefreshInterval);
                autoRefreshInterval = null;
            }
        } else {
            // Session-based logout
            await fetch('/api/v1/auth/logout', {
                method: 'POST',
                credentials: 'same-origin'
            });
        }

        // Redirect to login
        window.location.href = '/login';
    } catch (error) {
        console.error('Logout error:', error);
        // Redirect to login anyway
        window.location.href = '/login';
    }
}

/**
 * Get user info display
 */
function getUserDisplayName() {
    // Try JWT first
    if (window.authClient && window.authClient.isAuthenticated()) {
        const user = window.authClient.getUserInfo();
        return user?.username || 'User';
    }

    // Try session cookie
    const sessionCookie = document.cookie.split('; ').find(row => row.startsWith('session_user='));
    if (sessionCookie) {
        return sessionCookie.split('=')[1] || 'User';
    }

    return 'User';
}

// Initialize dashboard when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initDashboard);
} else {
    initDashboard();
}

// Export for global use
if (typeof window !== 'undefined') {
    window.dashboardFetch = dashboardFetch;
    window.logout = logout;
    window.getUserDisplayName = getUserDisplayName;
    window.updateUserInfo = updateUserInfo;
}
