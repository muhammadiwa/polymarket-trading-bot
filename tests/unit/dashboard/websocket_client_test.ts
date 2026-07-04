import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { createWSClient } from "@/lib/websocket";

// Mock WebSocket
class MockWebSocket {
  static instances: MockWebSocket[] = [];
  onopen: (() => void) | null = null;
  onclose: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onerror: (() => void) | null = null;
  readyState = 0;
  url: string;

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
    // Simulate async connect
    setTimeout(() => {
      this.readyState = 1;
      this.onopen?.();
    }, 10);
  }

  close() {
    this.readyState = 3;
    this.onclose?.();
  }

  send(_data: string) {}
}

describe("createWSClient", () => {
  let originalWebSocket: typeof globalThis.WebSocket;

  beforeEach(() => {
    originalWebSocket = globalThis.WebSocket;
    globalThis.WebSocket = MockWebSocket as unknown as typeof globalThis.WebSocket;
    MockWebSocket.instances = [];
    vi.stubGlobal("localStorage", {
      getItem: vi.fn((key: string) => (key === "jwt_token" ? "test-jwt-token" : null)),
      setItem: vi.fn(),
      removeItem: vi.fn(),
      clear: vi.fn(),
      length: 0,
      key: vi.fn(),
    });
  });

  afterEach(() => {
    globalThis.WebSocket = originalWebSocket;
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  it("connects and reports connected status", async () => {
    const onStatusChange = vi.fn();
    const client = createWSClient({
      onMessage: vi.fn(),
      onStatusChange,
    });

    await new Promise((r) => setTimeout(r, 50));

    expect(onStatusChange).toHaveBeenCalledWith("connecting");
    expect(onStatusChange).toHaveBeenCalledWith("connected");
    client.close();
  });

  it("appends JWT token to WebSocket URL", async () => {
    const client = createWSClient({
      onMessage: vi.fn(),
      onStatusChange: vi.fn(),
    });

    await new Promise((r) => setTimeout(r, 50));

    const ws = MockWebSocket.instances[0];
    expect(ws.url).toContain("?token=test-jwt-token");
    client.close();
  });

  it("parses and dispatches messages", async () => {
    const onMessage = vi.fn();
    const client = createWSClient({
      onMessage,
      onStatusChange: vi.fn(),
    });

    await new Promise((r) => setTimeout(r, 50));

    const ws = MockWebSocket.instances[0];
    ws.onmessage?.({
      data: JSON.stringify({
        type: "portfolio_update",
        payload: {
          totalCapital: "100000",
          dailyPnL: "100",
          totalPnL: "500",
          utilizationRate: "0.5",
          lastUpdated: "2026-07-04T00:00:00Z",
        },
        timestamp: "2026-07-04T00:00:00Z",
      }),
    });

    expect(onMessage).toHaveBeenCalledWith(
      expect.objectContaining({ type: "portfolio_update" })
    );
    client.close();
  });

  it("ignores malformed messages", async () => {
    const onMessage = vi.fn();
    const client = createWSClient({
      onMessage,
      onStatusChange: vi.fn(),
    });

    await new Promise((r) => setTimeout(r, 50));

    const ws = MockWebSocket.instances[0];
    ws.onmessage?.({ data: "not-json" });

    expect(onMessage).not.toHaveBeenCalled();
    client.close();
  });

  it("reports disconnected status on close", async () => {
    const onStatusChange = vi.fn();
    const client = createWSClient({
      onMessage: vi.fn(),
      onStatusChange,
    });

    await new Promise((r) => setTimeout(r, 50));

    MockWebSocket.instances[0].close();

    expect(onStatusChange).toHaveBeenCalledWith("disconnected");
  });
});
