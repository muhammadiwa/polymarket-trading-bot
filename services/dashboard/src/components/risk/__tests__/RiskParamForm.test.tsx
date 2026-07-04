import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RiskParamForm } from "@/components/risk/RiskParamForm";

vi.mock("@/lib/api", () => ({
  updateRiskParameters: vi.fn().mockResolvedValue({ status: "parameters_updated" }),
}));

describe("RiskParamForm", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders all input fields", () => {
    render(<RiskParamForm />);

    expect(screen.getByLabelText("Daily Loss Limit (%)")).toBeInTheDocument();
    expect(screen.getByLabelText("Max Position per Market (%)")).toBeInTheDocument();
    expect(screen.getByLabelText("Max Position per Strategy (%)")).toBeInTheDocument();
    expect(screen.getByText("Save Parameters")).toBeInTheDocument();
  });

  it("validates percentage range", async () => {
    const user = userEvent.setup();
    render(<RiskParamForm />);

    const input = screen.getByLabelText("Daily Loss Limit (%)");
    await user.type(input, "25");
    await user.click(screen.getByText("Save Parameters"));

    await waitFor(() => {
      expect(screen.getByText("Must be between 1% and 20%")).toBeInTheDocument();
    });
  });

  it("shows error for no changes", async () => {
    const user = userEvent.setup();
    render(<RiskParamForm />);

    await user.click(screen.getByText("Save Parameters"));

    await waitFor(() => {
      expect(screen.getByText("No changes to save")).toBeInTheDocument();
    });
  });

  it("calls API on valid submission", async () => {
    const { updateRiskParameters } = await import("@/lib/api");
    const user = userEvent.setup();
    const onSuccess = vi.fn();
    render(<RiskParamForm onSuccess={onSuccess} />);

    const input = screen.getByLabelText("Daily Loss Limit (%)");
    await user.type(input, "5");
    await user.click(screen.getByText("Save Parameters"));

    await waitFor(() => {
      expect(updateRiskParameters).toHaveBeenCalledWith({ dailyLossLimit: "5" });
    });

    await waitFor(() => {
      expect(screen.getByText("Parameters updated successfully")).toBeInTheDocument();
    });
  });

  it("shows error on API failure", async () => {
    const { updateRiskParameters } = await import("@/lib/api");
    vi.mocked(updateRiskParameters).mockRejectedValueOnce(new Error("Server error"));

    const user = userEvent.setup();
    render(<RiskParamForm />);

    const input = screen.getByLabelText("Daily Loss Limit (%)");
    await user.type(input, "5");
    await user.click(screen.getByText("Save Parameters"));

    await waitFor(() => {
      expect(screen.getByText("Server error")).toBeInTheDocument();
    });
  });
});
