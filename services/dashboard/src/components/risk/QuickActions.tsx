"use client";

import { useState, useCallback } from "react";
import { ConfirmationModal } from "@/components/ui/ConfirmationModal";
import { RiskParamForm } from "@/components/risk/RiskParamForm";
import { triggerEmergencyStop, fetchEmergencyStopToken, pauseTrading, resumeTrading } from "@/lib/api";
import { useRiskStatus } from "@/hooks/useRiskStatus";

type ActionFeedback = {
  type: "success" | "error";
  message: string;
} | null;

export function QuickActions() {
  const { data: riskData, refresh } = useRiskStatus();
  const [showStopModal, setShowStopModal] = useState(false);
  const [showResumeModal, setShowResumeModal] = useState(false);
  const [showPauseModal, setShowPauseModal] = useState(false);
  const [stopConfirmText, setStopConfirmText] = useState("");
  const [pauseReason, setPauseReason] = useState("");
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<ActionFeedback>(null);

  const clearFeedback = useCallback(() => {
    setFeedback(null);
  }, []);

  const handleEmergencyStop = useCallback(async () => {
    if (stopConfirmText !== "STOP") return;

    setActionLoading("emergency-stop");
    setFeedback(null);
    try {
      const { confirmationToken } = await fetchEmergencyStopToken();
      await triggerEmergencyStop("Manual emergency stop from dashboard", confirmationToken);
      setFeedback({ type: "success", message: "Emergency stop activated — all trading halted" });
      setShowStopModal(false);
      setStopConfirmText("");
      await refresh();
    } catch (err) {
      setFeedback({ type: "error", message: err instanceof Error ? err.message : "Failed to trigger emergency stop" });
      // #12: Keep modal open on error so user can retry
    } finally {
      setActionLoading(null);
    }
  }, [stopConfirmText, refresh]);

  const handlePause = useCallback(async () => {
    setActionLoading("pause");
    setFeedback(null);
    try {
      await pauseTrading(pauseReason || undefined);
      setFeedback({ type: "success", message: "Trading paused" });
      setShowPauseInput(false);
      setPauseReason("");
      await refresh();
    } catch (err) {
      setFeedback({ type: "error", message: err instanceof Error ? err.message : "Failed to pause trading" });
    } finally {
      setActionLoading(null);
    }
  }, [pauseReason, refresh]);

  const handleResume = useCallback(async () => {
    setActionLoading("resume");
    setFeedback(null);
    try {
      await resumeTrading();
      setFeedback({ type: "success", message: "Trading resumed" });
      setShowResumeModal(false);
      await refresh();
    } catch (err) {
      setFeedback({ type: "error", message: err instanceof Error ? err.message : "Failed to resume trading" });
    } finally {
      setActionLoading(null);
    }
  }, [refresh]);

  const isPaused = riskData?.isPaused ?? false;

  return (
    <section className="space-y-4" aria-label="Quick Actions">
      <h2 className="text-lg font-semibold text-white">Quick Actions</h2>

      {feedback && (
        <div
          className={`rounded-xl border p-4 text-sm ${
            feedback.type === "success"
              ? "border-[#00ff88]/30 bg-[#00ff88]/10 text-[#00ff88]"
              : "border-[#ff4757]/30 bg-[#ff4757]/10 text-[#ff4757]"
          }`}
          role="status"
        >
          <div className="flex items-center justify-between">
            <span>{feedback.message}</span>
            <button onClick={clearFeedback} className="text-gray-400 hover:text-white" aria-label="Dismiss">
              ×
            </button>
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="rounded-xl border border-white/10 bg-white/5 backdrop-blur-md p-5 space-y-4">
          <h3 className="text-sm font-medium text-gray-400">Emergency Controls</h3>

          <button
            type="button"
            onClick={() => setShowStopModal(true)}
            disabled={actionLoading !== null}
            className="w-full px-4 py-3 rounded-lg bg-[#ff4757] text-white font-bold hover:bg-[#ff4757]/80 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            Emergency Stop
          </button>

          <div className="space-y-2">
            <button
              type="button"
              onClick={() => {
                if (isPaused) {
                  // #10: Always show confirmation for resume
                  setShowResumeModal(true);
                } else {
                  // #9: Show confirmation modal for pause
                  setShowPauseModal(true);
                }
              }}
              disabled={actionLoading !== null}
              className={`w-full px-4 py-2 rounded-lg font-medium text-sm transition-colors disabled:opacity-40 disabled:cursor-not-allowed ${
                isPaused
                  ? "bg-[#00ff88] text-black hover:bg-[#00ff88]/80"
                  : "bg-yellow-500/10 text-yellow-400 border border-yellow-500/30 hover:bg-yellow-500/20"
              }`}
            >
              {isPaused
                ? actionLoading === "resume"
                  ? "Resuming..."
                  : "Resume Trading"
                : "Pause Trading"}
            </button>
          </div>
        </div>

        <div className="rounded-xl border border-white/10 bg-white/5 backdrop-blur-md p-5">
          <h3 className="text-sm font-medium text-gray-400 mb-4">Risk Parameters</h3>
          <RiskParamForm onSuccess={refresh} />
        </div>
      </div>

      <ConfirmationModal
        open={showStopModal}
        title="Emergency Stop"
        description="This will halt ALL trading and cancel open orders. Type STOP to confirm."
        confirmLabel="Stop Trading"
        confirmDisabled={stopConfirmText !== "STOP"}
        onConfirm={handleEmergencyStop}
        onCancel={() => { setShowStopModal(false); setStopConfirmText(""); }}
      >
        <input
          type="text"
          placeholder='Type "STOP" to confirm'
          value={stopConfirmText}
          onChange={(e) => setStopConfirmText(e.target.value)}
          className="w-full px-3 py-2 rounded-lg border border-[#ff4757]/30 bg-[#ff4757]/5 text-white font-mono text-sm placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-[#ff4757]/30"
          autoFocus
        />
      </ConfirmationModal>

      <ConfirmationModal
        open={showResumeModal}
        title="Resume After Circuit Breaker"
        description="The circuit breaker is tripped. Resuming trading may expose you to continued risk. Are you sure?"
        confirmLabel="Resume Trading"
        variant="warning"
        onConfirm={handleResume}
        onCancel={() => setShowResumeModal(false)}
      />

      {/* #9: Pause confirmation modal */}
      <ConfirmationModal
        open={showPauseModal}
        title="Pause Trading"
        description="This will halt all trading activity. Are you sure?"
        confirmLabel="Pause Trading"
        variant="warning"
        onConfirm={handlePause}
        onCancel={() => { setShowPauseModal(false); setPauseReason(""); }}
      >
        <input
          type="text"
          placeholder="Reason (optional)"
          value={pauseReason}
          onChange={(e) => setPauseReason(e.target.value)}
          className="w-full px-3 py-2 rounded-lg border border-yellow-500/30 bg-yellow-500/5 text-white text-sm placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-yellow-500/30"
        />
      </ConfirmationModal>
    </section>
  );
}
