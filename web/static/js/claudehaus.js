(function() {
    'use strict';

    let ws = null;
    let reconnectAttempts = 0;
    const maxReconnectAttempts = 10;
    let isAuthenticated = false;
    const STORAGE_KEY = 'claudehaus_token';

    // Check for stored token on page load
    function checkAuth() {
        const token = localStorage.getItem(STORAGE_KEY);
        if (token) {
            // Verify token is valid before proceeding
            verifyToken(token).then(isValid => {
                if (isValid) {
                    isAuthenticated = true;
                    connectWebSocket();
                } else {
                    // Invalid token, clear it and show login
                    localStorage.removeItem(STORAGE_KEY);
                    showLogin();
                }
            }).catch(() => {
                // Verification failed, try connecting anyway
                isAuthenticated = true;
                connectWebSocket();
            });
        } else {
            showLogin();
        }
    }

    function verifyToken(token) {
        return fetch('/api/verify-token', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({token})
        }).then(res => res.ok)
          .catch(() => false);
    }

    function showLogin(errorMsg) {
        const modal = document.getElementById('login-modal');
        const errorDiv = document.getElementById('login-error');
        const tokenInput = document.getElementById('token-input');

        if (modal) modal.classList.remove('hidden');
        if (errorMsg && errorDiv) {
            errorDiv.textContent = errorMsg;
            errorDiv.classList.remove('hidden');
        } else if (errorDiv) {
            errorDiv.classList.add('hidden');
        }
        if (tokenInput) tokenInput.focus();
    }

    function hideLogin() {
        const modal = document.getElementById('login-modal');
        if (modal) modal.classList.add('hidden');
    }

    // Exposed globally for the form submit handler
    window.submitLogin = function(event) {
        event.preventDefault();
        const tokenInput = document.getElementById('token-input');
        const token = tokenInput ? tokenInput.value.trim() : '';

        if (!token) {
            showLogin('Please enter a token');
            return false;
        }

        verifyToken(token).then(isValid => {
            if (isValid) {
                localStorage.setItem(STORAGE_KEY, token);
                isAuthenticated = true;
                hideLogin();
                connectWebSocket();
            } else {
                showLogin('Invalid token');
            }
        }).catch(() => {
            // If verification fails, try connecting anyway
            localStorage.setItem(STORAGE_KEY, token);
            isAuthenticated = true;
            hideLogin();
            connectWebSocket();
        });

        return false;
    };

    function connectWebSocket() {
        if (!isAuthenticated) {
            showLogin();
            return;
        }

        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const token = localStorage.getItem(STORAGE_KEY) || '';
        ws = new WebSocket(`${protocol}//${window.location.host}/ws?token=${token}`);

        ws.onopen = function() {
            console.log('[CLAUDEHAUS] WebSocket connected');
            reconnectAttempts = 0;
            updateStatus('CONNECTED');
            hideLogin();
        };

        ws.onclose = function(event) {
            console.log('[CLAUDEHAUS] WebSocket disconnected', event.code, event.reason);
            updateStatus('DISCONNECTED');

            // Check if closed due to unauthorized
            if (event.code === 1008 || event.code === 4001) {
                // Unauthorized - clear token and show login
                localStorage.removeItem(STORAGE_KEY);
                isAuthenticated = false;
                showLogin('Session expired. Please login again.');
                return;
            }

            scheduleReconnect();
        };

        ws.onerror = function(err) {
            console.error('[CLAUDEHAUS] WebSocket error:', err);
            // WebSocket connection failed - likely unauthorized
            // Clear token and show login on next error or close
        };

        ws.onmessage = function(event) {
            handleMessage(JSON.parse(event.data));
        };
    }

    function scheduleReconnect() {
        if (reconnectAttempts < maxReconnectAttempts && isAuthenticated) {
            reconnectAttempts++;
            const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
            console.log(`[CLAUDEHAUS] Reconnecting in ${delay}ms...`);
            setTimeout(connectWebSocket, delay);
        } else if (!isAuthenticated) {
            showLogin();
        }
    }

    function updateStatus(status) {
        const statusBar = document.querySelector('.status-bar span');
        if (statusBar) {
            statusBar.textContent = `> ${status}`;
        }
    }

    function handleMessage(msg) {
        switch (msg.type) {
            case 'event':
                handleEvent(msg);
                break;
            case 'approval_request':
                handleApprovalRequest(msg);
                break;
            case 'approval_resolved':
                handleApprovalResolved(msg);
                break;
            case 'session_update':
                handleSessionUpdate(msg);
                break;
        }
    }

    function handleEvent(msg) {
        htmx.trigger('#session-detail', 'refresh');
    }

    function handleApprovalRequest(msg) {
        htmx.trigger('#sessions', 'refresh');
        htmx.trigger('#session-detail', 'refresh');
    }

    function handleApprovalResolved(msg) {
        htmx.trigger('#session-detail', 'refresh');
    }

    function handleSessionUpdate(msg) {
        htmx.trigger('#sessions', 'refresh');
    }

    // Keyboard shortcuts
    document.addEventListener('keydown', function(e) {
        // Don't trigger shortcuts when in the login form
        if (document.getElementById('login-modal') &&
            !document.getElementById('login-modal').classList.contains('hidden')) {
            // Allow ESC to close login modal
            if (e.key === 'Escape') {
                const tokenInput = document.getElementById('token-input');
                if (tokenInput) tokenInput.value = '';
            }
            return;
        }

        if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') {
            return;
        }

        switch (e.key) {
            case '?':
                showHelp();
                break;
            case 'Escape':
                hideHelp();
                break;
            case 'y':
            case 'a':
                approveCurrentRequest();
                break;
            case 'n':
            case 'd':
                denyCurrentRequest();
                break;
            case 'j':
            case 'ArrowDown':
                navigateDown();
                break;
            case 'k':
            case 'ArrowUp':
                navigateUp();
                break;
            case '/':
                e.preventDefault();
                focusSearch();
                break;
            case '1':
            case '2':
            case '3':
            case '4':
            case '5':
            case '6':
            case '7':
            case '8':
            case '9':
                selectSession(parseInt(e.key) - 1);
                break;
        }
    });

    function showHelp() {
        const modal = document.getElementById('help-modal');
        if (modal) modal.classList.remove('hidden');
    }

    function hideHelp() {
        const modal = document.getElementById('help-modal');
        if (modal) modal.classList.add('hidden');
    }

    function approveCurrentRequest() {
        const btn = document.querySelector('.btn-allow');
        if (btn) btn.click();
    }

    function denyCurrentRequest() {
        const btn = document.querySelector('.btn-deny');
        if (btn) btn.click();
    }

    function navigateDown() {
        const items = document.querySelectorAll('.session-item');
        const active = document.querySelector('.session-item.active');
        if (!active && items.length > 0) {
            items[0].click();
        } else if (active) {
            const idx = Array.from(items).indexOf(active);
            if (idx < items.length - 1) {
                items[idx + 1].click();
            }
        }
    }

    function navigateUp() {
        const items = document.querySelectorAll('.session-item');
        const active = document.querySelector('.session-item.active');
        if (active) {
            const idx = Array.from(items).indexOf(active);
            if (idx > 0) {
                items[idx - 1].click();
            }
        }
    }

    function focusSearch() {
        const search = document.querySelector('input[type="search"]');
        if (search) search.focus();
    }

    function selectSession(idx) {
        const items = document.querySelectorAll('.session-item');
        if (items[idx]) {
            items[idx].click();
        }
    }

    // Expose to global scope for onclick handlers
    window.showHelp = showHelp;
    window.hideHelp = hideHelp;

    // Initialize authentication on page load
    document.addEventListener('DOMContentLoaded', function() {
        checkAuth();
    });
})();
