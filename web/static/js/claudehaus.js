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
            case 'notification':
                handleNotification(msg);
                break;
        }
    }

    function handleEvent(msg) {
        htmx.trigger(document.body, 'refresh');
    }

    function handleApprovalRequest(msg) {
        htmx.trigger('#sessions', 'refresh');
        htmx.trigger(document.body, 'refresh');
    }

    function handleApprovalResolved(msg) {
        htmx.trigger(document.body, 'refresh');
    }

    function parseMultiChoicePrompts() {
        // Find all approval prompts and check for multi-choice options
        const prompts = document.querySelectorAll('.approval-prompt');
        prompts.forEach(promptEl => {
            const promptText = promptEl.textContent;
            const choices = parseChoices(promptText);
            if (choices.length > 0) {
                updateApprovalButtons(promptEl.closest('.approval-card'), choices);
            }
        });
    }

    function parseChoices(promptText) {
        // Parse choice lines like "[Y] Yes", "[N] No", "[1] Option 1"
        const choices = [];
        const lines = promptText.split('\n');
        const choicePattern = /\[([^\]]+)\]\s*(.+)/;

        lines.forEach(line => {
            const match = line.match(choicePattern);
            if (match) {
                const key = match[1].trim();
                const label = match[2].trim();
                // Skip if it looks like a timestamp [HH:MM:SS] or boolean true/false
                if (!/^\d{2}:\d{2}:\d{2}$/.test(key) && key.toLowerCase() !== 'true' && key.toLowerCase() !== 'false') {
                    choices.push({ key, label });
                }
            }
        });

        return choices;
    }

    function updateApprovalButtons(approvalCard, choices) {
        const actionsDiv = approvalCard.querySelector('.approval-actions');
        if (!actionsDiv) return;

        // Clear existing buttons
        actionsDiv.innerHTML = '';

        // Create buttons for each choice
        choices.forEach(choice => {
            const btn = document.createElement('button');
            btn.className = 'btn btn-choice';
            btn.textContent = `[${choice.key}] ${choice.label}`;
            btn.dataset.decision = choice.key;
            btn.onclick = () => submitDecision(approvalCard, choice.key);
            actionsDiv.appendChild(btn);
        });
    }

    function submitDecision(approvalCard, decision) {
        const approvalId = approvalCard.dataset.approvalId;
        if (!approvalId) return;

        fetch(`/api/approvals/${approvalId}`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': 'Bearer ' + (localStorage.getItem(STORAGE_KEY) || '')
            },
            body: JSON.stringify({ decision })
        }).then(() => {
            htmx.trigger(document.body, 'refresh');
        });
    }

    function handleSessionUpdate(msg) {
        htmx.trigger('#sessions', 'refresh');
    }

    function handleNotification(msg) {
        // Only show notification if it's for the current session
        const currentSessionId = getCurrentSessionId();
        if (currentSessionId && currentSessionId !== msg.session_id) {
            return;
        }

        const container = document.querySelector('.notification-container');
        if (!container) return;

        const toast = document.createElement('div');
        const type = msg.data.type || 'info';
        const label = type === 'idle_prompt' ? 'WAITING FOR INPUT' : type.toUpperCase();

        toast.className = `notification-toast ${type}`;
        toast.innerHTML = `
            <div class="notification-header">
                <span>${label}</span>
                <button class="notification-close" onclick="dismissNotification(this)">[X]</button>
            </div>
            <div class="notification-message">${escapeHtml(msg.data.message || '')}</div>
        `;

        container.appendChild(toast);

        // Auto-dismiss after 5 seconds (except for idle_prompt which stays until user responds)
        if (type !== 'idle_prompt') {
            setTimeout(() => {
                dismissNotification(toast.querySelector('.notification-close'));
            }, 5000);
        }
    }

    function dismissNotification(btn) {
        const toast = btn.closest('.notification-toast');
        if (toast) {
            toast.classList.add('hiding');
            setTimeout(() => {
                toast.remove();
            }, 300);
        }
    }

    function getCurrentSessionId() {
        const activeSession = document.querySelector('.session-item.active');
        return activeSession ? activeSession.dataset.sessionId : null;
    }

    function escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // Expose dismissNotification globally
    window.dismissNotification = dismissNotification;

    function toggleEventDetails(element) {
        const expanded = element.getAttribute('data-expanded') === 'true';
        const details = element.querySelector('.event-details');

        if (expanded) {
            element.setAttribute('data-expanded', 'false');
            element.classList.remove('expanded');
            details.classList.remove('show');
        } else {
            element.setAttribute('data-expanded', 'true');
            element.classList.add('expanded');
            details.classList.add('show');
        }
    }

    function handleEventClick(element, event) {
        // Remove active class from all events
        document.querySelectorAll('.event-item.active').forEach(el => {
            el.classList.remove('active');
        });
        // Add active class to clicked event
        element.classList.add('active');
        // Toggle details
        toggleEventDetails(element);
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
            case 'e':
                toggleSelectedEvent();
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

    function toggleSelectedEvent() {
        const activeEvent = document.querySelector('.event-item.active');
        if (activeEvent) {
            toggleEventDetails(activeEvent);
        }
    }

    // Expose to global scope for onclick handlers
    window.showHelp = showHelp;
    window.hideHelp = hideHelp;
    window.toggleEventDetails = toggleEventDetails;
    window.handleEventClick = handleEventClick;

    // Initialize authentication on page load
    document.addEventListener('DOMContentLoaded', function() {
        checkAuth();

        // Add token to all HTMX requests
        document.body.addEventListener('htmx:configRequest', function(evt) {
            const token = localStorage.getItem(STORAGE_KEY);
            if (token) {
                evt.detail.headers['Authorization'] = 'Bearer ' + token;
            }
        });

        // Parse multi-choice prompts after HTMX swaps
        document.body.addEventListener('htmx:afterSwap', function() {
            parseMultiChoicePrompts();
        });

        // Also parse on initial load
        parseMultiChoicePrompts();
    });
})();
