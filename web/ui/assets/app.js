// Mortis Web UI - Main Application JavaScript
(function() {
	'use strict';

	// WebSocket connection
	let ws = null;
	let isConnected = false;
	let currentSession = 'default';
	let requestId = 0;
	let pendingRequests = new Map();

	// DOM elements
	const connectionStatus = document.getElementById('connection-status');
	const connectionText = document.getElementById('connection-text');
	const messagesContainer = document.getElementById('messages');
	const messageInput = document.getElementById('message-input');
	const sendButton = document.getElementById('send-button');

	// Initialize
	function init() {
		connectWebSocket();
		setupEventListeners();
	}

	// Connect to WebSocket
	function connectWebSocket() {
		const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
		const wsUrl = `${protocol}//${window.location.host}/ws`;

		ws = new WebSocket(wsUrl);

		ws.onopen = handleWebSocketOpen;
		ws.onmessage = handleWebSocketMessage;
		ws.onclose = handleWebSocketClose;
		ws.onerror = handleWebSocketError;
	}

	// Handle WebSocket open
	function handleWebSocketOpen() {
		isConnected = true;
		updateConnectionStatus(true);
		console.log('WebSocket connected');
	}

	// Handle WebSocket message
	function handleWebSocketMessage(event) {
		try {
			const frame = JSON.parse(event.data);

			if (frame.type === 'event') {
				handleEvent(frame.event);
			} else if (frame.type === 'res') {
				handleResponse(frame);
			}
		} catch (error) {
			console.error('Failed to parse WebSocket message:', error);
		}
	}

	// Handle WebSocket close
	function handleWebSocketClose() {
		isConnected = false;
		updateConnectionStatus(false);
		console.log('WebSocket disconnected, reconnecting in 3s...');
		setTimeout(connectWebSocket, 3000);
	}

	// Handle WebSocket error
	function handleWebSocketError(error) {
		console.error('WebSocket error:', error);
	}

	// Update connection status
	function updateConnectionStatus(connected) {
		if (connected) {
			connectionStatus.classList.remove('disconnected');
			connectionStatus.classList.add('connected');
			connectionText.textContent = 'Connected';
			sendButton.disabled = false;
		} else {
			connectionStatus.classList.remove('connected');
			connectionStatus.classList.add('disconnected');
			connectionText.textContent = 'Disconnected';
			sendButton.disabled = true;
		}
	}

	// Handle event frame
	function handleEvent(eventFrame) {
		const payload = eventFrame.payload || {};

		switch (eventFrame.event) {
		case 'gateway.start':
			console.log('Gateway started');
			break;

		case 'chat.thinking':
			showThinking(payload.sessionId, payload.messageId);
			break;

		case 'chat.textDelta':
			appendTextDelta(payload.sessionId, payload.messageId, payload.delta, payload.done);
			break;

		case 'chat.toolStart':
			console.log('Tool started:', payload.toolName);
			break;

		case 'chat.toolEnd':
			console.log('Tool ended:', payload.toolName, payload.success);
			break;

		case 'chat.done':
			hideThinking(payload.sessionId, payload.messageId);
			console.log('Chat done:', payload.finishReason, 'tokens:', payload.tokensUsed);
			break;

		case 'provider.status':
			console.log('Provider status:', payload);
			break;

		default:
			console.log('Unhandled event:', eventFrame.event, payload);
		}
	}

	// Handle response frame
	function handleResponse(responseFrame) {
		const requestId = responseFrame.id;

		if (pendingRequests.has(requestId)) {
			const { resolve } = pendingRequests.get(requestId);
			pendingRequests.delete(requestId);

			if (responseFrame.ok) {
				resolve({ ok: true, payload: responseFrame.payload });
			} else {
				resolve({ ok: false, error: responseFrame.error });
			}
		}
	}

	// Send RPC request
	function sendRequest(method, params = {}) {
		return new Promise((resolve) => {
			const id = `req_${Date.now()}_${requestId++}`;

			pendingRequests.set(id, { resolve });

			const frame = {
				type: 'req',
				id: id,
				method: method,
				params: params
			};

			ws.send(JSON.stringify(frame));

			// Timeout after 30 seconds
			setTimeout(() => {
				if (pendingRequests.has(id)) {
					pendingRequests.delete(id);
					resolve({ ok: false, error: { code: 'TIMEOUT', message: 'Request timeout' } });
				}
			}, 30000);
		});
	}

	// Send chat message
	async function sendChatMessage(message) {
		if (!message.trim()) {
			return;
		}

		// Clear input
		messageInput.value = '';
		messageInput.style.height = 'auto';

		// Add user message to UI
		addMessageToUI('user', message);

		try {
			const response = await sendRequest('chat.send', {
				message: message,
				sessionId: currentSession
			});

			if (!response.ok) {
				console.error('Chat send failed:', response.error);
				addMessageToUI('system', `Error: ${response.error.message}`);
			}
		} catch (error) {
			console.error('Chat send error:', error);
			addMessageToUI('system', `Error: ${error.message}`);
		}
	}

	// Show thinking indicator
	function showThinking(sessionId, messageId) {
		let thinkingEl = document.getElementById(`thinking-${messageId}`);
		if (!thinkingEl) {
			thinkingEl = document.createElement('div');
			thinkingEl.id = `thinking-${messageId}`;
			thinkingEl.className = 'thinking';
			thinkingEl.innerHTML = `
				<div class="thinking-dot"></div>
				<div class="thinking-dot"></div>
				<div class="thinking-dot"></div>
			`;
			messagesContainer.appendChild(thinkingEl);
			scrollToBottom();
		}
	}

	// Hide thinking indicator
	function hideThinking(sessionId, messageId) {
		const thinkingEl = document.getElementById(`thinking-${messageId}`);
		if (thinkingEl) {
			thinkingEl.remove();
		}
	}

	// Append text delta
	function appendTextDelta(sessionId, messageId, delta, done) {
		let contentEl = document.getElementById(`content-${messageId}`);
		if (!contentEl) {
			// Remove empty state
			const emptyState = messagesContainer.querySelector('.empty-state');
			if (emptyState) {
				emptyState.remove();
			}

			// Create message container
			const messageEl = document.createElement('div');
			messageEl.className = 'message';
			messageEl.id = `message-${messageId}`;

			messageEl.innerHTML = `
				<div class="message-header">
					<span class="message-role assistant">Assistant</span>
				</div>
				<div class="message-content" id="content-${messageId}"></div>
			`;

			messagesContainer.appendChild(messageEl);
			contentEl = document.getElementById(`content-${messageId}`);
		}

		contentEl.textContent += delta;
		scrollToBottom();
	}

	// Add message to UI
	function addMessageToUI(role, content) {
		// Remove empty state
		const emptyState = messagesContainer.querySelector('.empty-state');
		if (emptyState) {
			emptyState.remove();
		}

		const messageId = `msg_${Date.now()}`;
		const messageEl = document.createElement('div');
		messageEl.className = 'message';
		messageEl.id = `message-${messageId}`;

		messageEl.innerHTML = `
			<div class="message-header">
				<span class="message-role ${role}">${role}</span>
			</div>
			<div class="message-content">${escapeHtml(content)}</div>
		`;

		messagesContainer.appendChild(messageEl);
		scrollToBottom();
	}

	// Scroll to bottom
	function scrollToBottom() {
		messagesContainer.scrollTop = messagesContainer.scrollHeight;
	}

	// Escape HTML
	function escapeHtml(text) {
		const div = document.createElement('div');
		div.textContent = text;
		return div.innerHTML;
	}

	// Setup event listeners
	function setupEventListeners() {
		// Send button
		sendButton.addEventListener('click', () => {
			const message = messageInput.value.trim();
			if (message) {
				sendChatMessage(message);
			}
		});

		// Enter key to send (Shift+Enter for newline)
		messageInput.addEventListener('keydown', (event) => {
			if (event.key === 'Enter' && !event.shiftKey) {
				event.preventDefault();
				const message = messageInput.value.trim();
				if (message) {
					sendChatMessage(message);
				}
			}
		});

		// Auto-resize textarea
		messageInput.addEventListener('input', () => {
			messageInput.style.height = 'auto';
			messageInput.style.height = Math.min(messageInput.scrollHeight, 200) + 'px';
			sendButton.disabled = !messageInput.value.trim() || !isConnected;
		});

		// Session items
		document.querySelectorAll('.sidebar-item[data-session]').forEach(item => {
			item.addEventListener('click', () => {
				document.querySelectorAll('.sidebar-item[data-session]').forEach(i => {
					i.classList.remove('active');
				});
				item.classList.add('active');
				currentSession = item.dataset.session;
				console.log('Switched to session:', currentSession);
			});
		});
	}

	// Initialize on DOM ready
	if (document.readyState === 'loading') {
		document.addEventListener('DOMContentLoaded', init);
	} else {
		init();
	}
})();
