import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QuickActions } from "@/components/risk/QuickActions";
import { WSProvider } from "@/lib/ws-context";

const mockRefresh = vi.fn().mockResolvedValue(undefined);

vi.mock("@/hooks/useRiskStatus", () => ({
  useRiskStatus: () => ({
    data: {
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
    },
    loading: false,
    error: null,
    wsStatus: "connected",
    refresh: mockRefresh,
  }),
}));

vi.mock("@/lib/api", () => ({
  triggerEmergencyStop: vi.fn().mockResolvedValue({ status: "emergency_stop_activated" }),
  fetchEmergencyStopToken: vi.fn().mockResolvedValue({ confirmationToken: "test-token-uuid" }),
  pauseTrading: vi.fn().mockResolvedValue({ status: "trading_paused" }),
  resumeTrading: vi.fn().mockResolvedValue({ status: "trading_resumed" }),
  updateRiskParameters: vi.fn().mockResolvedValue({ status: "parameters_updated" }),
}));

function renderWithWS(ui: React.ReactNode) {
  return render(<WSProvider>{ui}</WSProvider>);
}

describe("QuickActions", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders quick actions section", () => {
    renderWithWS(<QuickActions />);
    expect(screen.getByText("Quick Actions")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Emergency Stop" })).toBeInTheDocument();
    expect(screen.getByText("Pause Trading")).toBeInTheDocument();
  });

  it("shows emergency stop confirmation modal on click", async () => {
    const user = userEvent.setup();
    renderWithWS(<QuickActions />);

    await user.click(screen.getByRole("button", { name: "Emergency Stop" }));

    expect(screen.getByPlaceholderText('Type "STOP" to confirm')).toBeInTheDocument();
  });

  it("shows risk parameter form", () => {
    renderWithWS(<QuickActions />);
    expect(screen.getByLabelText("Risk Parameter Adjustment")).toBeInTheDocument();
    expect(screen.getByLabelText("Daily Loss Limit (%)")).toBeInTheDocument();
  });

  it("shows pause input when pause is clicked", async () => {
    const user = userEvent.setup();
    renderWithWS(<QuickActions />);

    await user.click(screen.getByText("Pause Trading"));

    expect(screen.getByPlaceholderText("Reason (optional)")).toBeInTheDocument();
    expect(screen.getByText("Confirm Pause")).toBeInTheDocument();
  });

  it("fetches confirmation token before emergency stop", async () => {
    const { triggerEmergencyStop, fetchEmergencyStopToken } = await import("@/lib/api");
    const user = userEvent.setup();
    renderWithWS(<QuickActions />);

    await user.click(screen.getByRole("button", { name: "Emergency Stop" }));
    const input = screen.getByPlaceholderText('Type "STOP" to confirm');
    await user.type(input, "STOP");
    await user.click(screen.getByText("Stop Trading"));

    await waitFor(() => {
      expect(fetchEmergencyStopToken).toHaveBeenCalled();
    });

    await waitFor(() => {
      expect(triggerEmergencyStop).toHaveBeenCalledWith(
        "Manual emergency stop from dashboard",
        "test-token-uuid",
      );
    });
  });
});
