"use client";

import { Card } from "@/components/ui/Card";
import { usePortfolio } from "@/hooks/usePortfolio";
import Decimal from "decimal.js";

function formatCurrency(value: string, decimals = 2): string {
  try {
    const num = new Decimal(value);
    if (num.isNaN()) return "$0.00";
    const sign = num.isNeg() ? "-" : "";
    return `${sign}$${num.abs().toFixed(decimals)}`;
  } catch {
    return "$0.00";
  }
}

function pnlColor(value: string): string {
  try {
    const num = new Decimal(value);
    if (num.isNaN() || num.isZero()) return "text-gray-400";
    return num.isPos() ? "text-[#00ff88]" : "text-[#ff4757]";
  } catch {
    return "text-gray-400";
  }
}

function UtilizationBar({ rate }: { rate: string }) {
  let pct = 0;
  try {
    const parsed = new Decimal(rate).mul(100);
    if (!parsed.isNaN()) {
      pct = Math.min(100, Math.max(0, parsed.toNumber()));
    }
  } catch {
    pct = 0;
  }
  return (
    <div className="w-full">
      <div className="flex justify-between text-xs text-gray-400 mb-1">
        <span>Utilization</span>
        <span>{pct.toFixed(1)}%</span>
      </div>
      <div
        className="h-2 rounded-full bg-white/10 overflow-hidden"
        role="progressbar"
        aria-valuenow={pct}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-label={`Capital utilization: ${pct.toFixed(1)}%`}
      >
        <div
          className="h-full rounded-full bg-[#00d4ff] transition-all duration-500"
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  );
}

export function PortfolioOverview() {
  const { data, loading, error, wsStatus } = usePortfolio();

  if (loading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4" aria-busy="true" aria-label="Loading portfolio overview">
        {Array.from({ length: 4 }).map((_, i) => (
          <div
            key={i}
            className="rounded-xl border border-white/10 bg-white/5 backdrop-blur-md p-5 animate-pulse"
          >
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
        Failed to load portfolio data. Please try again.
      </div>
    );
  }

  if (!data) {
    return (
      <div className="rounded-xl border border-white/10 bg-white/5 backdrop-blur-md p-5 text-gray-400">
        No portfolio data available
      </div>
    );
  }

  return (
    <section className="space-y-4" aria-label="Portfolio Overview">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-white">Portfolio Overview</h2>
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
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card title="Total Capital">
          <p className="text-2xl font-bold font-mono text-white" aria-label={`Total capital: ${formatCurrency(data.totalCapital)}`}>
            {formatCurrency(data.totalCapital)}
          </p>
        </Card>
        <Card title="Daily PnL">
          <p className={`text-2xl font-bold font-mono ${pnlColor(data.dailyPnL)}`} aria-label={`Daily P&L: ${formatCurrency(data.dailyPnL)}`}>
            {formatCurrency(data.dailyPnL)}
          </p>
        </Card>
        <Card title="Total PnL">
          <p className={`text-2xl font-bold font-mono ${pnlColor(data.totalPnL)}`} aria-label={`Total P&L: ${formatCurrency(data.totalPnL)}`}>
            {formatCurrency(data.totalPnL)}
          </p>
        </Card>
        <Card title="Capital Utilization">
          <UtilizationBar rate={data.utilizationRate} />
        </Card>
      </div>
    </section>
  );
}
