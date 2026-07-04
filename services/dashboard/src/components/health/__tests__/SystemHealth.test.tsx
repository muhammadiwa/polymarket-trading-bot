import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { SystemHealth } from "@/components/health/SystemHealth";
import { WSProvider } from "@/lib/ws-context";

const { mockHealthData } = vi.hoisted(() => ({
  mockHealthData: {
    scanner: {
      name: "Scanner",
      status: "up",
      wsConnected: true,
      cpuPercent: 45.2,
      memoryMB: 512,
      errorRate: 0.3,
      lastHeartbeat: "2026-07-04T12:00:00Z",
    },
    arbEngine: {
      name: "Arb Engine",
      status: "up",
      wsConnected: false,
      cpuPercent: 72.5,
      memoryMB: 256,
      errorRate: 1.2,
      lastHeartbeat: "2026-07-04T12:00:00Z",
    },
    executionEngine: {
      name: "Execution Engine",
      status: "degraded",
      wsConnected: false,
      cpuPercent: 85.0,
      memoryMB: 384,
      errorRate: 6.5,
      lastHeartbeat: "2026-07-04T12:00:00Z",
    },
    riskManager: {
      name: "Risk Manager",
      status: "up",
      wsConnected: false,
      cpuPercent: 20.1,
      memoryMB: 128,
      errorRate: 0.0,
      lastHeartbeat: "2026-07-04T12:00:00Z",
    },
    positionManager: {
      name: "Position Manager",
      status: "down",
      wsConnected: false,
      cpuPercent: 0,
      memoryMB: 0,
      errorRate: 0,
      lastHeartbeat: "",
    },
    overall: "degraded",
    lastUpdated: "2026-07-04T12:00:00Z",
  },
}));

vi.mock("@/lib/api", () => ({
  fetchSystemHealth: vi.fn().mockResolvedValue(mockHealthData),
}));

function renderWithWS(ui: React.ReactNode) {
  return render(<WSProvider>{ui}</WSProvider>);
}

describe("SystemHealth", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders loading state initially", () => {
    renderWithWS(<SystemHealth />);
    expect(screen.getByLabelText("Loading system health")).toBeInTheDocument();
  });

  it("displays all service health cards after data loads", async () => {
    renderWithWS(<SystemHealth />);

    await waitFor(() => {
      expect(screen.getByText("Scanner")).toBeInTheDocument();
    });

    expect(screen.getByText("Arb Engine")).toBeInTheDocument();
    expect(screen.getByText("Execution Engine")).toBeInTheDocument();
    expect(screen.getByText("Risk Manager")).toBeInTheDocument();
    expect(screen.getByText("Position Manager")).toBeInTheDocument();
  });

  it("shows overall status badge", async () => {
    renderWithWS(<SystemHealth />);

    await waitFor(() => {
      expect(screen.getByText("Degraded Performance")).toBeInTheDocument();
    });
  });

  it("displays CPU percentages", async () => {
    renderWithWS(<SystemHealth />);

    await waitFor(() => {
      expect(screen.getByText("45.2%")).toBeInTheDocument();
    });

    expect(screen.getByText("72.5%")).toBeInTheDocument();
    expect(screen.getByText("85.0%")).toBeInTheDocument();
  });

  it("displays memory values", async () => {
    renderWithWS(<SystemHealth />);

    await waitFor(() => {
      expect(screen.getByText("512 MB")).toBeInTheDocument();
    });

    expect(screen.getByText("256 MB")).toBeInTheDocument();
    expect(screen.getByText("384 MB")).toBeInTheDocument();
  });

  it("displays error rates", async () => {
    renderWithWS(<SystemHealth />);

    await waitFor(() => {
      expect(screen.getByText("0.3/min")).toBeInTheDocument();
    });

    expect(screen.getByText("1.2/min")).toBeInTheDocument();
    expect(screen.getByText("6.5/min")).toBeInTheDocument();
  });

  it("shows service status indicators", async () => {
    renderWithWS(<SystemHealth />);

    await waitFor(() => {
      const upStatuses = screen.getAllByText("UP");
      expect(upStatuses.length).toBeGreaterThanOrEqual(1);
    });

    expect(screen.getByText("DEGRADED")).toBeInTheDocument();
    expect(screen.getByText("DOWN")).toBeInTheDocument();
  });

  it("shows WebSocket connection status for scanner", async () => {
    renderWithWS(<SystemHealth />);

    await waitFor(() => {
      expect(screen.getByText("Connected")).toBeInTheDocument();
    });
  });

  it("shows offline WS indicator when disconnected", async () => {
    renderWithWS(<SystemHealth />);

    await waitFor(() => {
      expect(screen.getByText("Offline")).toBeInTheDocument();
    });
  });
});
