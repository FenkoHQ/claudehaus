(function() {
    'use strict';

    let ws = null;
    let reconnectAttempts = 0;
    const maxReconnectAttempts = 10;
    let isAuthenticated = false;
    const STORAGE_KEY = 'claudehaus_token';

    // ================================================================
    // THEME
    // ================================================================
    function getTheme() {
        const stored = localStorage.getItem('fenko-theme');
        if (stored) return stored;
        return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    }

    function applyTheme(theme) {
        document.documentElement.setAttribute('data-theme', theme);
        localStorage.setItem('fenko-theme', theme);
        const icon = document.getElementById('theme-icon');
        if (icon) {
            icon.textContent = theme === 'dark' ? '\u263E' : '\u2600';
        }
    }

    window.toggleTheme = function() {
        const current = getTheme();
        applyTheme(current === 'dark' ? 'light' : 'dark');
    };

    // ================================================================
    // SETUP TABS
    // ================================================================
    window.switchSetupTab = function(tab, btn) {
        document.querySelectorAll('.setup-tab').forEach(t => t.classList.remove('active'));
        document.querySelectorAll('.setup-tab-content').forEach(c => c.classList.remove('active'));
        btn.classList.add('active');
        const content = document.getElementById('setup-' + tab);
        if (content) content.classList.add('active');
    };

    // ================================================================
    // AUTH
    // ================================================================
    function checkAuth() {
        const token = localStorage.getItem(STORAGE_KEY);
        if (token) {
            verifyToken(token).then(isValid => {
                if (isValid) {
                    isAuthenticated = true;
                    connectWebSocket();
                } else {
                    localStorage.removeItem(STORAGE_KEY);
                    showLogin();
                }
            }).catch(() => {
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
            localStorage.setItem(STORAGE_KEY, token);
            isAuthenticated = true;
            hideLogin();
            connectWebSocket();
        });

        return false;
    };

    // ================================================================
    // WEBSOCKET
    // ================================================================
    function connectWebSocket() {
        if (!isAuthenticated) {
            showLogin();
            return;
        }

        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const token = localStorage.getItem(STORAGE_KEY) || '';
        ws = new WebSocket(`${protocol}//${window.location.host}/ws?token=${token}`);

        ws.onopen = function() {
            reconnectAttempts = 0;
            updateStatus('CONNECTED');
            hideLogin();
        };

        ws.onclose = function(event) {
            updateStatus('DISCONNECTED');

            if (event.code === 1008 || event.code === 4001) {
                localStorage.removeItem(STORAGE_KEY);
                isAuthenticated = false;
                showLogin('Session expired. Please login again.');
                return;
            }

            scheduleReconnect();
        };

        ws.onerror = function() {};

        ws.onmessage = function(event) {
            handleMessage(JSON.parse(event.data));
        };
    }

    function scheduleReconnect() {
        if (reconnectAttempts < maxReconnectAttempts && isAuthenticated) {
            reconnectAttempts++;
            const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
            setTimeout(connectWebSocket, delay);
        } else if (!isAuthenticated) {
            showLogin();
        }
    }

    function updateStatus(status) {
        const statusBar = document.querySelector('.status-bar span');
        if (statusBar) {
            statusBar.textContent = '> ' + status;
        }
    }

    // ================================================================
    // MESSAGE HANDLING
    // ================================================================
    function handleMessage(msg) {
        switch (msg.type) {
            case 'event':
                htmx.trigger(document.body, 'refresh');
                break;
            case 'approval_request':
                htmx.trigger('#sessions', 'refresh');
                htmx.trigger(document.body, 'refresh');
                break;
            case 'approval_resolved':
                htmx.trigger(document.body, 'refresh');
                break;
            case 'session_update':
                htmx.trigger('#sessions', 'refresh');
                break;
            case 'notification':
                handleNotification(msg);
                break;
        }
    }

    // ================================================================
    // MULTI-CHOICE APPROVALS
    // ================================================================
    function parseMultiChoicePrompts() {
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
        const choices = [];
        const lines = promptText.split('\n');
        const choicePattern = /\[([^\]]+)\]\s*(.+)/;

        lines.forEach(line => {
            const match = line.match(choicePattern);
            if (match) {
                const key = match[1].trim();
                const label = match[2].trim();
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

        actionsDiv.innerHTML = '';

        choices.forEach(choice => {
            const btn = document.createElement('button');
            btn.className = 'btn btn-choice';
            btn.textContent = '[' + choice.key + '] ' + choice.label;
            btn.dataset.decision = choice.key;
            btn.onclick = () => submitDecision(approvalCard, choice.key);
            actionsDiv.appendChild(btn);
        });
    }

    function submitDecision(approvalCard, decision) {
        const approvalId = approvalCard.dataset.approvalId;
        if (!approvalId) return;

        fetch('/api/approvals/' + approvalId, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': 'Bearer ' + (localStorage.getItem(STORAGE_KEY) || '')
            },
            body: JSON.stringify({ decision: decision })
        }).then(() => {
            htmx.trigger(document.body, 'refresh');
        });
    }

    // ================================================================
    // NOTIFICATIONS
    // ================================================================
    function handleNotification(msg) {
        const currentSessionId = getCurrentSessionId();
        if (currentSessionId && currentSessionId !== msg.session_id) {
            return;
        }

        const container = document.querySelector('.notification-container');
        if (!container) return;

        const toast = document.createElement('div');
        const type = msg.data.type || 'info';
        const label = type === 'idle_prompt' ? 'Waiting for input' : type.replace(/_/g, ' ');

        toast.className = 'notification-toast ' + type;
        toast.innerHTML =
            '<div class="notification-header">' +
                '<span>' + escapeHtml(label) + '</span>' +
                '<button class="notification-close" onclick="dismissNotification(this)">&times;</button>' +
            '</div>' +
            '<div class="notification-message">' + escapeHtml(msg.data.message || '') + '</div>';

        container.appendChild(toast);

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
            setTimeout(() => { toast.remove(); }, 200);
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

    window.dismissNotification = dismissNotification;

    // ================================================================
    // EVENT DETAILS
    // ================================================================
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
        document.querySelectorAll('.event-item.active').forEach(el => {
            el.classList.remove('active');
        });
        element.classList.add('active');
        toggleEventDetails(element);
    }

    // ================================================================
    // KEYBOARD SHORTCUTS
    // ================================================================
    document.addEventListener('keydown', function(e) {
        if (document.getElementById('login-modal') &&
            !document.getElementById('login-modal').classList.contains('hidden')) {
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
            case '1': case '2': case '3': case '4': case '5':
            case '6': case '7': case '8': case '9':
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
            if (idx < items.length - 1) items[idx + 1].click();
        }
    }

    function navigateUp() {
        const items = document.querySelectorAll('.session-item');
        const active = document.querySelector('.session-item.active');
        if (active) {
            const idx = Array.from(items).indexOf(active);
            if (idx > 0) items[idx - 1].click();
        }
    }

    function focusSearch() {
        const search = document.querySelector('input[type="search"]');
        if (search) search.focus();
    }

    function selectSession(idx) {
        const items = document.querySelectorAll('.session-item');
        if (items[idx]) items[idx].click();
    }

    function toggleSelectedEvent() {
        const activeEvent = document.querySelector('.event-item.active');
        if (activeEvent) toggleEventDetails(activeEvent);
    }

    // Expose to global scope
    window.showHelp = showHelp;
    window.hideHelp = hideHelp;
    window.toggleEventDetails = toggleEventDetails;
    window.handleEventClick = handleEventClick;

    // ================================================================
    // INIT
    // ================================================================
    document.addEventListener('DOMContentLoaded', function() {
        // Apply theme icon
        applyTheme(getTheme());

        checkAuth();

        document.body.addEventListener('htmx:configRequest', function(evt) {
            const token = localStorage.getItem(STORAGE_KEY);
            if (token) {
                evt.detail.headers['Authorization'] = 'Bearer ' + token;
            }
        });

        document.body.addEventListener('htmx:afterSwap', function() {
            parseMultiChoicePrompts();
        });

        parseMultiChoicePrompts();
    });
})();
