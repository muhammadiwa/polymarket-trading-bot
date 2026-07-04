import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { createWSClient } from "@/lib/websocket";

class MockWebSocket {
  static instances: MockWebSocket[] = [];
  onopen: (() => void) | null = null;
  onclose: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onerror: (() => void) | null = null;
  readyState = 0;
  url: string;
  sentData: string[] = [];

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
    setTimeout(() => {
      this.readyState = 1;
      this.onopen?.();
    }, 10);
  }

  close() {
    this.readyState = 3;
    this.onclose?.();
  }

  send(data: string) {
    this.sentData.push(data);
  }
}

describe("WebSocket risk_update integration", () => {
  let originalWebSocket: typeof globalThis.WebSocket;

  beforeEach(() => {
    // #19: Use fake timers for deterministic tests
    vi.useFakeTimers();
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

  it("sends JWT token as first frame instead of URL parameter", async () => {
    const client = createWSClient({
      onMessage: vi.fn(),
      onStatusChange: vi.fn(),
    });

    // #19: Advance timers instead of real setTimeout
    await vi.advanceTimersByTimeAsync(20);

    const ws = MockWebSocket.instances[0];
    // #3: Token should NOT be in URL
    expect(ws.url).not.toContain("token=");
    // #3: Token should be sent as first frame
    expect(ws.sentData.length).toBeGreaterThanOrEqual(1);
    const firstFrame = JSON.parse(ws.sentData[0]);
    expect(firstFrame.token).toBe("test-jwt-token");
    client.close();
  });

  it("dispatches risk_update messages to onMessage callback", async () => {
    const onMessage = vi.fn();
    const client = createWSClient({
      onMessage,
      onStatusChange: vi.fn(),
    });

    await vi.advanceTimersByTimeAsync(20);

    const ws = MockWebSocket.instances[0];
    const riskPayload = {
      type: "risk_update",
      payload: {
        dailyBudgetRemaining: "750.00",
        dailyBudgetTotal: "1000.00",
        dailyBudgetUsedFraction: "0.25",
        currentDrawdown: "0.02",
        drawdownThreshold: "0.10",
        winStreakCurrent: 4,
        winStreakThreshold: 5,
        circuitBreakerStatus: "open",
        circuitBreakerTrippedAt: null,
        isPaused: false,
        pausedReason: null,
        lastUpdated: "2026-07-04T12:00:00Z",
      },
      timestamp: "2026-07-04T12:00:00Z",
    };

    ws.onmessage?.({ data: JSON.stringify(riskPayload) });

    expect(onMessage).toHaveBeenCalledWith(
      expect.objectContaining({
        type: "risk_update",
        payload: expect.objectContaining({
          winStreakCurrent: 4,
          dailyBudgetUsedFraction: "0.25",
        }),
      }),
    );
    client.close();
  });

  it("responds to server ping with pong", async () => {
    const client = createWSClient({
      onMessage: vi.fn(),
      onStatusChange: vi.fn(),
    });

    await vi.advanceTimersByTimeAsync(20);

    const ws = MockWebSocket.instances[0];
    const sendSpy = vi.spyOn(ws, "send");

    ws.onmessage?.({ data: JSON.stringify({ type: "ping" }) });

    expect(sendSpy).toHaveBeenCalledWith("pong");
    client.close();
  });
});
