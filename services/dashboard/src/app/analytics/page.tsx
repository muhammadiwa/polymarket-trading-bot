"use client";

import { useState } from "react";
import { AppShell } from "@/components/layout/AppShell";
import { useAnalytics } from "@/hooks/useAnalytics";
import { PnLLineChart } from "@/components/charts/PnLLineChart";
import { PnLHistogram } from "@/components/charts/PnLHistogram";
import { StrategyPieChart } from "@/components/charts/StrategyPieChart";
import { downloadCSV, downloadJSON } from "@/lib/api";

export default function AnalyticsPage() {
  const [startDate, setStartDate] = useState(() => {
    const d = new Date();
    d.setMonth(d.getMonth() - 1);
    return d.toISOString().split("T")[0];
  });
  const [endDate, setEndDate] = useState(() => new Date().toISOString().split("T")[0]);
  const [side, setSide] = useState<string>("all");
  const [pnlSign, setPnlSign] = useState<string>("all");
  const [strategyId, setStrategyId] = useState<string>("");
  const [marketId, setMarketId] = useState<string>("");
  const [exporting, setExporting] = useState(false);

  const { pnlData, histogramData, loading, error } = useAnalytics(startDate, endDate);

  const handleExportCSV = async () => {
    setExporting(true);
    try {
      await downloadCSV(startDate, endDate, side === "all" ? undefined : side, pnlSign === "all" ? undefined : pnlSign, strategyId || undefined, marketId || undefined);
    } catch (err) {
      console.error("Export failed:", err);
    } finally {
      setExporting(false);
    }
  };

  const handleExportJSON = async () => {
    setExporting(true);
    try {
      await downloadJSON(startDate, endDate, side === "all" ? undefined : side, pnlSign === "all" ? undefined : pnlSign, strategyId || undefined, marketId || undefined);
    } catch (err) {
      console.error("Export failed:", err);
    } finally {
      setExporting(false);
    }
  };

  return (
    <AppShell>
      <div className="space-y-6 p-6">
        <div className="flex items-center justify-between flex-wrap gap-3">
          <h1 className="text-2xl font-bold text-white">Analytics</h1>
        <div className="flex items-center gap-2 flex-wrap">
          <input
            type="date"
            value={startDate}
            onChange={(e) => setStartDate(e.target.value)}
            className="px-3 py-2 rounded-lg border border-white/10 bg-white/5 text-white text-sm"
          />
          <span className="text-gray-400">to</span>
          <input
            type="date"
            value={endDate}
            onChange={(e) => setEndDate(e.target.value)}
            className="px-3 py-2 rounded-lg border border-white/10 bg-white/5 text-white text-sm"
          />
          <select
            value={side}
            onChange={(e) => setSide(e.target.value)}
            className="px-3 py-2 rounded-lg border border-white/10 bg-white/5 text-white text-sm"
          >
            <option value="all">All Sides</option>
            <option value="YES">YES</option>
            <option value="NO">NO</option>
          </select>
          <select
            value={pnlSign}
            onChange={(e) => setPnlSign(e.target.value)}
            className="px-3 py-2 rounded-lg border border-white/10 bg-white/5 text-white text-sm"
          >
            <option value="all">All PnL</option>
            <option value="positive">Winning</option>
            <option value="negative">Losing</option>
            <option value="zero">Break-even</option>
          </select>
          <input
            type="text"
            placeholder="Strategy ID"
            value={strategyId}
            onChange={(e) => setStrategyId(e.target.value)}
            className="px-3 py-2 rounded-lg border border-white/10 bg-white/5 text-white text-sm w-32"
          />
          <input
            type="text"
            placeholder="Market ID"
            value={marketId}
            onChange={(e) => setMarketId(e.target.value)}
            className="px-3 py-2 rounded-lg border border-white/10 bg-white/5 text-white text-sm w-32"
          />
          <button
            onClick={handleExportCSV}
            disabled={exporting}
            className="px-4 py-2 rounded-lg bg-[#00d4ff] text-black font-medium text-sm hover:bg-[#00d4ff]/80 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {exporting ? "Exporting..." : "CSV"}
          </button>
          <button
            onClick={handleExportJSON}
            disabled={exporting}
            className="px-4 py-2 rounded-lg bg-white/10 text-white font-medium text-sm hover:bg-white/20 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            JSON
          </button>
        </div>
      </div>

      {error && (
        <div className="rounded-xl border border-[#ff4757]/30 bg-[#ff4757]/10 backdrop-blur-md p-5 text-[#ff4757]" role="alert">
          {error}
        </div>
      )}

      <div className="grid grid-cols-1 gap-6">
        <PnLLineChart data={pnlData?.by_period?.map(p => ({ ...p, pnl: parseFloat(p.pnl), trade_count: p.trade_count })) ?? []} loading={loading} />

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <PnLHistogram pnls={histogramData?.pnls ?? []} bins={histogramData?.bins ?? 20} loading={loading} />
          <StrategyPieChart data={pnlData?.by_strategy ?? []} loading={loading} />
        </div>
      </div>
      </div>
    </AppShell>
  );
}
