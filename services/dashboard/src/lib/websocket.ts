import type { WSMessage } from "@/types";

const WS_BASE = process.env.NEXT_PUBLIC_WS_URL ?? "wss://localhost:8080";
const RECONNECT_DELAYS = [1000, 2000, 4000, 8000, 16000];
const MAX_RECONNECT_ATTEMPTS = 10;
const POLL_INTERVAL_AFTER_MAX = 30000;

function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("jwt_token");
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
    // #3: Connect without token in URL; token sent as first frame
    ws = new WebSocket(`${WS_BASE}/ws/dashboard`);

    ws.onopen = () => {
      // #3: Send token as first frame after connection
      const token = getToken();
      if (token) {
        ws?.send(JSON.stringify({ token }));
      }
      attempt = 0;
      pollingMode = false;
      setStatus("connected");
    };

    ws.onmessage = (event) => {
      try {
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
      setStatus("disconnected");
      scheduleReconnect();
    };

    ws.onerror = (event) => {
      console.warn("[WS] WebSocket error:", event);
      ws?.close();
    };
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
