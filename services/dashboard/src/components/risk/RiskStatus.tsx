"use client";

import { Card } from "@/components/ui/Card";
import { useRiskStatus } from "@/hooks/useRiskStatus";
import { formatCurrency } from "@/lib/format";
import Decimal from "decimal.js";

// #12: Decimal precision is set once in layout.tsx (shared entry point)

function drawdownColor(value: string): string {
  try {
    const pct = new Decimal(value).mul(100);
    if (pct.isNaN()) return "text-gray-400";
    if (pct.gte(8)) return "text-[#ff4757]";
    if (pct.gte(5)) return "text-yellow-500";
    return "text-[#00ff88]";
  } catch {
    return "text-gray-400";
  }
}

function drawdownBgColor(value: string): string {
  try {
    const pct = new Decimal(value).mul(100);
    if (pct.isNaN()) return "bg-gray-400/10";
    if (pct.gte(8)) return "bg-[#ff4757]/10";
    if (pct.gte(5)) return "bg-yellow-500/10";
    return "bg-[#00ff88]/10";
  } catch {
    return "bg-gray-400/10";
  }
}

function BudgetBar({ used, total }: { used: string; total: string }) {
  let pct = 0;
  try {
    const usedNum = new Decimal(used);
    const totalNum = new Decimal(total);
    if (!totalNum.isZero() && !usedNum.isNaN() && !totalNum.isNaN()) {
      pct = Math.min(100, Math.max(0, usedNum.div(totalNum).mul(100).toNumber()));
    }
  } catch {
    pct = 0;
  }

  const barColor = pct >= 80 ? "bg-[#ff4757]" : pct >= 50 ? "bg-yellow-500" : "bg-[#00d4ff]";

  return (
    <div className="w-full">
      <div className="h-2 rounded-full bg-white/10 overflow-hidden" role="progressbar" aria-valuenow={pct} aria-valuemin={0} aria-valuemax={100}>
        <div
          className={`h-full rounded-full transition-all duration-500 ${barColor}`}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  );
}

function CircuitBreakerBadge({ status, trippedAt }: { status: "open" | "closed"; trippedAt: string | null }) {
  // "open" = tripped (danger), "closed" = normal (safe)
  const isTripped = status === "open";
  return (
    <div className={`inline-flex items-center gap-2 px-3 py-1.5 rounded-lg ${isTripped ? "bg-[#ff4757]/10" : "bg-[#00ff88]/10"}`}>
      <span className={`h-2 w-2 rounded-full ${isTripped ? "bg-[#ff4757] animate-pulse" : "bg-[#00ff88]"}`} />
      <span className={`text-sm font-medium ${isTripped ? "text-[#ff4757]" : "text-[#00ff88]"}`}>
        {isTripped ? "Tripped" : "Normal"}
      </span>
      {isTripped && trippedAt && (
        <span className="text-xs text-gray-400">
          Tripped at {new Date(trippedAt).toLocaleTimeString()}
        </span>
      )}
    </div>
  );
}

export function RiskStatus() {
  const { data, loading, error, wsStatus } = useRiskStatus();

  if (loading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4" aria-busy="true" aria-label="Loading risk status">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="rounded-xl border border-white/10 bg-white/5 backdrop-blur-md p-5 animate-pulse">
            <div className="h-4 w-24 bg-white/10 rounded mb-3" />
            <div className="h-8 w-32 bg-white/10 rounded" />
          </div>
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-xl border border-[#ff4757]/30 bg-[#ff4757]/10 backdrop-blur-md p-5 text-[#ff4757]" role="alert">
        Failed to load risk status. Please try again.
      </div>
    );
  }

  if (!data) {
    return (
      <div className="rounded-xl border border-white/10 bg-white/5 backdrop-blur-md p-5 text-gray-400">
        No risk data available
      </div>
    );
  }

  const drawdownPct = (() => {
    try {
      return `${new Decimal(data.currentDrawdown).mul(100).toFixed(2)}%`;
    } catch {
      return "0.00%";
    }
  })();

  return (
    <section className="space-y-4" aria-label="Risk Status">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-white">Risk Status</h2>
        <span
          className={`text-xs px-2 py-1 rounded-full ${
            wsStatus === "connected"
              ? "bg-[#00ff88]/10 text-[#00ff88]"
              : wsStatus === "connecting"
                ? "bg-yellow-500/10 text-yellow-500"
                : "bg-[#ff4757]/10 text-[#ff4757]"
          }`}
          role="status"
          aria-label={`WebSocket status: ${wsStatus}`}
        >
          {wsStatus === "connected" ? "Live" : wsStatus === "connecting" ? "Connecting..." : "Offline"}
        </span>
      </div>

      {data.isPaused && (
        <div className="rounded-xl border border-yellow-500/30 bg-yellow-500/10 backdrop-blur-md p-4 text-yellow-400" role="alert">
          <span className="font-medium">Trading Paused:</span> {data.pausedReason ?? "Manual pause"}
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card title="Daily Budget Remaining">
          <p className="text-2xl font-bold font-mono text-white" aria-label={`Daily budget remaining: ${formatCurrency(data.dailyBudgetRemaining)}`}>
            {formatCurrency(data.dailyBudgetRemaining)}
          </p>
          <p className="text-xs text-gray-400 mt-1">
            of {formatCurrency(data.dailyBudgetTotal)}
          </p>
          <div className="mt-3">
            <BudgetBar used={data.dailyBudgetUsedFraction} total="1" />
          </div>
          <p className="text-xs text-gray-400 mt-1">
            {(() => {
              try {
                return `${new Decimal(data.dailyBudgetUsedFraction).mul(100).toFixed(1)}% used`;
              } catch {
                return "0.0% used";
              }
            })()}
          </p>
        </Card>

        <Card title="Current Drawdown">
          <p className={`text-2xl font-bold font-mono ${drawdownColor(data.currentDrawdown)}`} aria-label={`Current drawdown: ${drawdownPct}`}>
            {drawdownPct}
          </p>
          <div className={`inline-block mt-2 px-2 py-0.5 rounded text-xs ${drawdownBgColor(data.currentDrawdown)} ${drawdownColor(data.currentDrawdown)}`}>
            Threshold: {(() => {
              try {
                return `${new Decimal(data.drawdownThreshold).mul(100).toFixed(2)}%`;
              } catch {
                return "0.00%";
              }
            })()}
          </div>
        </Card>

        <Card title="Win Streak">
          <p className="text-2xl font-bold font-mono text-white" aria-label={`Win streak: ${data.winStreakCurrent} of ${data.winStreakThreshold}`}>
            {data.winStreakCurrent} <span className="text-gray-400 text-lg">/ {data.winStreakThreshold}</span>
          </p>
          <div className="w-full mt-3">
            <div className="h-2 rounded-full bg-white/10 overflow-hidden">
              <div
                className="h-full rounded-full bg-[#00ff88] transition-all duration-500"
                style={{ width: `${Math.min(100, (data.winStreakCurrent / Math.max(1, data.winStreakThreshold)) * 100)}%` }}
              />
            </div>
          </div>
          {data.winStreakCurrent >= data.winStreakThreshold && (
            <p className="text-xs text-yellow-400 mt-2">Threshold reached — may pause trading</p>
          )}
        </Card>

        <Card title="Circuit Breaker">
          <CircuitBreakerBadge status={data.circuitBreakerStatus} trippedAt={data.circuitBreakerTrippedAt} />
        </Card>
      </div>
    </section>
  );
}
