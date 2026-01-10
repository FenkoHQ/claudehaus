(function() {
    'use strict';

    let ws = null;
    let reconnectAttempts = 0;
    const maxReconnectAttempts = 10;

    function connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const token = localStorage.getItem('claudehaus_token') || '';
        ws = new WebSocket(`${protocol}//${window.location.host}/ws?token=${token}`);

        ws.onopen = function() {
            console.log('[CLAUDEHAUS] WebSocket connected');
            reconnectAttempts = 0;
            updateStatus('CONNECTED');
        };

        ws.onclose = function() {
            console.log('[CLAUDEHAUS] WebSocket disconnected');
            updateStatus('DISCONNECTED');
            scheduleReconnect();
        };

        ws.onerror = function(err) {
            console.error('[CLAUDEHAUS] WebSocket error:', err);
        };

        ws.onmessage = function(event) {
            handleMessage(JSON.parse(event.data));
        };
    }

    function scheduleReconnect() {
        if (reconnectAttempts < maxReconnectAttempts) {
            reconnectAttempts++;
            const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
            console.log(`[CLAUDEHAUS] Reconnecting in ${delay}ms...`);
            setTimeout(connectWebSocket, delay);
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

    // Initialize WebSocket on page load
    document.addEventListener('DOMContentLoaded', function() {
        connectWebSocket();
    });
})();
