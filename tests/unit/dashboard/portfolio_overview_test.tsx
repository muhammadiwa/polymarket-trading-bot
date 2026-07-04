import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { PortfolioOverview } from "@/components/portfolio/PortfolioOverview";

vi.mock("@/hooks/usePortfolio", () => ({
  usePortfolio: vi.fn(),
}));

import { usePortfolio } from "@/hooks/usePortfolio";
const mockUsePortfolio = vi.mocked(usePortfolio);

describe("PortfolioOverview", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders loading skeleton", () => {
    mockUsePortfolio.mockReturnValue({
      data: null,
      loading: true,
      error: null,
      wsStatus: "connecting",
    });

    const { container } = render(<PortfolioOverview />);
    expect(container.querySelectorAll(".animate-pulse").length).toBeGreaterThan(0);
  });

  it("renders error state", () => {
    mockUsePortfolio.mockReturnValue({
      data: null,
      loading: false,
      error: "Network error",
      wsStatus: "disconnected",
    });

    render(<PortfolioOverview />);
    expect(screen.getByText(/Failed to load portfolio/)).toBeTruthy();
    expect(screen.getByText(/Network error/)).toBeTruthy();
  });

  it("renders portfolio metrics", () => {
    mockUsePortfolio.mockReturnValue({
      data: {
        totalCapital: "100000.00000000",
        dailyPnL: "250.50000000",
        totalPnL: "5000.00000000",
        utilizationRate: "0.4500",
        lastUpdated: "2026-07-04T00:00:00Z",
      },
      loading: false,
      error: null,
      wsStatus: "connected",
    });

    render(<PortfolioOverview />);

    expect(screen.getByText("Total Capital")).toBeTruthy();
    expect(screen.getByText("Daily PnL")).toBeTruthy();
    expect(screen.getByText("Total PnL")).toBeTruthy();
    expect(screen.getByText("Capital Utilization")).toBeTruthy();
    expect(screen.getByText("Utilization")).toBeTruthy();
  });

  it("shows green color for positive PnL", () => {
    mockUsePortfolio.mockReturnValue({
      data: {
        totalCapital: "100000.00000000",
        dailyPnL: "250.50000000",
        totalPnL: "5000.00000000",
        utilizationRate: "0.4500",
        lastUpdated: "2026-07-04T00:00:00Z",
      },
      loading: false,
      error: null,
      wsStatus: "connected",
    });

    const { container } = render(<PortfolioOverview />);
    const greenElements = container.querySelectorAll(".text-\\[\\#00ff88\\]");
    expect(greenElements.length).toBeGreaterThan(0);
  });

  it("shows red color for negative PnL", () => {
    mockUsePortfolio.mockReturnValue({
      data: {
        totalCapital: "100000.00000000",
        dailyPnL: "-150.00000000",
        totalPnL: "-2000.00000000",
        utilizationRate: "0.3000",
        lastUpdated: "2026-07-04T00:00:00Z",
      },
      loading: false,
      error: null,
      wsStatus: "connected",
    });

    const { container } = render(<PortfolioOverview />);
    const redElements = container.querySelectorAll(".text-\\[\\#ff4757\\]");
    expect(redElements.length).toBeGreaterThan(0);
  });

  it("shows Live badge when connected", () => {
    mockUsePortfolio.mockReturnValue({
      data: {
        totalCapital: "100000.00000000",
        dailyPnL: "0",
        totalPnL: "0",
        utilizationRate: "0",
        lastUpdated: "2026-07-04T00:00:00Z",
      },
      loading: false,
      error: null,
      wsStatus: "connected",
    });

    render(<PortfolioOverview />);
    expect(screen.getByText("Live")).toBeTruthy();
  });
});
