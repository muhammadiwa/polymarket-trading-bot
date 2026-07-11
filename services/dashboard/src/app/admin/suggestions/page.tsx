"use client";

import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import type { Suggestion, OverfittingAnalysis } from "@/types";
import {
  fetchSuggestions,
  approveSuggestion,
  rejectSuggestion,
  startABTest,
  fetchOverfittingAnalysis,
  runAnalysis,
} from "@/lib/api";

export default function SuggestionsPage() {
  const [suggestions, setSuggestions] = useState<Suggestion[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [strategyFilter, setStrategyFilter] = useState<string>("");
  const [analyzing, setAnalyzing] = useState(false);
  const [selectedSuggestion, setSelectedSuggestion] = useState<string | null>(null);
  const [overfitting, setOverfitting] = useState<OverfittingAnalysis | null>(null);
  const [confirmReject, setConfirmReject] = useState<string | null>(null);
  const successTimerRef = useRef<NodeJS.Timeout | null>(null);

  // Auto-dismiss success message after 5 seconds
  useEffect(() => {
    if (success) {
      if (successTimerRef.current) {
        clearTimeout(successTimerRef.current);
      }
      successTimerRef.current = setTimeout(() => setSuccess(null), 5000);
    }
    return () => {
      if (successTimerRef.current) {
        clearTimeout(successTimerRef.current);
      }
    };
  }, [success]);

  const loadSuggestions = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await fetchSuggestions(
        strategyFilter || undefined,
        statusFilter || undefined,
        100
      );
      setSuggestions(data.suggestions);
      setTotal(data.total);
    } catch (err) {
      setError("Failed to load suggestions");
    } finally {
      setLoading(false);
    }
  }, [statusFilter, strategyFilter]);

  useEffect(() => {
    loadSuggestions();
  }, [loadSuggestions]);

  const handleApprove = async (id: string) => {
    try {
      setError(null);
      setSuccess(null);
      await approveSuggestion(id);
      setSuccess("Suggestion approved");
      loadSuggestions();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Failed to approve suggestion");
    }
  };

  const handleRejectClick = (id: string) => {
    setConfirmReject(id);
  };

  const handleRejectConfirm = async () => {
    if (!confirmReject) return;

    try {
      setError(null);
      setSuccess(null);
      await rejectSuggestion(confirmReject);
      setSuccess("Suggestion rejected");
      setConfirmReject(null);
      loadSuggestions();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Failed to reject suggestion");
    }
  };

  const handleStartABTest = async (id: string) => {
    try {
      setError(null);
      setSuccess(null);
      await startABTest(id);
      setSuccess("A/B test started");
      loadSuggestions();
    } catch (err: any) {
      setError(err.message || "Failed to start A/B test");
    }
  };

  const handleAnalyze = async () => {
    if (!strategyFilter) {
      setError("Please enter a strategy ID to analyze");
      return;
    }

    try {
      setAnalyzing(true);
      setError(null);
      setSuccess(null);
      const result = await runAnalysis(strategyFilter);
      setSuccess(`Analysis complete: ${result.patternsFound} patterns found, ${result.suggestionsGenerated} suggestions generated`);
      loadSuggestions();
    } catch (err: any) {
      setError(err.message || "Failed to run analysis");
    } finally {
      setAnalyzing(false);
    }
  };

  const handleViewOverfitting = async (id: string) => {
    try {
      setError(null);
      const data = await fetchOverfittingAnalysis(id);
      setOverfitting(data);
      setSelectedSuggestion(id);
    } catch (err: any) {
      setError(err.message || "Failed to load overfitting analysis");
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case "pending":
        return "bg-yellow-900 text-yellow-400";
      case "approved":
        return "bg-green-900 text-green-400";
      case "rejected":
        return "bg-red-900 text-red-400";
      default:
        return "bg-gray-800 text-gray-400";
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Link href="/admin" className="text-gray-400 hover:text-white">
          ← Admin
        </Link>
        <h2 className="text-xl font-semibold text-white">AI Optimizer Suggestions</h2>
      </div>

      {error && (
        <div className="rounded-md bg-red-900/50 p-4 text-red-200">{error}</div>
      )}
      {success && (
        <div className="rounded-md bg-green-900/50 p-4 text-green-200">{success}</div>
      )}

      {/* Filters & Actions */}
      <div className="flex flex-wrap items-end gap-4">
        <div>
          <label className="block text-sm text-gray-400">Strategy ID</label>
          <input
            type="text"
            value={strategyFilter}
            onChange={(e) => setStrategyFilter(e.target.value)}
            placeholder="e.g., simple-arb-v1"
            className="mt-1 rounded-md border border-gray-700 bg-gray-800 px-3 py-2 text-white"
          />
        </div>

        <div>
          <label className="block text-sm text-gray-400">Status</label>
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="mt-1 rounded-md border border-gray-700 bg-gray-800 px-3 py-2 text-white"
          >
            <option value="">All</option>
            <option value="pending">Pending</option>
            <option value="approved">Approved</option>
            <option value="rejected">Rejected</option>
          </select>
        </div>

        <button
          onClick={loadSuggestions}
          className="rounded-md bg-gray-800 px-4 py-2 text-sm text-gray-300 hover:bg-gray-700"
        >
          Refresh
        </button>

        <button
          onClick={handleAnalyze}
          disabled={analyzing || !strategyFilter}
          className="rounded-md bg-blue-600 px-4 py-2 text-sm text-white hover:bg-blue-700 disabled:opacity-50"
        >
          {analyzing ? "Analyzing..." : "Run Analysis"}
        </button>
      </div>

      {/* Suggestions Table */}
      <div className="rounded-lg border border-gray-800 bg-gray-900">
        <div className="p-4 border-b border-gray-800">
          <span className="text-sm text-gray-400">{total} suggestions found</span>
        </div>

        {loading ? (
          <div className="p-8 text-center text-gray-400">Loading...</div>
        ) : suggestions.length === 0 ? (
          <div className="p-8 text-center text-gray-400">No suggestions found</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-800 text-left text-gray-400">
                  <th className="p-3">Strategy</th>
                  <th className="p-3">Pattern</th>
                  <th className="p-3">Parameter</th>
                  <th className="p-3">Current</th>
                  <th className="p-3">Suggested</th>
                  <th className="p-3">Impact</th>
                  <th className="p-3">Confidence</th>
                  <th className="p-3">Status</th>
                  <th className="p-3">Actions</th>
                </tr>
              </thead>
              <tbody>
                {suggestions.map((s) => (
                  <tr key={s.id} className="border-b border-gray-800 hover:bg-gray-800/50">
                    <td className="p-3 text-white">{s.strategyId}</td>
                    <td className="p-3 text-gray-300">{s.patternType}</td>
                    <td className="p-3 text-gray-300">{s.parameterName}</td>
                    <td className="p-3 font-mono text-gray-400">{s.currentValue}</td>
                    <td className="p-3 font-mono text-green-400">{s.suggestedValue}</td>
                    <td className="p-3 text-gray-300">{s.expectedImpact}</td>
                    <td className="p-3 text-gray-300">{(s.confidence * 100).toFixed(1)}%</td>
                    <td className="p-3">
                      <span className={`rounded px-2 py-0.5 text-xs ${getStatusBadge(s.status)}`}>
                        {s.status}
                      </span>
                      {s.isOverfitting && (
                        <span className="ml-1 rounded bg-orange-900 px-2 py-0.5 text-xs text-orange-400">
                          overfitting
                        </span>
                      )}
                    </td>
                    <td className="p-3">
                      <div className="flex gap-1">
                        {s.status === "pending" && (
                          <>
                            <button
                              onClick={() => handleApprove(s.id)}
                              className="rounded bg-green-800 px-2 py-1 text-xs text-green-300 hover:bg-green-700"
                            >
                              Approve
                            </button>
                            <button
                              onClick={() => handleRejectClick(s.id)}
                              className="rounded bg-red-800 px-2 py-1 text-xs text-red-300 hover:bg-red-700"
                            >
                              Reject
                            </button>
                          </>
                        )}
                        {s.status === "approved" && (
                          <button
                            onClick={() => handleStartABTest(s.id)}
                            className="rounded bg-blue-800 px-2 py-1 text-xs text-blue-300 hover:bg-blue-700"
                          >
                            A/B Test
                          </button>
                        )}
                        <button
                          onClick={() => handleViewOverfitting(s.id)}
                          className="rounded bg-gray-700 px-2 py-1 text-xs text-gray-300 hover:bg-gray-600"
                        >
                          Overfitting
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Reject Confirmation Dialog */}
      {confirmReject && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="w-full max-w-md rounded-lg bg-gray-900 p-6">
            <h3 className="text-lg font-semibold text-white">Confirm Reject</h3>
            <p className="mt-2 text-gray-400">
              Are you sure you want to reject this suggestion? This action cannot be undone.
            </p>
            <div className="mt-6 flex justify-end gap-4">
              <button
                onClick={() => setConfirmReject(null)}
                className="rounded-md bg-gray-800 px-4 py-2 text-sm text-gray-300 hover:bg-gray-700"
              >
                Cancel
              </button>
              <button
                onClick={handleRejectConfirm}
                className="rounded-md bg-red-600 px-4 py-2 text-sm text-white hover:bg-red-700"
              >
                Reject
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Overfitting Analysis Modal */}
      {overfitting && selectedSuggestion && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="w-full max-w-lg rounded-lg bg-gray-900 p-6">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-white">Overfitting Analysis</h3>
              <button
                onClick={() => { setOverfitting(null); setSelectedSuggestion(null); }}
                className="text-gray-400 hover:text-white"
              >
                ✕
              </button>
            </div>

            <div className="space-y-3">
              <div className="flex justify-between">
                <span className="text-gray-400">Overfitting Score</span>
                <span className={`font-medium ${overfitting.isOverfitting ? 'text-red-400' : 'text-green-400'}`}>
                  {(overfitting.overfittingScore * 100).toFixed(1)}%
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-400">In-Sample Win Rate</span>
                <span className="text-white">{overfitting.inSampleWinRate}%</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-400">Out-of-Sample Win Rate</span>
                <span className="text-white">{overfitting.outOfSampleWinRate}%</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-400">Degradation</span>
                <span className={`font-medium ${overfitting.isOverfitting ? 'text-red-400' : 'text-green-400'}`}>
                  {overfitting.degradationPct}%
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-400">Is Overfitting</span>
                <span className={`font-medium ${overfitting.isOverfitting ? 'text-red-400' : 'text-green-400'}`}>
                  {overfitting.isOverfitting ? 'Yes' : 'No'}
                </span>
              </div>

              {overfitting.warning && (
                <div className="mt-4 rounded-md bg-yellow-900/50 p-3 text-yellow-200 text-sm">
                  ⚠️ {overfitting.warning}
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
