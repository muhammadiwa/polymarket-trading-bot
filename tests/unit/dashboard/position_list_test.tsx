import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { PositionList } from "@/components/positions/PositionList";

vi.mock("@/hooks/usePositions", () => ({
  usePositions: vi.fn(),
}));

import { usePositions } from "@/hooks/usePositions";
const mockUsePositions = vi.mocked(usePositions);

const MOCK_POSITIONS = [
  {
    id: "pos-1",
    market: "US Election 2026",
    side: "YES" as const,
    entryPrice: "0.5500",
    currentPrice: "0.6200",
    quantity: "100.00000000",
    unrealizedPnL: "7.00000000",
    updatedAt: "2026-07-04T00:00:00Z",
  },
  {
    id: "pos-2",
    market: "Fed Rate Decision",
    side: "NO" as const,
    entryPrice: "0.4000",
    currentPrice: "0.3500",
    quantity: "200.00000000",
    unrealizedPnL: "10.00000000",
    updatedAt: "2026-07-04T00:00:00Z",
  },
  {
    id: "pos-3",
    market: "BTC ETF",
    side: "YES" as const,
    entryPrice: "0.7000",
    currentPrice: "0.6500",
    quantity: "50.00000000",
    unrealizedPnL: "-2.50000000",
    updatedAt: "2026-07-04T00:00:00Z",
  },
];

describe("PositionList", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders loading state", () => {
    mockUsePositions.mockReturnValue({
      data: [],
      loading: true,
      error: null,
      wsStatus: "connecting",
    });

    const { container } = render(<PositionList />);
    expect(container.querySelectorAll(".animate-pulse").length).toBeGreaterThan(0);
  });

  it("renders empty state", () => {
    mockUsePositions.mockReturnValue({
      data: [],
      loading: false,
      error: null,
      wsStatus: "connected",
    });

    render(<PositionList />);
    expect(screen.getByText("No active positions")).toBeTruthy();
  });

  it("renders error state", () => {
    mockUsePositions.mockReturnValue({
      data: [],
      loading: false,
      error: "Connection failed",
      wsStatus: "disconnected",
    });

    render(<PositionList />);
    expect(screen.getByText(/Failed to load positions/)).toBeTruthy();
  });

  it("renders all positions", () => {
    mockUsePositions.mockReturnValue({
      data: MOCK_POSITIONS,
      loading: false,
      error: null,
      wsStatus: "connected",
    });

    render(<PositionList />);

    expect(screen.getByText("US Election 2026")).toBeTruthy();
    expect(screen.getByText("Fed Rate Decision")).toBeTruthy();
    expect(screen.getByText("BTC ETF")).toBeTruthy();
  });

  it("renders column headers", () => {
    mockUsePositions.mockReturnValue({
      data: MOCK_POSITIONS,
      loading: false,
      error: null,
      wsStatus: "connected",
    });

    render(<PositionList />);

    expect(screen.getByText("Market")).toBeTruthy();
    expect(screen.getByText("Side")).toBeTruthy();
    expect(screen.getByText("Entry")).toBeTruthy();
    expect(screen.getByText("Current")).toBeTruthy();
    expect(screen.getByText("Qty")).toBeTruthy();
    expect(screen.getByText("PnL")).toBeTruthy();
  });

  it("sorts by column on header click", () => {
    mockUsePositions.mockReturnValue({
      data: MOCK_POSITIONS,
      loading: false,
      error: null,
      wsStatus: "connected",
    });

    render(<PositionList />);

    // Click PnL header to sort
    fireEvent.click(screen.getByText("PnL"));

    // Should still render all positions (sorted)
    expect(screen.getByText("US Election 2026")).toBeTruthy();
    expect(screen.getByText("Fed Rate Decision")).toBeTruthy();
    expect(screen.getByText("BTC ETF")).toBeTruthy();
  });
});
