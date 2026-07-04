import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { OpportunityFeed } from "@/components/opportunities/OpportunityFeed";
import { WSProvider } from "@/lib/ws-context";

const { mockOpportunities } = vi.hoisted(() => ({
  mockOpportunities: [
    {
      id: "opp-1",
      market: "Will Trump win 2024?",
      marketSlug: "trump-2024",
      score: "0.8500",
      spread: "2.50",
      fillProbability: "0.9200",
      timestamp: "2026-07-04T12:00:00Z",
      status: "detected" as const,
      filterReason: null,
      executionLatencyMs: null,
    },
    {
      id: "opp-2",
      market: "BTC above 100k EOY",
      marketSlug: "btc-100k-eoy",
      score: "0.7200",
      spread: "1.25",
      fillProbability: "0.8500",
      timestamp: "2026-07-04T11:59:30Z",
      status: "executed" as const,
      filterReason: null,
      executionLatencyMs: 45,
    },
    {
      id: "opp-3",
      market: "Fed rate cut July",
      marketSlug: "fed-rate-cut-july",
      score: "0.3100",
      spread: "0.50",
      fillProbability: "0.4000",
      timestamp: "2026-07-04T11:59:00Z",
      status: "filtered" as const,
      filterReason: "Score below threshold",
      executionLatencyMs: null,
    },
  ],
}));

vi.mock("@/lib/api", () => ({
  fetchOpportunities: vi.fn().mockResolvedValue({
    opportunities: mockOpportunities,
    total_count: 3,
    next_cursor: null,
  }),
}));

function renderWithWS(ui: React.ReactNode) {
  return render(<WSProvider>{ui}</WSProvider>);
}

describe("OpportunityFeed", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders loading state initially", () => {
    renderWithWS(<OpportunityFeed />);
    expect(screen.getByText("Loading opportunities...")).toBeInTheDocument();
  });

  it("displays opportunities after data loads", async () => {
    renderWithWS(<OpportunityFeed />);

    await waitFor(() => {
      expect(screen.getByText("Will Trump win 2024?")).toBeInTheDocument();
    });

    expect(screen.getByText("BTC above 100k EOY")).toBeInTheDocument();
    expect(screen.getByText("Fed rate cut July")).toBeInTheDocument();
  });

  it("displays scores formatted to 4 decimal places", async () => {
    renderWithWS(<OpportunityFeed />);

    await waitFor(() => {
      expect(screen.getByText("0.8500")).toBeInTheDocument();
    });

    expect(screen.getByText("0.7200")).toBeInTheDocument();
    expect(screen.getByText("0.3100")).toBeInTheDocument();
  });

  it("displays spread values", async () => {
    renderWithWS(<OpportunityFeed />);

    await waitFor(() => {
      expect(screen.getByText("$2.50")).toBeInTheDocument();
    });

    expect(screen.getByText("$1.25")).toBeInTheDocument();
    expect(screen.getByText("$0.50")).toBeInTheDocument();
  });

  it("displays status badges with correct text", async () => {
    renderWithWS(<OpportunityFeed />);

    await waitFor(() => {
      const detectedBadges = screen.getAllByText("Detected");
      expect(detectedBadges.length).toBeGreaterThanOrEqual(1);
    });

    const executedBadges = screen.getAllByText("Executed");
    expect(executedBadges.length).toBeGreaterThanOrEqual(1);

    const filteredBadges = screen.getAllByText("Filtered");
    expect(filteredBadges.length).toBeGreaterThanOrEqual(1);
  });

  it("displays filter reason for filtered opportunities", async () => {
    renderWithWS(<OpportunityFeed />);

    await waitFor(() => {
      expect(screen.getByText("Score below threshold")).toBeInTheDocument();
    });
  });

  it("displays execution latency for executed opportunities", async () => {
    renderWithWS(<OpportunityFeed />);

    await waitFor(() => {
      expect(screen.getByText("45ms")).toBeInTheDocument();
    });
  });

  it("renders filter buttons", async () => {
    renderWithWS(<OpportunityFeed />);

    await waitFor(() => {
      expect(screen.getByRole("tab", { name: "All" })).toBeInTheDocument();
    });

    expect(screen.getByRole("tab", { name: "Detected" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Executed" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Filtered" })).toBeInTheDocument();
  });

  it("renders the section heading", async () => {
    renderWithWS(<OpportunityFeed />);

    await waitFor(() => {
      expect(screen.getByText("Opportunity Feed")).toBeInTheDocument();
    });
  });

  it("shows empty state when no opportunities", async () => {
    const api = await import("@/lib/api");
    vi.mocked(api.fetchOpportunities).mockResolvedValueOnce({
      opportunities: [],
      total_count: 0,
      next_cursor: null,
    });

    renderWithWS(<OpportunityFeed />);

    await waitFor(() => {
      expect(screen.getByText("No opportunities found")).toBeInTheDocument();
    });
  });
});
