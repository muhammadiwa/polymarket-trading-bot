import type { WSMessage } from "@/types";

const WS_BASE = process.env.NEXT_PUBLIC_WS_URL ?? "wss://localhost:8080";
const RECONNECT_DELAYS = [1000, 2000, 4000, 8000, 16000];
const MAX_RECONNECT_ATTEMPTS = 10;
const POLL_INTERVAL_AFTER_MAX = 30000;

function getToken(): string | null {
  if (typeof window === "undefined") return null;
  // Read from HttpOnly cookie (set by server) instead of localStorage
  const cookies = document.cookie.split(";").map((c) => c.trim());
  const sessionCookie = cookies.find((c) => c.startsWith("pqap_session="));
  return sessionCookie?.split("=")[1] ?? null;
}

export type WSStatus = "connecting" | "connected" | "disconnected";

export interface WSClientOptions {
  onMessage: (message: WSMessage) => void;
  onStatusChange?: (status: WSStatus) => void;
}

export function createWSClient(options: WSClientOptions) {
  let ws: WebSocket | null = null;
  let attempt = 0;
  let closed = false;
  let status: WSStatus = "disconnected";
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  // #18: Track if we switched to polling mode
  let pollingMode = false;

  function setStatus(s: WSStatus) {
    status = s;
    options.onStatusChange?.(s);
  }

  function connect() {
    if (closed) return;
    if (attempt >= MAX_RECONNECT_ATTEMPTS) {
      // #18: Switch to longer polling interval instead of giving up permanently
      console.warn(`[WS] Max reconnect attempts (${MAX_RECONNECT_ATTEMPTS}) reached, switching to polling mode`);
      pollingMode = true;
      attempt = 0;
      reconnectTimer = setTimeout(connect, POLL_INTERVAL_AFTER_MAX);
      return;
    }

    setStatus("connecting");
    // #3: Close any existing connection before creating new one
    if (ws) {
      ws.onclose = null;
      ws.onerror = null;
      ws.close();
      ws = null;
    }
    // Pass token as query parameter (backend expects it there)
    const token = getToken();
    if (!token) {
      console.warn("[WS] No JWT token available, deferring connection");
      setStatus("disconnected");
      scheduleReconnect();
      return;
    }
    ws = new WebSocket(`${WS_BASE}/ws/dashboard?token=${encodeURIComponent(token)}`);

    ws.onopen = () => {
      attempt = 0;
      pollingMode = false;
      setStatus("connected");
      startHeartbeat();
    };

    ws.onmessage = (event) => {
      try {
        lastMessageTime = Date.now();
        const raw = JSON.parse(event.data as string);
        if (raw.type === "ping") {
          ws?.send("pong");
          return;
        }
        options.onMessage(raw as WSMessage);
      } catch {
        console.warn("[WS] Failed to parse message:", event.data);
      }
    };

    ws.onclose = () => {
      if (closed) return;
      stopHeartbeat();
      setStatus("disconnected");
      scheduleReconnect();
    };

    ws.onerror = () => {
      // #3: Don't call ws.close() here — browser already closes it and fires onclose
      console.warn("[WS] WebSocket error");
    };
  }

  // Client-side heartbeat to detect dead connections
  let heartbeatTimer: ReturnType<typeof setInterval> | null = null;
  let lastMessageTime = Date.now();

  function startHeartbeat() {
    if (heartbeatTimer) clearInterval(heartbeatTimer);
    lastMessageTime = Date.now();
    heartbeatTimer = setInterval(() => {
      if (Date.now() - lastMessageTime > 30000) {
        // No message received in 30s, connection might be dead
        console.warn("[WS] No messages received in 30s, reconnecting...");
        ws?.close();
      }
    }, 15000);
  }

  function stopHeartbeat() {
    if (heartbeatTimer) {
      clearInterval(heartbeatTimer);
      heartbeatTimer = null;
    }
  }

  function scheduleReconnect() {
    let delay: number;
    if (pollingMode) {
      // #18: In polling mode, use long interval and reset attempt counter
      delay = POLL_INTERVAL_AFTER_MAX;
      attempt = 0;
    } else {
      delay = RECONNECT_DELAYS[Math.min(attempt, RECONNECT_DELAYS.length - 1)];
      attempt++;
    }
    reconnectTimer = setTimeout(connect, delay);
  }

  function close() {
    closed = true;
    stopHeartbeat();
    if (reconnectTimer) {
      clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }
    ws?.close();
    setStatus("disconnected");
  }

  connect();

  return { close, getStatus: () => status };
}
