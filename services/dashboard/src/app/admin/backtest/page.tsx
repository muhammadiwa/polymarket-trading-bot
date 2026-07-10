"use client";

import { useCallback, useState } from "react";
import type { BacktestRequest, BacktestResults, BacktestStatus, SimulationConfig } from "@/types";
import { startBacktest, fetchBacktestStatus, fetchBacktestResults } from "@/lib/api";

const DEFAULT_SIMULATION: SimulationConfig = {
  slippagePct: 0.01,
  partialFillProbability: 0.1,
  latencyMs: 100,
  minFillRatio: 0.5,
  rngSeed: 42,
};

export default function BacktestPage() {
  const [strategyId, setStrategyId] = useState("");
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");
  const [simulation, setSimulation] = useState<SimulationConfig>(DEFAULT_SIMULATION);
  const [running, setRunning] = useState(false);
  const [status, setStatus] = useState<BacktestStatus | null>(null);
  const [results, setResults] = useState<BacktestResults | null>(null);
  const [error, setError] = useState<string | null>(null);

  const pollStatus = useCallback(async (runId: string) => {
    const poll = async () => {
      try {
        const s = await fetchBacktestStatus(runId);
        setStatus(s);

        if (s.status === "completed") {
          const r = await fetchBacktestResults(runId);
          setResults(r);
          setRunning(false);
          return;
        }

        if (s.status === "failed") {
          setError(s.errorMessage || "Backtest failed");
          setRunning(false);
          return;
        }

        // Continue polling
        setTimeout(poll, 1000);
      } catch (err) {
        setError("Failed to fetch status");
        setRunning(false);
      }
    };
    poll();
  }, []);

  const handleRun = async () => {
    if (!strategyId || !startDate || !endDate) {
      setError("Please fill all required fields");
      return;
    }

    setRunning(true);
    setError(null);
    setResults(null);
    setStatus(null);

    try {
      const request: BacktestRequest = {
        strategyId,
        startDate,
        endDate,
        simulation,
      };

      const result = await startBacktest(request);
      setStatus(result);
      pollStatus(result.runId);
    } catch (err: any) {
      setError(err.message || "Failed to start backtest");
      setRunning(false);
    }
  };

  return (
    <div className="space-y-6">
      <h2 className="text-xl font-semibold text-white">Backtesting</h2>

      {error && (
        <div className="rounded-md bg-red-900/50 p-4 text-red-200">{error}</div>
      )}

      {/* Run Form */}
      <div className="rounded-lg border border-gray-800 bg-gray-900 p-6">
        <h3 className="mb-4 text-lg font-semibold text-white">New Backtest</h3>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
          <div>
            <label className="block text-sm text-gray-400">Strategy ID</label>
            <input
              type="text"
              value={strategyId}
              onChange={(e) => setStrategyId(e.target.value)}
              placeholder="e.g., simple-arb-v1"
              className="mt-1 w-full rounded-md border border-gray-700 bg-gray-800 px-3 py-2 text-white"
            />
          </div>

          <div>
            <label className="block text-sm text-gray-400">Start Date</label>
            <input
              type="date"
              value={startDate}
              onChange={(e) => setStartDate(e.target.value)}
              className="mt-1 w-full rounded-md border border-gray-700 bg-gray-800 px-3 py-2 text-white"
            />
          </div>

          <div>
            <label className="block text-sm text-gray-400">End Date</label>
            <input
              type="date"
              value={endDate}
              onChange={(e) => setEndDate(e.target.value)}
              className="mt-1 w-full rounded-md border border-gray-700 bg-gray-800 px-3 py-2 text-white"
            />
          </div>
        </div>

        {/* Simulation Config */}
        <div className="mt-4">
          <h4 className="mb-2 text-sm font-medium text-gray-400">Simulation Config</h4>
          <div className="grid grid-cols-2 gap-4 md:grid-cols-5">
            <div>
              <label className="block text-xs text-gray-500">Slippage %</label>
              <input
                type="number"
                value={simulation.slippagePct}
                onChange={(e) => setSimulation({ ...simulation, slippagePct: parseFloat(e.target.value) })}
                step="0.001"
                min="0"
                max="0.1"
                className="mt-1 w-full rounded border border-gray-700 bg-gray-800 px-2 py-1 text-sm text-white"
              />
            </div>
            <div>
              <label className="block text-xs text-gray-500">Fill Probability</label>
              <input
                type="number"
                value={simulation.partialFillProbability}
                onChange={(e) => setSimulation({ ...simulation, partialFillProbability: parseFloat(e.target.value) })}
                step="0.1"
                min="0"
                max="1"
                className="mt-1 w-full rounded border border-gray-700 bg-gray-800 px-2 py-1 text-sm text-white"
              />
            </div>
            <div>
              <label className="block text-xs text-gray-500">Latency (ms)</label>
              <input
                type="number"
                value={simulation.latencyMs}
                onChange={(e) => setSimulation({ ...simulation, latencyMs: parseInt(e.target.value) })}
                min="0"
                max="10000"
                className="mt-1 w-full rounded border border-gray-700 bg-gray-800 px-2 py-1 text-sm text-white"
              />
            </div>
            <div>
              <label className="block text-xs text-gray-500">Min Fill Ratio</label>
              <input
                type="number"
                value={simulation.minFillRatio}
                onChange={(e) => setSimulation({ ...simulation, minFillRatio: parseFloat(e.target.value) })}
                step="0.1"
                min="0"
                max="1"
                className="mt-1 w-full rounded border border-gray-700 bg-gray-800 px-2 py-1 text-sm text-white"
              />
            </div>
            <div>
              <label className="block text-xs text-gray-500">RNG Seed</label>
              <input
                type="number"
                value={simulation.rngSeed}
                onChange={(e) => setSimulation({ ...simulation, rngSeed: parseInt(e.target.value) })}
                className="mt-1 w-full rounded border border-gray-700 bg-gray-800 px-2 py-1 text-sm text-white"
              />
            </div>
          </div>
        </div>

        <button
          onClick={handleRun}
          disabled={running}
          className="mt-4 rounded-md bg-blue-600 px-6 py-2 text-white hover:bg-blue-700 disabled:opacity-50"
        >
          {running ? "Running..." : "Run Backtest"}
        </button>
      </div>

      {/* Status */}
      {status && (
        <div className="rounded-lg border border-gray-800 bg-gray-900 p-6">
          <h3 className="mb-4 text-lg font-semibold text-white">Status</h3>
          <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
            <div>
              <span className="text-sm text-gray-400">Run ID</span>
              <p className="font-mono text-white">{status.runId}</p>
            </div>
            <div>
              <span className="text-sm text-gray-400">Status</span>
              <p className={`font-medium ${
                status.status === "completed" ? "text-green-400" :
                status.status === "failed" ? "text-red-400" :
                "text-yellow-400"
              }`}>
                {status.status.toUpperCase()}
              </p>
            </div>
            <div>
              <span className="text-sm text-gray-400">Progress</span>
              <p className="text-white">{status.progress || "-"}</p>
            </div>
            <div>
              <span className="text-sm text-gray-400">Started</span>
              <p className="text-white">{status.startedAt ? new Date(status.startedAt).toLocaleTimeString() : "-"}</p>
            </div>
          </div>
        </div>
      )}

      {/* Results */}
      {results && (
        <div className="rounded-lg border border-gray-800 bg-gray-900 p-6">
          <h3 className="mb-4 text-lg font-semibold text-white">Results</h3>

          {/* Summary */}
          <div className="mb-6 grid grid-cols-2 gap-4 md:grid-cols-4">
            <div className="rounded-md bg-gray-800 p-3">
              <span className="text-sm text-gray-400">Total PnL</span>
              <p className={`text-xl font-bold ${
                parseFloat(results.summary.totalPnl) >= 0 ? "text-green-400" : "text-red-400"
              }`}>
                ${results.summary.totalPnl}
              </p>
            </div>
            <div className="rounded-md bg-gray-800 p-3">
              <span className="text-sm text-gray-400">Win Rate</span>
              <p className="text-xl font-bold text-white">{results.summary.winRate}%</p>
            </div>
            <div className="rounded-md bg-gray-800 p-3">
              <span className="text-sm text-gray-400">Sharpe Ratio</span>
              <p className="text-xl font-bold text-white">{results.summary.sharpeRatio}</p>
            </div>
            <div className="rounded-md bg-gray-800 p-3">
              <span className="text-sm text-gray-400">Max Drawdown</span>
              <p className="text-xl font-bold text-red-400">{results.summary.maxDrawdown}%</p>
            </div>
          </div>

          {/* Trades Table */}
          <h4 className="mb-2 text-sm font-medium text-gray-400">Trades ({results.trades.length})</h4>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-800 text-left text-gray-400">
                  <th className="pb-2">Time</th>
                  <th className="pb-2">Market</th>
                  <th className="pb-2">Side</th>
                  <th className="pb-2">Price</th>
                  <th className="pb-2">Qty</th>
                  <th className="pb-2">PnL</th>
                </tr>
              </thead>
              <tbody>
                {results.trades.slice(0, 50).map((trade, i) => (
                  <tr key={i} className="border-b border-gray-800">
                    <td className="py-2 text-gray-400">{new Date(trade.timestamp).toLocaleTimeString()}</td>
                    <td className="py-2 text-white">{trade.marketId}</td>
                    <td className="py-2">
                      <span className={`rounded px-1 text-xs ${
                        trade.side === "YES" ? "bg-green-900 text-green-400" : "bg-red-900 text-red-400"
                      }`}>
                        {trade.side}
                      </span>
                    </td>
                    <td className="py-2 text-white">${trade.price}</td>
                    <td className="py-2 text-white">{trade.quantity}</td>
                    <td className={`py-2 ${parseFloat(trade.pnl) >= 0 ? "text-green-400" : "text-red-400"}`}>
                      ${trade.pnl}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
