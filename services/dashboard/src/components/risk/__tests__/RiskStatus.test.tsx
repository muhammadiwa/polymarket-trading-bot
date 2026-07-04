import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { RiskStatus } from "@/components/risk/RiskStatus";
import { WSProvider } from "@/lib/ws-context";

vi.mock("@/lib/api", () => ({
  fetchRiskStatus: vi.fn().mockResolvedValue({
    dailyBudgetRemaining: "500.00",
    dailyBudgetTotal: "1000.00",
    dailyBudgetUsedFraction: "0.5",
    currentDrawdown: "0.03",
    drawdownThreshold: "0.10",
    winStreakCurrent: 3,
    winStreakThreshold: 5,
    circuitBreakerStatus: "open",
    circuitBreakerTrippedAt: null,
    isPaused: false,
    pausedReason: null,
    lastUpdated: "2025-01-01T00:00:00Z",
  }),
}));

function renderWithWS(ui: React.ReactNode) {
  return render(<WSProvider>{ui}</WSProvider>);
}

describe("RiskStatus", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders loading state initially", () => {
    renderWithWS(<RiskStatus />);
    expect(screen.getByLabelText("Loading risk status")).toBeInTheDocument();
  });

  it("displays risk metrics after data loads", async () => {
    renderWithWS(<RiskStatus />);

    await waitFor(() => {
      expect(screen.getByLabelText(/Daily budget remaining/)).toBeInTheDocument();
    });

    expect(screen.getByLabelText(/Current drawdown/)).toBeInTheDocument();
    expect(screen.getByLabelText(/Win streak/)).toBeInTheDocument();
    expect(screen.getByText("Risk Status")).toBeInTheDocument();
  });

  it("shows circuit breaker status", async () => {
    renderWithWS(<RiskStatus />);

    await waitFor(() => {
      expect(screen.getByText("Open")).toBeInTheDocument();
    });
  });

  it("shows live WS indicator", async () => {
    renderWithWS(<RiskStatus />);

    await waitFor(() => {
      expect(screen.getByText("Offline")).toBeInTheDocument();
    });
  });
});
